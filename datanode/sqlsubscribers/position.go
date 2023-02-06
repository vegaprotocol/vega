// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlsubscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/pkg/errors"
)

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

type PositionStore interface {
	Add(context.Context, entities.Position) error
	GetByMarket(ctx context.Context, marketID string) ([]entities.Position, error)
	GetByMarketAndParty(ctx context.Context, marketID string, partyID string) (entities.Position, error)
	GetByMarketAndParties(ctx context.Context, marketID string, parties []string) ([]entities.Position, error)
	Flush(ctx context.Context) error
}

type Position struct {
	subscriber
	store PositionStore
	log   *logging.Logger
	mutex sync.Mutex
}

func NewPosition(
	store PositionStore,
	log *logging.Logger,
) *Position {
	t := &Position{
		store: store,
		log:   log,
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
	default:
		return errors.Errorf("unknown event type %s", evt.Type().String())
	}
}

func (p *Position) handleOrdersClosedEvent(ctx context.Context, event ordersClosed) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	// @TODO implement this
	positions, err := p.store.GetByMarketAndParties(ctx, event.MarketID, event.Parties())
	if err != nil {
		return errors.Wrap(err, "error getting positions")
	}
	for _, pos := range positions {
		pos.UpdateOrdersClosed()
		// an update of an existing position here can't really fail
		_ = p.updatePosition(ctx, pos)
	}
	return nil
}

func (p *Position) handleTradeEvent(ctx context.Context, event tradeEvent) error {
	trade := event.Trade()
	p.mutex.Lock()
	defer p.mutex.Unlock()
	buyer, seller := p.getPositionsByTrade(ctx, trade)
	buyer.UpdateWithTrade(trade, false)
	// this can't really result in an error...
	_ = p.updatePosition(ctx, buyer)
	seller.UpdateWithTrade(trade, true)
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
	pos.VegaTime = p.vegaTime

	err := p.store.Add(ctx, pos)
	return errors.Wrap(err, "error updating position")
}
