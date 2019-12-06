package plugins

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrMarketNotFound = errors.New("could not find market")
	ErrPartyNotFound  = errors.New("party not found")
)

// PosBuffer ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/pos_buffer_mock.go -package mocks code.vegaprotocol.io/vega/plugins PosBuffer
type PosBuffer interface {
	Subscribe() (<-chan []events.SettlePosition, int)
	Unsubscribe(int)
}

// Positions plugin taking settlement data to build positions API data
type Positions struct {
	mu   *sync.RWMutex
	buf  PosBuffer
	ref  int
	ch   <-chan []events.SettlePosition
	data map[string]map[string]types.Position
}

func NewPositions(buf PosBuffer) *Positions {
	return &Positions{
		mu:   &sync.RWMutex{},
		data: map[string]map[string]types.Position{},
		buf:  buf,
	}
}

func (p *Positions) Start(ctx context.Context) {
	p.mu.Lock()
	if p.ch == nil {
		// get the channel and the reference
		p.ch, p.ref = p.buf.Subscribe()
		// start consuming the data
		go p.consume(ctx)
	}
	p.mu.Unlock()
}

func (p *Positions) Stop() {
	p.mu.Lock()
	if p.ch != nil {
		// only unsubscribe if ch was set, otherwise we might end up unregistering ref 0, which
		// could (in theory at least) be used by another component
		p.buf.Unsubscribe(p.ref)
		p.ch = nil
		p.ref = 0
	}
	// we don't need to reassign ch here, because the channel is closed, the consume routine
	// will pick up on the fact that we don't have to consume data anylonger, and the ch/ref fields
	// will be unset there
	p.mu.Unlock()
}

// consume keep reading the channel for as long as we need to
func (p *Positions) consume(ctx context.Context) {
	defer func() {
		p.mu.Lock()
		p.buf.Unsubscribe(p.ref)
		p.ref = 0
		p.ch = nil
		p.mu.Unlock()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case update, open := <-p.ch:
			if !open {
				return
			}
			p.mu.Lock()
			p.updateData(update)
			p.mu.Unlock()
		}
	}
}

func (p *Positions) updateData(raw []events.SettlePosition) {
	for _, sp := range raw {
		mID, tID := sp.MarketID(), sp.Party()
		if _, ok := p.data[mID]; !ok {
			p.data[mID] = map[string]types.Position{}
		}
		calc, ok := p.data[mID][tID]
		if !ok {
			calc = evtToProto(sp)
		}
		updatePosition(&calc, sp)
		p.data[mID][tID] = calc
		if calc.OpenVolume == 0 {
			delete(p.data[mID], tID)
		}
	}
}

// GetPositionsByMarketAndParty get the position of a single trader in a given market
func (p *Positions) GetPositionsByMarketAndParty(market, party string) (*types.Position, error) {
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
func (p *Positions) GetPositionsByParty(party string) ([]*types.Position, error) {
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
func (p *Positions) GetPositionsByMarket(market string) ([]*types.Position, error) {
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
