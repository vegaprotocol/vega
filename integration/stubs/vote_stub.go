package stubs

import (
	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

func (b *BrokerStub) Reset() {
	b.mu.Lock()
	b.data = map[events.Type][]events.Event{}
	b.mu.Unlock()
}

type VoteStub struct {
	data []types.Vote
}

func NewVoteStub() *VoteStub {
	return &VoteStub{
		data: []types.Vote{},
	}
}

func (v *VoteStub) Add(vote types.Vote) {
	v.data = append(v.data, vote)
}

func (v *VoteStub) Flush() {}