package erc20multisig

import (
	"context"
	"sort"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	events "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
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
	if mc.LastBlockSeen != 0 {
		t.ethEventSource.UpdateMultisigControlStartingBlock(mc.LastBlockSeen)
	}

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

	for _, v := range t.pendingSigners {
		if block == 0 {
			block = v.BlockNumber
			continue
		}

		if block > v.BlockNumber {
			block = v.BlockNumber
		}
	}

	for _, v := range t.pendingThresholds {
		if block == 0 {
			block = v.BlockNumber
			continue
		}

		if block > v.BlockNumber {
			block = v.BlockNumber
		}
	}

	// now if we have got any pending one, let's just pick the
	// most recent verified
	if block == 0 {
		for _, evts := range t.eventsPerAddress {
			for _, evt := range evts {
				if evt.BlockNumber > block {
					block = evt.BlockNumber
				}
			}
		}
	}

	return block
}
