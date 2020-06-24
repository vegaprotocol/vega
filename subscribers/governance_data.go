package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type GovernanceDataSub struct {
	*Base
	mu        sync.Mutex
	proposals map[string]types.Proposal
	byPID     map[string]*types.GovernanceData
	all       []*types.GovernanceData
}

func NewGovernanceDataSub(ctx context.Context) *GovernanceDataSub {
	gd := &GovernanceDataSub{
		Base:      newBase(ctx, 10),
		proposals: map[string]types.Proposal{},
		byPID:     map[string]*types.GovernanceData{},
		all:       []*types.GovernanceData{},
	}
	gd.running = true
	go gd.loop(ctx)
	return gd
}

func (g *GovernanceDataSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			g.Halt()
		case e := <-g.ch:
			if g.isRunning() {
				g.Push(e)
			}
		}
	}
}

func (g *GovernanceDataSub) Push(e events.Event) {
	switch et := e.(type) {
	case PropE:
		prop := et.Proposal()
		gd := g.getData(prop.ID)
		g.proposals[prop.ID] = prop
		gd.Proposal = &prop
	case VoteE:
		vote := et.Vote()
		gd := g.getData(vote.ProposalID)
		if vote.Value == types.Vote_VALUE_YES {
			gd.Yes = append(gd.Yes, &vote)
			delete(gd.NoParty, vote.PartyID)
			gd.YesParty[vote.PartyID] = &vote
		} else {
			gd.No = append(gd.No, &vote)
			delete(gd.YesParty, vote.PartyID)
			gd.NoParty[vote.PartyID] = &vote
		}
	}
}

// Filter - get filtered proposals, value receiver so no data races
// uniqueVotes will replace the slice (containing older/duplicate votes) with only the latest votes
// for each participant
func (g GovernanceDataSub) Filter(uniqueVotes bool, params ...ProposalFilter) []*types.GovernanceData {
	ret := []*types.GovernanceData{}
	for id, p := range g.proposals {
		add := true
		for _, f := range params {
			if !f(p) {
				add = false
				break
			}
		}
		if add {
			// create a copy
			gd := *g.byPID[id]
			if uniqueVotes {
				gd.Yes = make([]*types.Vote, 0, len(gd.YesParty))
				for _, v := range gd.YesParty {
					gd.Yes = append(gd.Yes, v)
				}
				gd.No = make([]*types.Vote, 0, len(gd.NoParty))
				for _, v := range gd.NoParty {
					gd.No = append(gd.No, v)
				}
			}
			//  add to the return value
			ret = append(ret, &gd)
		}
	}
	return ret
}

func (g *GovernanceDataSub) getData(id string) *types.GovernanceData {
	g.mu.Lock()
	defer g.mu.Unlock()
	if gd, ok := g.byPID[id]; ok {
		return gd
	}
	gd := &types.GovernanceData{
		Yes:      []*types.Vote{},
		No:       []*types.Vote{},
		YesParty: map[string]*types.Vote{},
		NoParty:  map[string]*types.Vote{},
	}
	g.byPID[id] = gd
	g.all = append(g.all, gd)
	return gd
}

func (g *GovernanceDataSub) Types() []events.Type {
	return []events.Type{
		events.ProposalEvent,
		events.VoteEvent,
	}
}
