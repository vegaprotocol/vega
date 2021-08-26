package governance

import (
	"code.vegaprotocol.io/protos/vega"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/proto"
)

func (e *Engine) Name() types.CheckpointName {
	return types.GovernanceCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	if len(e.enactedProposals) == 0 {
		return nil, nil
	}
	snap := &snapshot.Proposals{
		Proposals: e.getSnapshotProposals(),
	}
	return proto.Marshal(snap)
}

func (e *Engine) Load(data []byte) error {
	snap := &snapshot.Proposals{}
	if err := proto.Unmarshal(data, snap); err != nil {
		return err
	}
	// just make sure the time is set
	if e.currentTime.IsZero() {
		e.currentTime = vegatime.Now()
	}

	e.activeProposals = make([]*proposal, 0, len(snap.Proposals))
	for _, p := range snap.Proposals {
		if p.Terms.ClosingTimestamp < e.currentTime.Unix() {
			// the proposal in question has expired, ignore it
			continue
		}
		e.activeProposals = append(e.activeProposals, &proposal{
			Proposal: types.ProposalFromProto(p),
		})
	}
	// @TODO ensure OnChainTimeUpdate is called
	return nil
}

func (e *Engine) getSnapshotProposals() []*vega.Proposal {
	ret := make([]*vega.Proposal, 0, len(e.enactedProposals))
	for _, p := range e.enactedProposals {
		ret = append(ret, p.IntoProto())
	}
	return ret
}
