package v1

import (
	"errors"

	types "code.vegaprotocol.io/vega/protos/vega"
)

func ProposalSubmissionFromProposal(p *types.Proposal) (*ProposalSubmission, error) {
	terms := p.GetTerms()
	if terms == nil {
		return nil, errors.New("can not proposal submission from batch proposal")
	}

	return &ProposalSubmission{
		Reference: p.Reference,
		Terms:     terms,
	}, nil
}
