package positions

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

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
		p.mu.RUnlock()
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

func updatePosition(p *types.Position, e events.SettlePosition) {
	totPrice, totVolume := p.AverageEntryPrice, p.OpenVolume
	totPrice *= absUint64(totVolume)
	for _, t := range e.Trades() {
		price, size := t.Price(), t.Size()
		if size == 0 {
			continue
		}
		app := true
		for i, pt := range p.FifoQueue {
			if pt.Price == price {
				pt.Volume += size
				app = false
				if pt.Volume == 0 {
					p.FifoQueue = p.FifoQueue[:i+copy(p.FifoQueue[i:], p.FifoQueue[i+1:])]
				}
				break
			}
		}
		totPrice += price * absUint64(size)
		totVolume += size
		if app {
			p.FifoQueue = append(p.FifoQueue, &types.PositionTrade{
				Volume: size,
				Price:  price,
			})
		}
	}
	if totVolume == 0 {
		totVolume = 1
	}
	p.PendingVolume = p.OpenVolume + e.Buy() - e.Sell()
	p.AverageEntryPrice = totPrice / absUint64(totVolume)
	// MTM price * open volume == total value of current pos the entry price/cost of said position
	p.UnrealisedPNL = int64(e.Price())*p.OpenVolume - p.OpenVolume*int64(p.AverageEntryPrice)
	// get the realised PNL (final value of asset should market settle at current price)
	p.RealisedPNL = int64(e.Price())*p.PendingVolume - p.PendingVolume*int64(p.AverageEntryPrice)
}

func evtToProto(e events.SettlePosition) types.Position {
	trades := e.Trades()
	p := types.Position{
		MarketID:   e.MarketID(),
		PartyID:    e.Party(),
		OpenVolume: e.Size(),
		FifoQueue:  make([]*types.PositionTrade, 0, len(trades)),
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
