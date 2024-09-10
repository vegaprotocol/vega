// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package plugins

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/subscribers"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
)

var ErrMarketNotFound = errors.New("could not find market")

// FP FundingPaymentsEvent.
type FP interface {
	events.Event
	MarketID() string
	IsParty(id string) bool
	FundingPayments() *eventspb.FundingPayments
}

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
	IsFunding() bool
}

// DOC DistressedOrdersClosedEvent.
type DOC interface {
	events.Event
	MarketID() string
	Parties() []string
}

// DPE DistressedPositionsEvent.
type DPE interface {
	events.Event
	MarketID() string
	DistressedParties() []string
	SafeParties() []string
}

// SME SettleMarketEvent.
type SME interface {
	MarketID() string
	SettledPrice() *num.Uint
	PositionFactor() num.Decimal
	TxHash() string
}

// TE TradeEvent.
type TE interface {
	MarketID() string
	IsParty(id string) bool // we don't use this one, but it's to make sure we identify the event correctly
	Trade() vega.Trade
}

// Positions plugin taking settlement data to build positions API data.
type Positions struct {
	*subscribers.Base
	mu      *sync.RWMutex
	data    map[string]map[string]Position
	factors map[string]num.Decimal
}

func NewPositions(ctx context.Context) *Positions {
	return &Positions{
		Base:    subscribers.NewBase(ctx, 10, true),
		mu:      &sync.RWMutex{},
		data:    map[string]map[string]Position{},
		factors: map[string]num.Decimal{},
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
		case DOC:
			p.applyDistressedOrders(te)
		case DPE:
			p.applyDistressedPositions(te)
		case SME:
			p.handleSettleMarket(te)
		case FP:
			p.handleFundingPayments(te)
		case TE:
			p.handleTradeEvent(te)
		}
	}
	p.mu.Unlock()
}

func (p *Positions) handleRegularTrade(e TE) {
	trade := e.Trade()
	if trade.Type == types.TradeTypeNetworkCloseOutBad {
		return
	}
	marketID := e.MarketID()
	partyPos, ok := p.data[marketID]
	if !ok {
		// @TODO should this be done?
		return
	}
	buyerFee, sellerFee := getFeeAmounts(&trade)
	buyer, ok := partyPos[trade.Buyer]
	if !ok {
		buyer = Position{
			Position:            types.NewPosition(marketID, trade.Buyer),
			AverageEntryPriceFP: num.DecimalZero(),
			RealisedPnlFP:       num.DecimalZero(),
			UnrealisedPnlFP:     num.DecimalZero(),
		}
	}
	buyer.setFees(buyerFee)
	seller, ok := partyPos[trade.Seller]
	if !ok {
		seller = Position{
			Position:            types.NewPosition(marketID, trade.Seller),
			AverageEntryPriceFP: num.DecimalZero(),
			RealisedPnlFP:       num.DecimalZero(),
			UnrealisedPnlFP:     num.DecimalZero(),
		}
	}
	seller.setFees(sellerFee)
	partyPos[trade.Buyer] = buyer
	partyPos[trade.Seller] = seller
	p.data[marketID] = partyPos
}

// handle trade event closing distressed parties.
func (p *Positions) handleTradeEvent(e TE) {
	trade := e.Trade()
	if trade.Type != types.TradeTypeNetworkCloseOutBad {
		p.handleRegularTrade(e)
		return
	}
	marketID := e.MarketID()
	partyPos, ok := p.data[marketID]
	if !ok {
		return
	}
	posFactor := num.DecimalOne()
	// keep track of position factors
	if pf, ok := p.factors[marketID]; ok {
		posFactor = pf
	}
	mPrice, _ := num.UintFromString(trade.Price, 10)
	markPriceDec := num.DecimalFromUint(mPrice)
	size := int64(trade.Size)
	pos, ok := partyPos[types.NetworkParty]
	if !ok {
		pos = Position{
			Position:            types.NewPosition(marketID, types.NetworkParty),
			AverageEntryPriceFP: num.DecimalZero(),
			RealisedPnlFP:       num.DecimalZero(),
			UnrealisedPnlFP:     num.DecimalZero(),
		}
	}
	dParty := trade.Seller
	networkFee, otherFee := getFeeAmounts(&trade)
	if trade.Seller == types.NetworkParty {
		size *= -1
		dParty = trade.Buyer
		networkFee, otherFee = otherFee, networkFee
	}
	other, ok := partyPos[dParty]
	if !ok {
		other = Position{
			Position:            types.NewPosition(marketID, dParty),
			AverageEntryPriceFP: num.DecimalZero(),
			RealisedPnlFP:       num.DecimalZero(),
			UnrealisedPnlFP:     num.DecimalZero(),
		}
	}
	other.setFees(otherFee)
	other.ResetSince()
	pos.setFees(networkFee)
	opened, closed := calculateOpenClosedVolume(pos.OpenVolume, size)
	realisedPnlDelta := markPriceDec.Sub(pos.AverageEntryPriceFP).Mul(num.DecimalFromInt64(closed)).Div(posFactor)
	pos.RealisedPnl = pos.RealisedPnl.Add(realisedPnlDelta)
	pos.RealisedPnlFP = pos.RealisedPnlFP.Add(realisedPnlDelta)
	// what was realised is no longer unrealised
	pos.UnrealisedPnl = pos.UnrealisedPnl.Sub(realisedPnlDelta)
	pos.UnrealisedPnlFP = pos.UnrealisedPnlFP.Sub(realisedPnlDelta)
	pos.OpenVolume -= closed

	pos.AverageEntryPriceFP = updateVWAP(pos.AverageEntryPriceFP, pos.OpenVolume, opened, mPrice)
	pos.AverageEntryPrice, _ = num.UintFromDecimal(pos.AverageEntryPriceFP.Round(0))
	pos.OpenVolume += opened
	mtm(&pos, mPrice, posFactor)
	partyPos[types.NetworkParty] = pos
	partyPos[dParty] = other
	p.data[marketID] = partyPos
}

func (p *Positions) handleFundingPayments(e FP) {
	marketID := e.MarketID()
	partyPos, ok := p.data[marketID]
	if !ok {
		return
	}
	payments := e.FundingPayments().Payments
	for _, pay := range payments {
		pos, ok := partyPos[pay.PartyId]
		if !ok {
			continue
		}
		amt, _ := num.DecimalFromString(pay.Amount)
		iAmt, _ := num.IntFromDecimal(amt)
		pos.RealisedPnl = pos.RealisedPnl.Add(amt)
		pos.RealisedPnlFP = pos.RealisedPnlFP.Add(amt)
		// add funding totals
		pos.FundingPaymentAmount.Add(iAmt)
		pos.FundingPaymentAmountSince.Add(iAmt)
		partyPos[pay.PartyId] = pos
	}
	p.data[marketID] = partyPos
}

func (p *Positions) applyDistressedPositions(e DPE) {
	marketID := e.MarketID()
	partyPos, ok := p.data[marketID]
	if !ok {
		return
	}
	for _, party := range e.DistressedParties() {
		if pos, ok := partyPos[party]; ok {
			pos.state = vega.PositionStatus_POSITION_STATUS_DISTRESSED
			partyPos[party] = pos
		}
	}
	for _, party := range e.SafeParties() {
		if pos, ok := partyPos[party]; ok {
			pos.state = vega.PositionStatus_POSITION_STATUS_UNSPECIFIED
			partyPos[party] = pos
		}
	}
	p.data[marketID] = partyPos
}

func (p *Positions) applyDistressedOrders(e DOC) {
	marketID, parties := e.MarketID(), e.Parties()
	partyPos, ok := p.data[marketID]
	if !ok {
		return
	}
	for _, party := range parties {
		if pos, ok := partyPos[party]; ok {
			pos.state = vega.PositionStatus_POSITION_STATUS_ORDERS_CLOSED
			partyPos[party] = pos
		}
	}
	p.data[marketID] = partyPos
}

func (p *Positions) applyLossSocialization(e LSE) {
	iAmt := e.Amount()
	marketID, partyID, amountLoss := e.MarketID(), e.PartyID(), num.DecimalFromInt(iAmt)
	pos, ok := p.data[marketID][partyID]
	if !ok {
		return
	}
	if amountLoss.IsNegative() {
		pos.loss = pos.loss.Add(amountLoss)
	} else {
		pos.adjustment = pos.adjustment.Add(amountLoss)
	}
	if e.IsFunding() {
		// adjust funding amounts if needed.
		pos.FundingPaymentAmount.Add(iAmt)
		pos.FundingPaymentAmountSince.Add(iAmt)
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
	calc.AverageEntryPrice = num.UintZero()
	// realised P&L includes whatever we had in margin account at this point
	dMargin := num.DecimalFromUint(margin)
	calc.RealisedPnl = calc.RealisedPnl.Sub(dMargin)
	calc.RealisedPnlFP = calc.RealisedPnlFP.Sub(dMargin)
	// @TODO average entry price shouldn't be affected(?)
	// the volume now is zero, though, so we'll end up moving this position to storage
	calc.UnrealisedPnlFP = num.DecimalZero()
	calc.AverageEntryPriceFP = num.DecimalZero()
	calc.Position.UpdatedAt = e.Timestamp()
	calc.state = vega.PositionStatus_POSITION_STATUS_CLOSED_OUT
	p.data[mID][tID] = calc
}

func (p *Positions) handleSettleMarket(e SME) {
	market := e.MarketID()
	posFactor := e.PositionFactor()
	// keep track of position factors
	if _, ok := p.factors[market]; !ok {
		p.factors[market] = posFactor
	}
	markPriceDec := num.DecimalFromUint(e.SettledPrice())
	mp, ok := p.data[market]
	if !ok {
		panic(ErrMarketNotFound)
	}
	for pid, pos := range mp {
		openVolumeDec := num.DecimalFromInt64(pos.OpenVolume)

		unrealisedPnl := openVolumeDec.Mul(markPriceDec.Sub(pos.AverageEntryPriceFP)).Div(posFactor).Round(0)
		pos.RealisedPnl = pos.RealisedPnl.Add(unrealisedPnl)
		pos.UnrealisedPnl = num.DecimalZero()
		p.data[market][pid] = pos
	}
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

func (p *Positions) GetStateByMarketAndParty(market, party string) (vega.PositionStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	mp, ok := p.data[market]
	if !ok {
		return vega.PositionStatus_POSITION_STATUS_UNSPECIFIED, nil
	}
	if pos, ok := mp[party]; ok {
		return pos.state, nil
	}
	return vega.PositionStatus_POSITION_STATUS_UNSPECIFIED, nil
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

func (p *Positions) GetPositionStatesByParty(party string) ([]vega.PositionStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// max 1 state per market
	states := make([]vega.PositionStatus, 0, len(p.data))
	for _, parties := range p.data {
		if pos, ok := parties[party]; ok {
			states = append(states, pos.state)
		}
	}
	return states, nil
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
		reset := p.OpenVolume == 0
		pr := t.Price()
		openedVolume, closedVolume := calculateOpenClosedVolume(p.OpenVolume, t.Size())
		_ = closeV(p, closedVolume, pr, e.PositionFactor())
		before := p.OpenVolume
		openV(p, openedVolume, pr)
		// was the position zero, or did the position flip sides?
		if reset || (before < 0 && p.OpenVolume > 0) || (before > 0 && p.OpenVolume < 0) {
			p.ResetSince()
		}
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
	state      vega.PositionStatus
}

func seToProto(e SE) Position {
	return Position{
		Position:            types.NewPosition(e.MarketID(), e.PartyID()),
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
		events.DistressedOrdersClosedEvent,
		events.DistressedPositionsEvent,
		events.SettleMarketEvent,
		events.FundingPaymentsEvent,
		events.TradeEvent,
	}
}

func (p *Position) setFees(fee *feeAmounts) {
	p.TakerFeesPaid.AddSum(fee.taker)
	p.TakerFeesPaidSince.AddSum(fee.taker)
	p.MakerFeesReceived.AddSum(fee.maker)
	p.MakerFeesReceivedSince.AddSum(fee.maker)
	p.FeesPaid.AddSum(fee.other)
	p.FeesPaidSince.AddSum(fee.other)
}
