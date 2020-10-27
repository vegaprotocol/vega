package plugins

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"

	"github.com/pkg/errors"
)

var (
	ErrNoMarketOrPartyFilters = errors.New("market or party filters are required")
)

type LiquidityProvisionEvent interface {
	events.Event
	LiquidityProvision() types.LiquidityProvision
}

type LiquidityProvision struct {
	*subscribers.Base

	// marketID -> partyID -> liquidityProvision
	marketsLPs map[string]map[string]types.LiquidityProvision
	mu         sync.RWMutex
	ch         chan types.LiquidityProvision
}

func NewLiquidityProvision(ctx context.Context) *LiquidityProvision {
	l := &LiquidityProvision{
		Base:       subscribers.NewBase(ctx, 10, true),
		marketsLPs: map[string]map[string]types.LiquidityProvision{},
		ch:         make(chan types.LiquidityProvision, 100),
	}

	go l.consume()
	return l
}

func (l *LiquidityProvision) Push(evts ...events.Event) {
	for _, e := range evts {
		select {
		case <-l.Closed():
			return
		default:
			if lpe, ok := e.(LiquidityProvisionEvent); ok {
				l.ch <- lpe.LiquidityProvision()
			}
		}
	}
}

func (l *LiquidityProvision) consume() {
	defer func() { close(l.ch) }()
	for {
		select {
		case <-l.Closed():
			return
		case lp, ok := <-l.ch:
			if !ok {
				// cleanup base
				l.Halt()
				// channel is closed
				return
			}
			l.mu.Lock()
			partiesLPs, ok := l.marketsLPs[lp.MarketID]
			if !ok {
				partiesLPs = map[string]types.LiquidityProvision{}
				l.marketsLPs[lp.MarketID] = partiesLPs
			}
			partiesLPs[lp.PartyID] = lp
			l.mu.Unlock()
		}
	}
}

func (l *LiquidityProvision) Get(party, market *string) ([]types.LiquidityProvision, error) {
	if party == nil && market == nil {
		return nil, ErrNoMarketOrPartyFilters
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if market != nil {
		return l.getByMarket(*market, party), nil
	}
	return l.getByParty(*party), nil
}

func (l *LiquidityProvision) getByMarket(market string, party *string) []types.LiquidityProvision {
	partiesLPs, ok := l.marketsLPs[market]
	if !ok {
		return nil
	}

	if party != nil {
		partyLP, ok := partiesLPs[*party]
		if !ok {
			return nil
		}
		return []types.LiquidityProvision{partyLP}
	}

	out := make([]types.LiquidityProvision, 0, len(partiesLPs))
	for _, v := range partiesLPs {
		out = append(out, v)
	}
	return out
}

func (l *LiquidityProvision) getByParty(party string) []types.LiquidityProvision {
	out := []types.LiquidityProvision{}
	for _, v := range l.marketsLPs {
		if plp, ok := v[party]; ok {
			out = append(out, plp)
		}
	}
	return out
}

func (l *LiquidityProvision) Types() []events.Type {
	return []events.Type{
		events.LiquidityProvisionEvent,
	}
}
