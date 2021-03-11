package stubs

import types "code.vegaprotocol.io/vega/proto"

type ProposalStub struct {
	data []types.Proposal
}

func NewProposalStub() *ProposalStub {
	return &ProposalStub{
		data: []types.Proposal{},
	}
}

func (p *ProposalStub) Add(v types.Proposal) {
	p.data = append(p.data, v)
}

func (p *ProposalStub) Flush() {}