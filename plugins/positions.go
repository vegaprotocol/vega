package plugins

import (
	"context"
	"math"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/pkg/errors"
)

var (
	ErrMarketNotFound = errors.New("could not find market")
)

// SE SettleEvent - common denominator between SPE & SDE
type SE interface {
	events.Event
	PartyID() string
	MarketID() string
	Price() uint64
	Timestamp() int64
}

// SPE SettlePositionEvent
type SPE interface {
	SE
	Trades() []events.TradeSettlement
	Timestamp() int64
}

// SDE SettleDistressedEvent
type SDE interface {
	SE
	Margin() uint64
	Timestamp() int64
}

// LSE LossSocializationEvent
type LSE interface {
	events.Event
	PartyID() string
	MarketID() string
	Loss() *num.Uint
	Adjustment() *num.Uint
	Timestamp() int64
}

// Positions plugin taking settlement data to build positions API data
type Positions struct {
	*subscribers.Base
	mu   *sync.RWMutex
	data map[string]map[string]Position
}

func NewPositions(ctx context.Context) *Positions {
	return &Positions{
		Base: subscribers.NewBase(ctx, 10, true),
		mu:   &sync.RWMutex{},
		data: map[string]map[string]Position{},
	}
}

func (p *Positions) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	// lock here, because some of these events are sent in batch (if not all of them)
	p.mu.Lock()
	for _, e := range evts {
		switch te := e.(type) {
		case SPE:
			p.updatePosition(te)
		case SDE:
			p.updateSettleDestressed(te)
		case LSE:
			p.applyLossSocialization(te)
		}
	}
	p.mu.Unlock()
}

func (p *Positions) applyLossSocialization(e LSE) {
	marketID, partyID, loss, adjustment := e.MarketID(), e.PartyID(), e.Loss(), e.Adjustment()
	pos, ok := p.data[marketID][partyID]
	if !ok {
		return
	}
	if loss != nil  {
		pos.loss = pos.loss.Add(pos.loss, loss)
	} else {
		pos.adjustment = pos.adjustment.Add(pos.adjustment, adjustment)
	}
	pos.RealisedPnlFP += float64(loss.Uint64())
	pos.RealisedPnl += int64(loss.Uint64())
	pos.Position.UpdatedAt = e.Timestamp()
	p.data[marketID][partyID] = pos
}

func (p *Positions) updatePosition(e SPE) {
	mID, tID := e.MarketID(), e.PartyID()
	if _, ok := p.data[mID]; !ok {
		p.data[mID] = map[string]Position{}
	}
	calc, ok := p.data[mID][tID]
	if !ok {
		calc = seToProto(e)
	}
	updateSettlePosition(&calc, e)
	calc.Position.UpdatedAt = e.Timestamp()
	p.data[mID][tID] = calc
}

func (p *Positions) updateSettleDestressed(e SDE) {
	mID, tID := e.MarketID(), e.PartyID()
	if _, ok := p.data[mID]; !ok {
		p.data[mID] = map[string]Position{}
	}
	calc, ok := p.data[mID][tID]
	if !ok {
		calc = seToProto(e)
	}
	margin := e.Margin()
	calc.RealisedPnl += calc.UnrealisedPnl
	calc.RealisedPnlFP += calc.UnrealisedPnlFP
	calc.OpenVolume = 0
	calc.UnrealisedPnl = 0
	calc.AverageEntryPrice = 0
	// realised P&L includes whatever we had in margin account at this point
	calc.RealisedPnl -= int64(margin)
	calc.RealisedPnlFP -= float64(margin)
	// @TODO average entry price shouldn't be affected(?)
	// the volume now is zero, though, so we'll end up moving this position to storage
	calc.UnrealisedPnlFP = 0
	calc.AverageEntryPriceFP = 0
	calc.Position.UpdatedAt = e.Timestamp()
	p.data[mID][tID] = calc
}

// GetPositionsByMarketAndParty get the position of a single trader in a given market
func (p *Positions) GetPositionsByMarketAndParty(market, party string) (*types.Position, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	mp, ok := p.data[market]
	if !ok {
		return nil, nil
	}
	pos, ok := mp[party]
	if !ok {
		return nil, nil
	}
	return &pos.Position, nil
}

// GetPositionsByParty get all positions for a given trader
func (p *Positions) GetPositionsByParty(party string) ([]*types.Position, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// at most, trader is active in all markets
	positions := make([]*types.Position, 0, len(p.data))
	for _, traders := range p.data {
		if pos, ok := traders[party]; ok {
			positions = append(positions, &pos.Position)
		}
	}
	if len(positions) == 0 {
		return nil, nil
		// return nil, ErrPartyNotFound
	}
	return positions, nil
}

// GetAllPositions returns all positions, across markets
func (p *Positions) GetAllPositions() ([]*types.Position, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var pos []*types.Position
	for k := range p.data {
		// guesstimate what the slice cap ought to be: number of markets * number of traders in 1 market
		pos = make([]*types.Position, 0, len(p.data)*len(p.data[k]))
		break
	}
	for _, traders := range p.data {
		for _, tp := range traders {
			pos = append(pos, &tp.Position)
		}
	}
	return pos, nil
}

// GetPositionsByMarket get all trader positions in a given market
func (p *Positions) GetPositionsByMarket(market string) ([]*types.Position, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	mp, ok := p.data[market]
	if !ok {
		return nil, ErrMarketNotFound
	}
	s := make([]*types.Position, 0, len(mp))
	for _, tp := range mp {
		s = append(s, &tp.Position)
	}
	return s, nil
}

func calculateOpenClosedVolume(currentOpenVolume, tradedVolume int64) (int64, int64) {
	if currentOpenVolume != 0 && ((currentOpenVolume > 0) != (tradedVolume > 0)) {
		var closedVolume int64
		if absUint64(tradedVolume) > absUint64(currentOpenVolume) {
			closedVolume = currentOpenVolume
		} else {
			closedVolume = -tradedVolume
		}
		return tradedVolume + closedVolume, closedVolume
	}
	return tradedVolume, 0
}

func closeV(p *Position, closedVolume int64, tradedPrice uint64) float64 {
	if closedVolume == 0 {
		return 0
	}
	realisedPnlDelta := float64(closedVolume) * (float64(tradedPrice) - p.AverageEntryPriceFP)
	p.RealisedPnlFP += realisedPnlDelta
	p.OpenVolume -= closedVolume
	return realisedPnlDelta
}

func updateVWAP(vwap float64, volume int64, addVolume int64, addPrice uint64) float64 {
	if volume+addVolume == 0 {
		return 0
	}
	return ((vwap * float64(volume)) + (float64(addPrice) * float64(addVolume))) / (float64(volume) + float64(addVolume))
}

func openV(p *Position, openedVolume int64, tradedPrice uint64) {
	// calculate both average entry price here.
	p.AverageEntryPriceFP = updateVWAP(p.AverageEntryPriceFP, p.OpenVolume, openedVolume, tradedPrice)
	p.OpenVolume += openedVolume
}

func mtm(p *Position, markPrice uint64) {
	if p.OpenVolume == 0 {
		p.UnrealisedPnlFP = 0
		p.UnrealisedPnl = 0
		return
	}
	p.UnrealisedPnlFP = float64(p.OpenVolume) * (float64(markPrice) - p.AverageEntryPriceFP)
}

func updateSettlePosition(p *Position, e SPE) {
	for _, t := range e.Trades() {
		openedVolume, closedVolume := calculateOpenClosedVolume(p.OpenVolume, t.Size())
		_ = closeV(p, closedVolume, t.Price().Uint64())
		openV(p, openedVolume, t.Price().Uint64())
		p.AverageEntryPrice = uint64(math.Round(p.AverageEntryPriceFP))
		p.RealisedPnl = int64(math.Round(p.RealisedPnlFP))
	}
	mtm(p, e.Price())
	p.UnrealisedPnl = int64(math.Round(p.UnrealisedPnlFP))
}

type Position struct {
	types.Position
	AverageEntryPriceFP float64
	RealisedPnlFP       float64
	UnrealisedPnlFP     float64

	// what the party lost because of loss socialization
	loss *num.Uint
	// what a party was missing which triggered loss socialization
	adjustment *num.Uint
}

func seToProto(e SE) Position {
	return Position{
		Position: types.Position{
			MarketId: e.MarketID(),
			PartyId:  e.PartyID(),
		},
		AverageEntryPriceFP: 0,
		RealisedPnlFP:       0,
		UnrealisedPnlFP:     0,
	}
}

func absUint64(v int64) uint64 {
	if v < 0 {
		v *= -1
	}
	return uint64(v)
}

func (p *Positions) Types() []events.Type {
	return []events.Type{
		events.SettlePositionEvent,
		events.SettleDistressedEvent,
		events.LossSocializationEvent,
	}
}
