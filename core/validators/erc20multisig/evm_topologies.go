// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package erc20multisig

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

// Topology keeps track of all the validators
// registered in the erc20 bridge.
type EVMTopologies struct {
	topologies []*Topology
}

func NewEVMTopologies(
	secondBridge *Topology,
) *EVMTopologies {
	return &EVMTopologies{
		topologies: []*Topology{secondBridge},
	}
}

func (t *EVMTopologies) Namespace() types.SnapshotNamespace {
	return types.EVMMultiSigTopologiesSnapshot
}

func (t *EVMTopologies) Keys() []string {
	return []string{
		(&types.PayloadEventForwarder{}).Key(),
	}
}

func (t *EVMTopologies) GetState(k string) ([]byte, []types.StateProvider, error) {
	if k != (&types.PayloadEventForwarder{}).Key() {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}
	data, err := t.serialise()
	return data, nil, err
}

func (t *EVMTopologies) Stopped() bool {
	return false
}

func (t *EVMTopologies) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if t.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadEVMMultisigTopologies:
		return nil, t.restore(ctx, pl.Topologies)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

// get the serialised form of the given key.
func (t *EVMTopologies) restore(ctx context.Context, topologies []*snapshotpb.EVMMultisigTopology) error {
	secondBridge := t.topologies[0]
	if err := secondBridge.restorePendingState(ctx, topologies[0].Pending); err != nil {
		return err
	}
	if err := secondBridge.restoreVerifiedState(ctx, topologies[0].Verified); err != nil {
		return err
	}
	return nil
}

// get the serialised form of the given key.
func (t *EVMTopologies) serialise() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadEVMMultisigTopologies{
			Topologies: []*snapshotpb.EVMMultisigTopology{
				{
					Verified: t.topologies[0].constructVerifiedState(),
					Pending:  t.topologies[0].constructPendingState(),
					ChainId:  t.topologies[0].chainID,
				},
			},
		},
	}
	return proto.Marshal(payload.IntoProto())
}

func (t *EVMTopologies) OnStateLoaded(ctx context.Context) error {
	// tell the internal EEF where it got up to so we do not resend events we're already seen

	topology := t.topologies[0]
	lastSeen := topology.getLastBlockSeen()
	if lastSeen != 0 {
		// TODO snapshot migration stuff
		topology.log.Info("restoring multisig starting block", logging.Uint64("block", lastSeen), logging.String("chain-id", topology.chainID))
		// topology.ethEventSource.UpdateMultisigControlStartingBlock(topology.getLastBlockSeen())
	}
	return nil
}
