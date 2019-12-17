package positions

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/positions_subscriber_mock.go -package mocks code.vegaprotocol.io/vega/plugins/positions Subscriber
type Subscriber interface {
	Recv() <-chan []events.SettlePosition
	Done() <-chan struct{}
}

type Pos struct {
	mu   sync.RWMutex // sadly, we still need this because we'll be updating this map and reading from it
	sub  Subscriber
	data map[string]map[string]types.Position
}

func New(sub Subscriber) *Pos {
	return &Pos{
		sub:  sub,
		data: map[string]map[string]types.Position{},
	}
}

// Start - just an exposed func that hides the fact that we're using routines
// better for a clean API
func (p *Pos) Start(ctx context.Context) {
	go p.consume(ctx)
}

// GetPositionsByMarketAndParty get the position of a single trader in a given market
func (p *Pos) GetPositionsByMarketAndParty(market, party string) (*types.Position, error) {
	p.mu.RLock()
	mp, ok := p.data[market]
	if !ok {
		p.mu.RUnlock()
		return nil, ErrMarketNotFound
	}
	pos, ok := mp[party]
	if !ok {
		pos = types.Position{
			PartyID:  party,
			MarketID: market,
		}
		// return nil, ErrPartyNotFound
	}
	p.mu.RUnlock()
	return &pos, nil
}

// GetPositionsByParty get all positions for a given trader
func (p *Pos) GetPositionsByParty(party string) ([]*types.Position, error) {
	p.mu.RLock()
	// at most, trader is active in all markets
	positions := make([]*types.Position, 0, len(p.data))
	for _, traders := range p.data {
		if pos, ok := traders[party]; ok {
			positions = append(positions, &pos)
		}
	}
	p.mu.RUnlock()
	if len(positions) == 0 {
		return nil, nil
		// return nil, ErrPartyNotFound
	}
	return positions, nil
}

// GetPositionsByMarket get all trader positions in a given market
func (p *Pos) GetPositionsByMarket(market string) ([]*types.Position, error) {
	p.mu.RLock()
	mp, ok := p.data[market]
	if !ok {
		p.mu.RUnlock()
		return nil, ErrMarketNotFound
	}
	s := make([]*types.Position, 0, len(mp))
	for _, tp := range mp {
		s = append(s, &tp)
	}
	p.mu.RUnlock()
	return s, nil
}

func (p *Pos) consume(ctx context.Context) {
	for {
		select {
		case <-p.sub.Done():
			// if we no longer receive data, we can stop here
			return
		case <-ctx.Done():
			return
		case data, ok := <-p.sub.Recv():
			if !ok {
				// the channel was closed
				return
			}
			if len(data) == 0 {
				continue
			}
			p.mu.RLock()
			// get a copy, we only need a read lock here
			cpy := p.data
			p.mu.RUnlock()
			p.updateData(cpy, data)
		}
	}
}

func (p *Pos) updateData(data map[string]map[string]types.Position, raw []events.SettlePosition) {
	for _, sp := range raw {
		mID, tID := sp.MarketID(), sp.Party()
		if _, ok := data[mID]; !ok {
			data[mID] = map[string]types.Position{}
		}
		calc, ok := data[mID][tID]
		if !ok {
			calc = evtToProto(sp)
		}
		updatePosition(&calc, sp)
		data[mID][tID] = calc
		if calc.OpenVolume == 0 {
			delete(data[mID], tID)
		}
	}
	// keep lock time to a minimum, we're working on a copy here, and reassign the data field
	// only after everything has been updated (instead of maintaining a lock throughout)
	p.mu.Lock()
	p.data = data
	p.mu.Unlock()
}

func (p *Pos) QuickNDirty(e events.SettlePosition) {
	p.mu.RLock()
	cpy := p.data
	p.mu.RUnlock()
	p.updateData(cpy, []events.SettlePosition{e})
}

func updatePosition(p *types.Position, e events.SettlePosition) {
	current := p.OpenVolume
	var (
		// delta uint64
		pnl, delta int64
	)
	tradePnl := make([]int64, 0, len(e.Trades()))
	for _, t := range e.Trades() {
		size, sAbs := t.Size(), absUint64(t.Size())
		if current != 0 {
			cAbs := absUint64(current)
			// trade direction is actually closing volume
			if (current > 0 && size < 0) || (current < 0 && size > 0) {
				if sAbs > cAbs {
					delta = current
					current = 0
				} else {
					delta = -size
					current += size
				}
			}
			pnl = delta * int64(t.Price()-p.AverageEntryPrice)
			p.RealisedPNL += pnl
			tradePnl = append(tradePnl, pnl)
			// @TODO store trade record with this realised P&L value
		}
		net := delta + size
		if net != 0 {
			if size != p.OpenVolume {
				p.AverageEntryPrice = (p.AverageEntryPrice*absUint64(p.OpenVolume) + t.Price()*absUint64(size)) / uint64(p.OpenVolume+size)
			} else {
				p.AverageEntryPrice = 0
			}
		}
		p.OpenVolume += size
	}
	// p.PendingVolume = p.OpenVolume + e.Buy() - e.Sell()
	// MTM price * open volume == total value of current pos the entry price/cost of said position
	p.UnrealisedPNL = (int64(e.Price()) - int64(p.AverageEntryPrice)) * p.OpenVolume
	// Technically not needed, but safer to copy the open volume from event regardless
	p.OpenVolume = e.Size()
	if p.OpenVolume != 0 && p.AverageEntryPrice == 0 {
		p.AverageEntryPrice = e.Price()
	}
}

func evtToProto(e events.SettlePosition) types.Position {
	p := types.Position{
		MarketID: e.MarketID(),
		PartyID:  e.Party(),
	}
	// NOTE: We don't call this here because the call is made in updateEvt for all positions
	// we don't want to add the same data twice!
	// updatePosition(&p, e)
	return p
}

func absUint64(v int64) uint64 {
	if v < 0 {
		v *= -1
	}
	return uint64(v)
}
