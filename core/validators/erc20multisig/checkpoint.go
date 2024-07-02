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
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	events "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

func (t *Topology) Name() types.CheckpointName {
	return types.MultisigControlCheckpoint
}

func (t *Topology) Checkpoint() ([]byte, error) {
	var thresholdSet *events.ERC20MultiSigThresholdSetEvent
	if t.threshold != nil {
		thresholdSet = t.threshold.IntoProto()
	}

	msg := &checkpoint.MultisigControl{
		Signers:       t.getSigners(),
		ThresholdSet:  thresholdSet,
		LastBlockSeen: t.getLastBlockSeen(),
	}
	ret, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (t *Topology) Load(ctx context.Context, data []byte) error {
	mc := checkpoint.MultisigControl{}
	if err := proto.Unmarshal(data, &mc); err != nil {
		return err
	}

	// load signers
	for _, sevt := range mc.Signers {
		signerEvent := types.SignerEventFromEventProto(sevt)
		t.addSignerEvent(ctx, signerEvent)

		// now ensure that the seen map is OK
		if !t.ensureNotDuplicate(signerEvent.Hash()) {
			t.log.Panic("invalid checkpoint data, duplicated signer event",
				logging.String("event-id", signerEvent.ID))
		}
	}

	// load threshold
	if mc.ThresholdSet != nil {
		t.setThresholdSetEvent(
			ctx, types.SignerThresholdSetEventFromEventProto(mc.ThresholdSet))
	}

	// 0 is default value, we assume that it was then not set
	//if mc.LastBlockSeen != 0 {
	//	t.ethEventSource.UpdateMultisigControlStartingBlock(mc.LastBlockSeen)
	//}

	return nil
}

func (t *Topology) getSigners() []*events.ERC20MultiSigSignerEvent {
	// we only keep the list of all verified events
	out := []*events.ERC20MultiSigSignerEvent{}
	for _, evts := range t.eventsPerAddress {
		for _, evt := range evts {
			out = append(out, evt.IntoProto())
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Id < out[j].Id
	})

	return out
}

func (t *Topology) getLastBlockSeen() uint64 {
	var block uint64

	for _, evts := range t.eventsPerAddress {
		for _, evt := range evts {
			if evt.BlockNumber > block {
				block = evt.BlockNumber
			}
		}
	}

	return block
}
