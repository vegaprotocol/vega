package governance

import (
	"context"

	"code.vegaprotocol.io/protos/vega"
	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"

	"code.vegaprotocol.io/vega/libs/proto"
)

func (e *Engine) Name() types.CheckpointName {
	return types.GovernanceCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	if len(e.enactedProposals) == 0 {
		return nil, nil
	}
	snap := &checkpoint.Proposals{
		Proposals: e.getCheckpointProposals(),
	}
	return proto.Marshal(snap)
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	snap := &checkpoint.Proposals{}
	if err := proto.Unmarshal(data, snap); err != nil {
		return err
	}

	e.activeProposals = make([]*proposal, 0, len(snap.Proposals))
	evts := make([]events.Event, 0, len(snap.Proposals))
	for _, p := range snap.Proposals {
		if p.Terms.ClosingTimestamp < e.currentTime.Unix() {
			// the proposal in question has expired, ignore it
			continue
		}
		prop := types.ProposalFromProto(p)
		evts = append(evts, events.NewProposalEvent(ctx, *prop))
		e.activeProposals = append(e.activeProposals, &proposal{
			Proposal: prop,
		})
	}
	// sned events for restored proposals
	e.broker.SendBatch(evts)
	// @TODO ensure OnChainTimeUpdate is called
	return nil
}

func (e *Engine) getCheckpointProposals() []*vega.Proposal {
	ret := make([]*vega.Proposal, 0, len(e.enactedProposals))
	for _, p := range e.enactedProposals {
		ret = append(ret, p.IntoProto())
	}
	return ret
}
