package buffer

import "github.com/tendermint/tendermint/types"

type Proposal struct {
	buf []types.Proposal
}

func NewProposal() *Proposal {
	return &Proposal{
		buf: []types.Proposal{},
	}
}

func (o *Proposal) Add(ord types.Proposal) {
	// noop
}

func (o *Proposal) Flush() error {
	return nil
}
