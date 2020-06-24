package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type VoteSub struct {
	*Base
	mu      sync.Mutex
	all     []types.Vote
	filters []VoteFilter
	stream  bool
}

// VoteByPartyID filters votes cast by given party
func VoteByPartyID(id string) VoteFilter {
	return func(v types.Vote) bool {
		if v.PartyID == id {
			return true
		}
		return false
	}
}

func VoteByProposalID(id string) VoteFilter {
	return func(v types.Vote) bool {
		if v.ProposalID == id {
			return true
		}
		return false
	}
}

func NewVoteSub(ctx context.Context, stream bool, filters ...VoteFilter) *VoteSub {
	v := &VoteSub{
		Base:    newBase(ctx, 10),
		all:     []types.Vote{},
		filters: filters,
		stream:  stream,
	}
	v.running = true
	go v.loop(v.ctx)
	return v
}

func (v *VoteSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			v.Halt()
			return
		case e := <-v.ch:
			if v.isRunning() {
				v.Push(e)
			}
		}
	}
}

func (v *VoteSub) Push(e events.Event) {
	te, ok := e.(VoteE)
	if !ok {
		return
	}
	vote := te.Vote()
	for _, f := range v.filters {
		if !f(vote) {
			return
		}
	}
	v.mu.Lock()
	v.all = append(v.all, vote)
	v.mu.Unlock()
}

// GetData - either returns the full data-set, or just updates, depending on configuration
func (v *VoteSub) GetData() []types.Vote {
	if v.stream {
		return v.getStreamData()
	}
	return v.getData()
}

// getData returns all votes, without clearing the slice
// can be used when this subscriber is used as aggregator subscriber
func (v VoteSub) getData() []types.Vote {
	return v.all
}

// getStreamData - returns the votes for this subscriber for a stream
// only returns new data (since last call), used when this subscriber is
// used to stream data to the client
func (v *VoteSub) getStreamData() []types.Vote {
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.all) == 0 {
		return nil
	}
	data := v.all
	v.all = make([]types.Vote, 0, cap(data))
	return data
}

func (v VoteSub) Types() []events.Type {
	return []events.Type{
		events.VoteEvent,
	}
}
