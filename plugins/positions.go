package plugins

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/types"
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
	Price() *num.Uint
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
	Margin() *num.Uint
	Timestamp() int64
}

// LSE LossSocializationEvent
type LSE interface {
	events.Event
	PartyID() string
	MarketID() string
	Amount() int64
	AmountLost() int64
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
	marketID, partyID, amountLoss := e.MarketID(), e.PartyID(), num.DecimalFromUint(num.NewUint(uint64(e.AmountLost())))
	pos, ok := p.data[marketID][partyID]
	if !ok {
		return
	}
	if amountLoss.LessThan(num.DecimalFromFloat(0.0)) {
		pos.loss.Sub(amountLoss)
	} else {
		pos.adjustment.Add(amountLoss)
	}
	pos.RealisedPnlFP.Add(amountLoss)
	pos.RealisedPnl.Add(amountLoss)
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
	calc.RealisedPnl.Add(calc.UnrealisedPnl)
	calc.RealisedPnlFP.Add(calc.UnrealisedPnlFP)
	calc.OpenVolume = 0
	calc.UnrealisedPnl = num.NewDecimalFromFloat(0)
	calc.AverageEntryPrice = num.NewUint(0)
	// realised P&L includes whatever we had in margin account at this point
	calc.RealisedPnl.Sub(num.DecimalFromUint(margin))
	calc.RealisedPnlFP.Sub(num.DecimalFromUint(margin))
	// @TODO average entry price shouldn't be affected(?)
	// the volume now is zero, though, so we'll end up moving this position to storage
	calc.UnrealisedPnlFP = num.DecimalFromFloat(0.0)
	calc.AverageEntryPriceFP = num.DecimalFromFloat(0.0)
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

func closeV(p *Position, closedVolume int64, tradedPrice *num.Uint) num.Decimal {
	if closedVolume == 0 {
		return num.DecimalFromFloat(0.0)
	}
	realisedPnlDelta := num.DecimalFromUint(tradedPrice).Sub(p.AverageEntryPriceFP).Mul(num.DecimalFromUint(num.NewUint(uint64(closedVolume))))
	p.RealisedPnlFP.Add(realisedPnlDelta)
	p.OpenVolume -= closedVolume
	return realisedPnlDelta
}

func updateVWAP(vwap num.Decimal, volume int64, addVolume int64, addPrice *num.Uint) num.Decimal {
	if volume+addVolume == 0 {
		return num.DecimalFromFloat(0.0)
	}

	volumeDec := num.DecimalFromFloat(float64(volume))
	addVolumeDec := num.DecimalFromFloat(float64(addVolume))
	addPriceDec := num.DecimalFromUint(addPrice)

	//	return ((vwap * float64(volume)) + (float64(addPrice) * float64(addVolume))) / (float64(volume) + float64(addVolume))
	return vwap.Mul(volumeDec).Add(addPriceDec.Mul(addVolumeDec)).Div(volumeDec.Add(addVolumeDec))
}

func openV(p *Position, openedVolume int64, tradedPrice *num.Uint) {
	// calculate both average entry price here.
	p.AverageEntryPriceFP = updateVWAP(p.AverageEntryPriceFP, p.OpenVolume, openedVolume, tradedPrice)
	p.OpenVolume += openedVolume
}

func mtm(p *Position, markPrice *num.Uint) {
	if p.OpenVolume == 0 {
		p.UnrealisedPnlFP = num.DecimalFromFloat(0.0)
		p.UnrealisedPnl = num.DecimalFromFloat(0.0)
		return
	}
	markPriceDec := num.DecimalFromUint(markPrice)
	openVolumeDec := num.DecimalFromFloat(float64(p.OpenVolume))

	//	p.UnrealisedPnlFP = float64(p.OpenVolume) * (float64(markPrice) - p.AverageEntryPriceFP)
	p.UnrealisedPnlFP = openVolumeDec.Mul(markPriceDec.Sub(p.AverageEntryPriceFP))
}

func updateSettlePosition(p *Position, e SPE) {
	var overflow bool
	for _, t := range e.Trades() {
		openedVolume, closedVolume := calculateOpenClosedVolume(p.OpenVolume, t.Size())
		_ = closeV(p, closedVolume, t.Price().Clone())
		openV(p, openedVolume, t.Price().Clone())
		p.AverageEntryPrice, overflow = num.UintFromDecimal(p.AverageEntryPriceFP.Round(0))
		if overflow {
			// We need to report the error somehow
		}

		p.RealisedPnl = p.RealisedPnlFP.Round(0)
	}
	mtm(p, e.Price().Clone())
	p.UnrealisedPnl = p.UnrealisedPnlFP.Round(0)
}

type Position struct {
	types.Position
	AverageEntryPriceFP num.Decimal
	RealisedPnlFP       num.Decimal
	UnrealisedPnlFP     num.Decimal

	// what the party lost because of loss socialization
	loss num.Decimal
	// what a party was missing which triggered loss socialization
	adjustment num.Decimal
}

func seToProto(e SE) Position {
	return Position{
		Position: types.Position{
			MarketId: e.MarketID(),
			PartyId:  e.PartyID(),
		},
		AverageEntryPriceFP: num.NewDecimalFromFloat(0.0),
		RealisedPnlFP:       num.NewDecimalFromFloat(0.0),
		UnrealisedPnlFP:     num.NewDecimalFromFloat(0.0),
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
