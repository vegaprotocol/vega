// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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

var ErrMarketNotFound = errors.New("could not find market")

// SE SettleEvent - common denominator between SPE & SDE.
type SE interface {
	events.Event
	PartyID() string
	MarketID() string
	Price() *num.Uint
	Timestamp() int64
}

// SPE SettlePositionEvent.
type SPE interface {
	SE
	PositionFactor() num.Decimal
	Trades() []events.TradeSettlement
	Timestamp() int64
}

// SDE SettleDistressedEvent.
type SDE interface {
	SE
	Margin() *num.Uint
	Timestamp() int64
}

// LSE LossSocializationEvent.
type LSE interface {
	events.Event
	PartyID() string
	MarketID() string
	Amount() *num.Int
	Timestamp() int64
}

// Positions plugin taking settlement data to build positions API data.
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
	marketID, partyID, amountLoss := e.MarketID(), e.PartyID(), num.DecimalFromInt(e.Amount())
	pos, ok := p.data[marketID][partyID]
	if !ok {
		return
	}
	if amountLoss.IsNegative() {
		pos.loss = pos.loss.Add(amountLoss)
	} else {
		pos.adjustment = pos.adjustment.Add(amountLoss)
	}
	pos.RealisedPnlFP = pos.RealisedPnlFP.Add(amountLoss)
	pos.RealisedPnl = pos.RealisedPnl.Add(amountLoss)

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
	calc.RealisedPnl = calc.RealisedPnl.Add(calc.UnrealisedPnl)
	calc.RealisedPnlFP = calc.RealisedPnlFP.Add(calc.UnrealisedPnlFP)
	calc.OpenVolume = 0
	calc.UnrealisedPnl = num.DecimalZero()
	calc.AverageEntryPrice = num.Zero()
	// realised P&L includes whatever we had in margin account at this point
	dMargin := num.DecimalFromUint(margin)
	calc.RealisedPnl = calc.RealisedPnl.Sub(dMargin)
	calc.RealisedPnlFP = calc.RealisedPnlFP.Sub(dMargin)
	// @TODO average entry price shouldn't be affected(?)
	// the volume now is zero, though, so we'll end up moving this position to storage
	calc.UnrealisedPnlFP = num.DecimalZero()
	calc.AverageEntryPriceFP = num.DecimalZero()
	calc.Position.UpdatedAt = e.Timestamp()
	p.data[mID][tID] = calc
}

// GetPositionsByMarketAndParty get the position of a single party in a given market.
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

// GetPositionsByParty get all positions for a given party.
func (p *Positions) GetPositionsByParty(party string) ([]*types.Position, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// at most, party is active in all markets
	positions := make([]*types.Position, 0, len(p.data))
	for _, parties := range p.data {
		if pos, ok := parties[party]; ok {
			positions = append(positions, &pos.Position)
		}
	}
	if len(positions) == 0 {
		return nil, nil
		// return nil, ErrPartyNotFound
	}
	return positions, nil
}

// GetAllPositions returns all positions, across markets.
func (p *Positions) GetAllPositions() ([]*types.Position, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var pos []*types.Position
	for k := range p.data {
		// guesstimate what the slice cap ought to be: number of markets * number of parties in 1 market
		pos = make([]*types.Position, 0, len(p.data)*len(p.data[k]))
		break
	}
	for _, parties := range p.data {
		for _, tp := range parties {
			tp := tp
			pos = append(pos, &tp.Position)
		}
	}
	return pos, nil
}

// GetPositionsByMarket get all party positions in a given market.
func (p *Positions) GetPositionsByMarket(market string) ([]*types.Position, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	mp, ok := p.data[market]
	if !ok {
		return nil, ErrMarketNotFound
	}
	s := make([]*types.Position, 0, len(mp))
	for _, tp := range mp {
		tp := tp
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

func closeV(p *Position, closedVolume int64, tradedPrice *num.Uint, positionFactor num.Decimal) num.Decimal {
	if closedVolume == 0 {
		return num.DecimalZero()
	}
	realisedPnlDelta := num.DecimalFromUint(tradedPrice).Sub(p.AverageEntryPriceFP).Mul(num.DecimalFromInt64(closedVolume)).Div(positionFactor)
	p.RealisedPnlFP = p.RealisedPnlFP.Add(realisedPnlDelta)
	p.OpenVolume -= closedVolume
	return realisedPnlDelta
}

func updateVWAP(vwap num.Decimal, volume int64, addVolume int64, addPrice *num.Uint) num.Decimal {
	if volume+addVolume == 0 {
		return num.DecimalZero()
	}

	volumeDec := num.DecimalFromInt64(volume)
	addVolumeDec := num.DecimalFromInt64(addVolume)
	addPriceDec := num.DecimalFromUint(addPrice)

	//	return ((vwap * float64(volume)) + (float64(addPrice) * float64(addVolume))) / (float64(volume) + float64(addVolume))
	return vwap.Mul(volumeDec).Add(addPriceDec.Mul(addVolumeDec)).Div(volumeDec.Add(addVolumeDec))
}

func openV(p *Position, openedVolume int64, tradedPrice *num.Uint) {
	// calculate both average entry price here.
	p.AverageEntryPriceFP = updateVWAP(p.AverageEntryPriceFP, p.OpenVolume, openedVolume, tradedPrice)
	p.OpenVolume += openedVolume
}

func mtm(p *Position, markPrice *num.Uint, positionFactor num.Decimal) {
	if p.OpenVolume == 0 {
		p.UnrealisedPnlFP = num.DecimalZero()
		p.UnrealisedPnl = num.DecimalZero()
		return
	}
	markPriceDec := num.DecimalFromUint(markPrice)
	openVolumeDec := num.DecimalFromInt64(p.OpenVolume)

	//	p.UnrealisedPnlFP = float64(p.OpenVolume) * (float64(markPrice) - p.AverageEntryPriceFP)
	p.UnrealisedPnlFP = openVolumeDec.Mul(markPriceDec.Sub(p.AverageEntryPriceFP)).Div(positionFactor)
}

func updateSettlePosition(p *Position, e SPE) {
	for _, t := range e.Trades() {
		pr := t.Price()
		openedVolume, closedVolume := calculateOpenClosedVolume(p.OpenVolume, t.Size())
		_ = closeV(p, closedVolume, pr, e.PositionFactor())
		openV(p, openedVolume, pr)
		p.AverageEntryPrice, _ = num.UintFromDecimal(p.AverageEntryPriceFP.Round(0))

		p.RealisedPnl = p.RealisedPnlFP.Round(0)
	}
	mtm(p, e.Price(), e.PositionFactor())
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
			MarketId:          e.MarketID(),
			PartyId:           e.PartyID(),
			AverageEntryPrice: num.Zero(),
		},
		AverageEntryPriceFP: num.DecimalZero(),
		RealisedPnlFP:       num.DecimalZero(),
		UnrealisedPnlFP:     num.DecimalZero(),
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
