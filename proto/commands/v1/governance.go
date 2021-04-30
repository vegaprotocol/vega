package v1

import types "code.vegaprotocol.io/vega/proto"

func ProposalSubmissionFromProposal(p *types.Proposal) *ProposalSubmission {
	return &ProposalSubmission{
		Reference: p.Reference,
		Terms:     p.Terms,
	}
}
