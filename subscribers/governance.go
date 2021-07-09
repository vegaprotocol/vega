package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type GovernanceEvent interface {
	events.Event
	ProposalID() string
	PartyID() string
}

type VoteE interface {
	GovernanceEvent
	Vote() types.Vote
	Value() types.Vote_Value
}

// ProposalFilter - a callback to be applied to the proposals we're interested in
// some common filter calls will be provided
type ProposalFilter func(p types.Proposal) bool

// GovernanceFilter - callback to filter out proposal and vote events we're after
type GovernanceFilter func(e GovernanceEvent) bool

// VoteFilter - callbacks to filter out only the vote events we're after
type VoteFilter func(v types.Vote) bool

// Filter - variadic argument for constructor so we can set different types of filters
// as a single vararg
type Filter func(g *GovernanceSub)

type GovernanceSub struct {
	*Base
	gfilters []GovernanceFilter
	pfilters []ProposalFilter
	vfilters []VoteFilter
	combined []*types.GovernanceData
	byPID    map[string]*types.GovernanceData
	changed  map[string]types.GovernanceData
	update   chan struct{}
	mu       *sync.Mutex
}

// Governance - vararg to set governance filters
func Governance(f ...GovernanceFilter) Filter {
	return func(g *GovernanceSub) {
		g.gfilters = f
	}
}

// Proposals - varargs setting filters for proposals
func Proposals(f ...ProposalFilter) Filter {
	return func(g *GovernanceSub) {
		g.pfilters = f
	}
}

// Votes - vararg setting filters on votes
func Votes(f ...VoteFilter) Filter {
	return func(g *GovernanceSub) {
		g.vfilters = f
	}
}

// NewGovernanceSub creates a new governance data subscriber with specific filters
// this subscriber is used by the governance service for gRPC streams, not for general use
// @TODO this subscriber needs to move to the appropriate package
func NewGovernanceSub(ctx context.Context, ack bool, filters ...Filter) *GovernanceSub {
	g := GovernanceSub{
		Base:     NewBase(ctx, 10, ack),
		gfilters: []GovernanceFilter{},
		pfilters: []ProposalFilter{},
		vfilters: []VoteFilter{},
		combined: []*types.GovernanceData{},
		changed:  map[string]types.GovernanceData{},
		byPID:    map[string]*types.GovernanceData{},
		update:   make(chan struct{}),
		mu:       &sync.Mutex{},
	}
	for _, f := range filters {
		f(&g)
	}
	if g.isRunning() {
		go g.loop(g.ctx)
	}
	return &g
}

func (g *GovernanceSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			g.Halt()
			return
		case e := <-g.ch:
			if g.isRunning() {
				g.Push(e...)
			}
		}
	}
}

func (g *GovernanceSub) Halt() {
	g.mu.Lock()
	// update channel is  open, close it
	if len(g.changed) == 0 {
		close(g.update)
	}
	g.mu.Unlock()
	g.Base.Halt()
}

func (g *GovernanceSub) filter(e GovernanceEvent) bool {
	for _, f := range g.gfilters {
		if !f(e) {
			return false
		}
	}
	switch et := e.(type) {
	case PropE:
		p := et.Proposal()
		for _, f := range g.pfilters {
			if !f(p) {
				return false
			}
		}
	case VoteE:
		v := et.Vote()
		for _, f := range g.vfilters {
			if !f(v) {
				return false
			}
		}
	}
	return true
}

func (g *GovernanceSub) Push(evts ...events.Event) {
	for _, e := range evts {
		// we only deal with GovernanceEvents, and only if the filter applies at that
		ge, ok := e.(GovernanceEvent)
		if !ok || !g.filter(ge) {
			continue
		}
		// this data isn't stored ATM, so we'll acquire a lock per event
		// meanwhile we can continue to serve the API, provided we get the data copied fast
		// which, in getData, we do (lock, copy, unlock)
		g.mu.Lock()
		closeUpdate := false
		if len(g.changed) == 0 { // no data has changed
			closeUpdate = true
		}
		switch et := e.(type) {
		case PropE:
			prop := et.Proposal()
			gd := g.getData(prop.Id)
			gd.Proposal = &prop
			g.changed[prop.Id] = *gd
		case VoteE:
			vote := et.Vote()
			gd := g.getData(vote.ProposalId)
			if vote.Value == types.Vote_VALUE_YES {
				delete(gd.NoParty, vote.PartyId)
				gd.YesParty[vote.PartyId] = &vote
			} else {
				delete(gd.YesParty, vote.PartyId)
				gd.NoParty[vote.PartyId] = &vote
			}
			g.changed[vote.ProposalId] = *gd
		}
		// data has changed for the first time
		// close the signal channel
		if closeUpdate && len(g.changed) > 0 {
			close(g.update)
		}
		g.mu.Unlock()
	}
}

func (g *GovernanceSub) getData(id string) *types.GovernanceData {
	gd, ok := g.byPID[id]
	if !ok {
		gd = &types.GovernanceData{
			Yes:      []*types.Vote{},
			No:       []*types.Vote{},
			YesParty: map[string]*types.Vote{},
			NoParty:  map[string]*types.Vote{},
		}
		g.byPID[id] = gd
		g.combined = append(g.combined, gd)
	}
	return gd
}

func (g *GovernanceSub) Types() []events.Type {
	return []events.Type{
		events.ProposalEvent,
		events.VoteEvent,
	}
}

// GetGovernanceData - returns current data, clears up the changed map
// and reassignes the update channel.
// This call will block until the update channel has been closed. This avoids a busy loop
// in the caller (governance service). With this mechanism, we can just call this function
// knowing it'll only return once governance data has well and truly been updated
func (g *GovernanceSub) GetGovernanceData() []types.GovernanceData {
	// block on channel. This call will wait for the update channel to be closed
	<-g.update
	g.mu.Lock()
	// copy the map of changed proposals, and return that subset
	// no need to check if changed is empty -> update channel was closed!
	data := g.changed
	// reset the changes, so next time we call, there's nothing to worry about
	g.changed = map[string]types.GovernanceData{}
	// reset change indication channel
	g.update = make(chan struct{})
	g.mu.Unlock()
	// create a copy
	ret := make([]types.GovernanceData, 0, len(data))
	// copy the votes
	for _, d := range data {
		d.Yes = make([]*types.Vote, 0, len(d.YesParty))
		for _, v := range d.YesParty {
			vc := *v
			d.Yes = append(d.Yes, &vc)
		}
		d.No = make([]*types.Vote, 0, len(d.NoParty))
		for _, v := range d.NoParty {
			vc := *v
			d.No = append(d.No, &vc)
		}
		ret = append(ret, d)
	}
	return ret
}
