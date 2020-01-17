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
	}
}

// GetPositionsByMarketAndParty get the position of a single trader in a given market
func (p *Positions) GetPositionsByMarketAndParty(market, party string) (*types.Position, error) {
	p.mu.RLock()
	mp, ok := p.data[market]
	if !ok {
		p.mu.RUnlock()
		return nil, nil
	}
	pos, ok := mp[party]
	if !ok {
		p.mu.RUnlock()
		pos = types.Position{
			PartyID:  party,
			MarketID: market,
		}
		return nil, nil
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
	// if this settlePosition event has a margin event embedded, that means we're dealing
	// with a trader who was closed out...
	if margin, ok := e.Margin(); ok {
		p.OpenVolume = 0
		p.UnrealisedPNL = 0
		p.AverageEntryPrice = 0
		// realised P&L includes whatever we had in margin account at this point
		p.RealisedPNL -= int64(margin.MarginBalance())
		// @TODO average entry price shouldn't be affected(?)
		// the volume now is zero, though, so we'll end up moving this position to storage
		return
	}
	current := p.OpenVolume
	var (
		delta int64
		reset bool
	)
	if current == 0 {
		reset = true
	}
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
			// only increment realised P&L if the size goes the opposite way compared to the the
			// current position
			if (size > 0 && p.OpenVolume <= 0) || (size < 0 && p.OpenVolume >= 0) {
				p.RealisedPNL += delta * int64(t.Price()-p.AverageEntryPrice)
			}
			// @TODO store trade record with this realised P&L value
		}
		// we're going to set the AEP to the mark price, the realised P&L at this point
		// is the MTM of the trade price <-> mark price
		if reset {
			// the P&L is the value at mark price - value at trade price
			p.RealisedPNL += (t.Size() * int64(e.Price())) - (t.Size() * int64(t.Price()))
		}
		if net := delta + size; net != 0 {
			p.AverageEntryPrice = calcAEP(size, p.OpenVolume, t.Price(), p.AverageEntryPrice)
		}
		p.OpenVolume += size
		current += size
		delta = 0
	}
	if reset {
		p.AverageEntryPrice = e.Price()
	}

	// p.PendingVolume = p.OpenVolume + e.Buy() - e.Sell()
	// MTM price * open volume == total value of current pos the entry price/cost of said position
	p.UnrealisedPNL = (int64(e.Price()) - int64(p.AverageEntryPrice)) * p.OpenVolume
	// Technically not needed, but safer to copy the open volume from event regardless
	p.OpenVolume = e.Size()
	if p.OpenVolume == 0 {
		p.UnrealisedPNL = 0
		p.AverageEntryPrice = 0
		return
	}
	if p.AverageEntryPrice == 0 {
		p.AverageEntryPrice = e.Price()
	}
}

func calcAEP(size, open int64, newPrice, avgPrice uint64) uint64 {
	// first up: AEP is zero if we close out:
	if size == 0 {
		return 0
	}
	// our position has reduced (short to less short, long to less long)
	// but we've not closed out or inverted (short -> long, long -> short)
	// then the AEP remains as is
	if (open > 0 && size > 0 && open > size) || (open < 0 && size < 0 && open < size) {
		return avgPrice
	}
	oAbs, sAbs := absUint64(open), absUint64(size)
	// also create signed versions of abs value to see if we flipped position
	soAbs, ssAbs := int64(oAbs), int64(sAbs)
	// the abs value of one volume doesn't match, but the other doesn't -> we've flipped position
	if (soAbs != open && ssAbs == size) || (soAbs == open && ssAbs != size) {
		return newPrice // we went from long to short (or short to long) -> the average entry price == new price
	}
	// we've increased our position: long -> longer || short -> shorter
	// we need to calculate the new AEP: average price * open + new price * (new size-old size) / newSize
	delta := absUint64(int64(oAbs) - int64(sAbs))
	return (avgPrice*oAbs + newPrice*delta) / sAbs
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
