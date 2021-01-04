package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

type VoteSub struct {
	*Base
	mu      *sync.Mutex
	all     []types.Vote
	filters []VoteFilter
	stream  bool
	update  chan struct{}
}

// VoteByPartyID filters votes cast by given party
func VoteByPartyID(id string) VoteFilter {
	return func(v types.Vote) bool {
		return v.PartyID == id
	}
}

func VoteByProposalID(id string) VoteFilter {
	return func(v types.Vote) bool {
		return v.ProposalID == id
	}
}

func NewVoteSub(ctx context.Context, stream, ack bool, filters ...VoteFilter) *VoteSub {
	v := &VoteSub{
		Base:    NewBase(ctx, 10, ack),
		mu:      &sync.Mutex{},
		all:     []types.Vote{},
		filters: filters,
		stream:  stream,
		update:  make(chan struct{}),
	}
	if v.isRunning() {
		go v.loop(v.ctx)
	}
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
				v.Push(e...)
			}
		}
	}
}

func (v *VoteSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	add := make([]types.Vote, 0, len(evts))
	for _, e := range evts {
		te, ok := e.(VoteE)
		if ok {
			vote := te.Vote()
			for _, f := range v.filters {
				if !f(vote) {
					ok = false
					break
				}
			}
			if ok {
				add = append(add, vote)
			}
		}
	}
	if len(add) == 0 {
		return
	}
	v.mu.Lock()
	// no data in subscriber, first time adding
	// close the update channel to signal callers they can call GetData
	if len(v.all) == 0 {
		close(v.update)
	}
	v.all = append(v.all, add...)
	v.mu.Unlock()
}

// Filter allows us to fetch votes using callbacks (e.g. filter out all votes by party)
func (v VoteSub) Filter(filters ...VoteFilter) []*types.Vote {
	ret := []*types.Vote{}
	for _, vote := range v.all {
		add := true
		for _, f := range filters {
			if !f(vote) {
				add = false
				break
			}
		}
		if add {
			cpy := vote
			ret = append(ret, &cpy)
		}
	}
	return ret
}

// GetData - either returns the full data-set, or just updates, depending on configuration
func (v *VoteSub) GetData() []types.Vote {
	if v.stream {
		// wait for data to have changed
		<-v.update
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
	data := v.all
	// GetData blocks on update channel prior to being called
	// now we can create a new channel
	v.update = make(chan struct{})
	v.all = make([]types.Vote, 0, cap(data))
	v.mu.Unlock()
	if len(data) == 0 {
		return nil
	}
	return data
}

func (v VoteSub) Types() []events.Type {
	return []events.Type{
		events.VoteEvent,
	}
}
