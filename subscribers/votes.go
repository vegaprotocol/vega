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
}

func NewVoteSub(ctx context.Context, filters ...VoteFilter) *VoteSub {
	v := &VoteSub{
		Base:    newBase(ctx, 10),
		all:     []types.Vote{},
		filters: filters,
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

func (v *VoteSub) GetData() []types.Vote {
	v.mu.Lock()
	if len(v.all) == 0 {
		v.mu.Unlock()
		return nil
	}
	data := v.all
	v.all = make([]types.Vote, 0, cap(data))
	v.mu.Unlock()
	return data
}

func (v VoteSub) Types() []events.Type {
	return []events.Type{
		events.VoteEvent,
	}
}
