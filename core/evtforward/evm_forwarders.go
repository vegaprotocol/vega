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

package evtforward

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

// EVMForwarders a little wrapper around the second bridge forwarder as a first step to generalising to
// multiple EVM bridges.
type EVMForwarders struct {
	forwarders []*Forwarder
}

// New creates a new instance of the event forwarder.
func NewEVMForwarders(
	secondBridge *Forwarder,
) *EVMForwarders {
	return &EVMForwarders{
		forwarders: []*Forwarder{secondBridge},
	}
}

func (f *EVMForwarders) Namespace() types.SnapshotNamespace {
	return types.EVMEventForwardersSnapshot
}

func (f *EVMForwarders) Keys() []string {
	return []string{(&types.PayloadEventForwarder{}).Key()}
}

func (f *EVMForwarders) Stopped() bool {
	return false
}

func (f *EVMForwarders) serialise() ([]byte, error) {
	slice := make([]*snapshotpb.EventForwarderBucket, 0, f.forwarders[0].ackedEvts.Size())
	iter := f.forwarders[0].ackedEvts.events.Iterator()
	for iter.Next() {
		v := iter.Value().(*ackedEvtBucket)
		slice = append(slice, &snapshotpb.EventForwarderBucket{
			Ts:     v.ts,
			Hashes: v.hashes,
		})
	}

	payload := types.Payload{
		Data: &types.PayloadEVMEventForwarders{
			EVMEventForwarders: []*snapshotpb.EventForwarder{
				{
					Buckets: slice,
					ChainId: f.forwarders[0].chainID,
				},
			},
		},
	}
	return proto.Marshal(payload.IntoProto())
}

// get the serialised form of the given key.
func (f *EVMForwarders) getSerialised(k string) (data []byte, err error) {
	if k != key {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	return f.serialise()
}

func (f *EVMForwarders) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := f.getSerialised(k)
	return state, nil, err
}

func (f *EVMForwarders) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if f.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	if pl, ok := p.Data.(*types.PayloadEVMEventForwarders); ok {
		f.restore(ctx, pl)
		return nil, nil
	}

	return nil, types.ErrUnknownSnapshotType
}

func (f *EVMForwarders) restore(ctx context.Context, p *types.PayloadEVMEventForwarders) {
	f.forwarders[0].restore(ctx, &types.PayloadEventForwarder{
		Buckets: p.EVMEventForwarders[0].Buckets,
	})
}
