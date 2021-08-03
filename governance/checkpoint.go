package governance

import (
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/vegatime"
)

func (e *Engine) Name() types.CheckpointName {
	return types.GovernanceCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	if len(e.enactedProposals) == 0 {
		return nil, nil
	}
	snap := &vega.Proposals{
		Proposals: e.getSnapshotProposals(),
	}
	return vega.Marshal(snap)
}

func (e *Engine) Load(data []byte) error {
	snap := &vega.Proposals{}
	if err := vega.Unmarshal(data, snap); err != nil {
		return err
	}
	// just make sure the time is set
	if e.currentTime.IsZero() {
		e.currentTime = vegatime.Now()
	}

	for _, p := range snap.Proposals {
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
