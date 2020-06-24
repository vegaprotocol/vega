package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type GovernanceAggSub struct {
	*Base
	mu    sync.Mutex
	byPID map[string]*types.GovernanceData
}

func NewGoveranceAggSub(ctx context.Context) *GovernanceAggSub {
	gd := GovernanceAggSub{
		Base:  newBase(ctx, 10),
		byPID: map[string]*types.GovernanceData{},
	}
	gd.running = true
	go gd.loop(gd.ctx)
	return &gd
}

func (g *GovernanceAggSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			g.Halt()
			return
		case e := <-g.ch:
			if g.isRunning() {
				g.Push(e)
			}
		}
	}
}

func (g *GovernanceAggSub) getData(id string) *types.GovernanceData {
	gd, ok := g.byPID[id]
	if !ok {
		gd = &types.GovernanceData{
			Yes:      []*types.Vote{},
			No:       []*types.Vote{},
			YesParty: map[string]*types.Vote{},
			NoParty:  map[string]*types.Vote{},
		}
		g.byPID[id] = gd
	}
	return gd
}

func (g *GovernanceAggSub) Push(e events.Event) {
	if _, ok := e.(TimeEvent); ok {
		g.mu.Lock()
		for id, d := range g.byPID {
			g.all[id] = *d
		}
		g.mu.Unlock()
		return
	}
	g.GovernanceSub.Push(e)
}

func (g *GovernanceAggSub) Types() []events.Type {
	t := g.GovernanceSub.Types()
	return append(t, events.TimeUpdate)
}
