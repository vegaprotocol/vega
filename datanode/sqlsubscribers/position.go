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

package sqlsubscribers

import (
	"context"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
)

type fundingPaymentsEvent interface {
	MarketID() string
	IsParty(id string) bool
	FundingPayments() *eventspb.FundingPayments
}

type tradeEvent interface {
	MarketID() string
	IsParty(id string) bool // we don't use this one, but it's to make sure we identify the event correctly
	Trade() vega.Trade
}

type positionEventBase interface {
	events.Event
	MarketID() string
	PartyID() string
	Timestamp() int64
}

type positionSettlement interface {
	positionEventBase
	Price() *num.Uint
	PositionFactor() num.Decimal
	Trades() []events.TradeSettlement
}

type lossSocialization interface {
	positionEventBase
	Amount() *num.Int
	IsFunding() bool
}

type settleDistressed interface {
	positionEventBase
	Margin() *num.Uint
}

type ordersClosed interface {
	MarketID() string
	Parties() []string
}

type settleMarket interface {
	positionEventBase
	SettledPrice() *num.Uint
	PositionFactor() num.Decimal
}

type distressedPositions interface {
	MarketID() string
	SafeParties() []string
	DistressedParties() []string
}

type PositionStore interface {
	Add(context.Context, entities.Position) error
	GetByMarket(ctx context.Context, marketID string) ([]entities.Position, error)
	GetByMarketAndParty(ctx context.Context, marketID string, partyID string) (entities.Position, error)
	GetByMarketAndParties(ctx context.Context, marketID string, parties []string) ([]entities.Position, error)
	Flush(ctx context.Context) error
}

type MarketSvc interface {
	GetMarketScalingFactor(ctx context.Context, marketID string) (num.Decimal, bool)
	IsSpotMarket(ctx context.Context, marketID string) bool
}

type Position struct {
	subscriber
	store  PositionStore
	mktSvc MarketSvc
	mutex  sync.Mutex
}

func NewPosition(store PositionStore, mktSvc MarketSvc) *Position {
	t := &Position{
		store:  store,
		mktSvc: mktSvc,
	}
	return t
}

func (p *Position) Types() []events.Type {
	return []events.Type{
		events.SettlePositionEvent,
		events.SettleDistressedEvent,
		events.LossSocializationEvent,
		events.SettleMarketEvent,
		events.TradeEvent,
		events.DistressedOrdersClosedEvent,
		events.DistressedPositionsEvent,
		events.FundingPaymentsEvent,
	}
}

func (p *Position) Flush(ctx context.Context) error {
	err := p.store.Flush(ctx)
	return errors.Wrap(err, "flushing positions")
}

func (p *Position) Push(ctx context.Context, evt events.Event) error {
	switch event := evt.(type) {
	case positionSettlement:
		return p.handlePositionSettlement(ctx, event)
	case lossSocialization:
		return p.handleLossSocialization(ctx, event)
	case settleDistressed:
		return p.handleSettleDistressed(ctx, event)
	case settleMarket:
		return p.handleSettleMarket(ctx, event)
	case tradeEvent:
		return p.handleTradeEvent(ctx, event)
	case ordersClosed:
		return p.handleOrdersClosedEvent(ctx, event)
	case distressedPositions:
		return p.handleDistressedPositions(ctx, event)
	case fundingPaymentsEvent:
		return p.handleFundingPayments(ctx, event)
	default:
		return errors.Errorf("unknown event type %s", evt.Type().String())
	}
}

func (p *Position) handleFundingPayments(ctx context.Context, event fundingPaymentsEvent) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	mkt := event.MarketID()
	evt := event.FundingPayments()
	parties := make([]string, 0, len(evt.Payments))
	amounts := make(map[string]*num.Int, len(evt.Payments))
	for _, pay := range evt.Payments {
		// amount is integer, but can be negative
		amt, _ := num.IntFromString(pay.Amount, 10)
		parties = append(parties, pay.PartyId)
		amounts[pay.PartyId] = amt
	}
	positions, err := p.store.GetByMarketAndParties(ctx, mkt, parties)
	if err != nil {
		return err
	}
	for _, pos := range positions {
		amt, ok := amounts[pos.PartyID.String()]
		if !ok {
			// should not be possible, but we may want to return an error here
			continue
		}
		pos.ApplyFundingPayment(amt)
		if err := p.updatePosition(ctx, pos); err != nil {
			return fmt.Errorf("failed to apply funding payment: %w", err)
		}
	}
	return nil
}

func (p *Position) handleDistressedPositions(ctx context.Context, event distressedPositions) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	parties := append(event.DistressedParties(), event.SafeParties()...)
	positions, err := p.store.GetByMarketAndParties(ctx, event.MarketID(), parties)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}
	for _, pos := range positions {
		pos.ToggleDistressedStatus()
		if err := p.updatePosition(ctx, pos); err != nil {
			return fmt.Errorf("failed to update position: %w", err)
		}
	}
	return nil
}

func (p *Position) handleOrdersClosedEvent(ctx context.Context, event ordersClosed) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if sm := p.mktSvc.IsSpotMarket(ctx, event.MarketID()); sm {
		return nil
	}

	positions, err := p.store.GetByMarketAndParties(ctx, event.MarketID(), event.Parties())
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}
	for _, pos := range positions {
		pos.UpdateOrdersClosed()
		if err := p.updatePosition(ctx, pos); err != nil {
			return fmt.Errorf("failed to update position: %w", err)
		}
	}
	return nil
}

func (p *Position) handleTradeEvent(ctx context.Context, event tradeEvent) error {
	trade := event.Trade()
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if sm := p.mktSvc.IsSpotMarket(ctx, trade.MarketId); sm {
		return nil
	}
	sf, ok := p.mktSvc.GetMarketScalingFactor(ctx, trade.MarketId)
	if !ok {
		return fmt.Errorf("failed to get market scaling factor for market %s", trade.MarketId)
	}

	if trade.Type == types.TradeTypeNetworkCloseOutBad {
		pos := p.getNetworkPosition(ctx, trade.MarketId)
		seller := trade.Seller == types.NetworkParty
		pos.UpdateWithTrade(trade, seller, sf)
		return p.updatePosition(ctx, pos)
	}
	buyer, seller := p.getPositionsByTrade(ctx, trade)
	buyer.UpdateWithTrade(trade, false, sf)
	// this can't really result in an error...
	_ = p.updatePosition(ctx, buyer)
	seller.UpdateWithTrade(trade, true, sf)
	return p.updatePosition(ctx, seller)
}

func (p *Position) handlePositionSettlement(ctx context.Context, event positionSettlement) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	pos := p.getPosition(ctx, event)
	pos.UpdateWithPositionSettlement(event)
	return p.updatePosition(ctx, pos)
}

func (p *Position) handleLossSocialization(ctx context.Context, event lossSocialization) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	pos := p.getPosition(ctx, event)
	pos.UpdateWithLossSocialization(event)
	return p.updatePosition(ctx, pos)
}

func (p *Position) handleSettleDistressed(ctx context.Context, event settleDistressed) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	pos := p.getPosition(ctx, event)
	pos.UpdateWithSettleDistressed(event)
	return p.updatePosition(ctx, pos)
}

func (p *Position) handleSettleMarket(ctx context.Context, event settleMarket) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	pos, err := p.store.GetByMarket(ctx, event.MarketID())
	if err != nil {
		return errors.Wrap(err, "error getting positions")
	}
	if sm := p.mktSvc.IsSpotMarket(ctx, event.MarketID()); sm {
		return nil
	}
	for i := range pos {
		pos[i].UpdateWithSettleMarket(event)
		err := p.updatePosition(ctx, pos[i])
		if err != nil {
			return errors.Wrap(err, "error updating position")
		}
	}

	return nil
}

func (p *Position) getPositionsByTrade(ctx context.Context, trade vega.Trade) (buyer entities.Position, seller entities.Position) {
	mID := entities.MarketID(trade.MarketId)
	bID := entities.PartyID(trade.Buyer)
	sID := entities.PartyID(trade.Seller)

	var err error
	buyer, err = p.store.GetByMarketAndParty(ctx, mID.String(), bID.String())
	if errors.Is(err, entities.ErrNotFound) {
		buyer = entities.NewEmptyPosition(mID, bID)
	} else if err != nil {
		// this is a really bad thing to happen :)
		panic("unable to query for existing position")
	}
	seller, err = p.store.GetByMarketAndParty(ctx, mID.String(), sID.String())
	if errors.Is(err, entities.ErrNotFound) {
		seller = entities.NewEmptyPosition(mID, sID)
	} else if err != nil {
		// this is a really bad thing to happen :)
		panic("unable to query for existing position")
	}
	return buyer, seller
}

func (p *Position) getNetworkPosition(ctx context.Context, market string) entities.Position {
	mID := entities.MarketID(market)
	pID := entities.PartyID(types.NetworkParty)
	pos, err := p.store.GetByMarketAndParty(ctx, mID.String(), pID.String())
	if errors.Is(err, entities.ErrNotFound) {
		return entities.NewEmptyPosition(mID, pID)
	}
	if err != nil {
		panic("unable to query existing positions")
	}
	return pos
}

func (p *Position) getPosition(ctx context.Context, e positionEventBase) entities.Position {
	mID := entities.MarketID(e.MarketID())
	pID := entities.PartyID(e.PartyID())

	position, err := p.store.GetByMarketAndParty(ctx, mID.String(), pID.String())
	if errors.Is(err, entities.ErrNotFound) {
		return entities.NewEmptyPosition(mID, pID)
	}

	if err != nil {
		// TODO: Can we do something less drastic here? If we can't get existing positions
		//       things are a bit screwed as we'll start writing down wrong aggregates.
		panic("unable to query for existing position")
	}

	return position
}

func (p *Position) updatePosition(ctx context.Context, pos entities.Position) error {
	if pos.PartyID == entities.PartyID(types.NetworkParty) {
		pos.PendingRealisedPnl = num.DecimalZero()
		pos.RealisedPnl = num.DecimalZero()
		pos.PendingUnrealisedPnl = num.DecimalZero()
		pos.UnrealisedPnl = num.DecimalZero()
	}
	pos.VegaTime = p.vegaTime

	err := p.store.Add(ctx, pos)
	return errors.Wrap(err, "error updating position")
}

func (p *Position) Name() string {
	return "Position"
}
