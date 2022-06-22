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

package sqlsubscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/pkg/errors"
)

type positionEventBase interface {
	events.Event
	PartyID() string
	MarketID() string
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

//go:generate go run github.com/golang/mock/mockgen -destination mocks/positions_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers PositionStore
type PositionStore interface {
	Add(context.Context, entities.Position) error
	GetByMarket(ctx context.Context, marketID entities.MarketID) ([]entities.Position, error)
	GetByMarketAndParty(ctx context.Context, marketID entities.MarketID, partyID entities.PartyID) (entities.Position, error)
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

func (t *Position) Types() []events.Type {
	return []events.Type{
		events.SettlePositionEvent,
		events.SettleDistressedEvent,
		events.LossSocializationEvent,
	}
}

func (nl *Position) Flush(ctx context.Context) error {
	err := nl.store.Flush(ctx)
	return errors.Wrap(err, "flushing positions")
}

func (nl *Position) Push(ctx context.Context, evt events.Event) error {
	switch event := evt.(type) {
	case positionSettlement:
		return nl.handlePositionSettlement(ctx, event)
	case lossSocialization:
		return nl.handleLossSocialization(ctx, event)
	case settleDistressed:
		return nl.handleSettleDestressed(ctx, event)
	default:
		return errors.Errorf("unknown event type %s", evt.Type().String())
	}
}

func (ps *Position) handlePositionSettlement(ctx context.Context, event positionSettlement) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	pos := ps.getPosition(ctx, event)
	pos.UpdateWithPositionSettlement(event)
	return ps.updatePosition(ctx, pos)
}

func (ps *Position) handleLossSocialization(ctx context.Context, event lossSocialization) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	pos := ps.getPosition(ctx, event)
	pos.UpdateWithLossSocialization(event)
	return ps.updatePosition(ctx, pos)
}

func (ps *Position) handleSettleDestressed(ctx context.Context, event settleDistressed) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	pos := ps.getPosition(ctx, event)
	pos.UpdateWithSettleDestressed(event)
	return ps.updatePosition(ctx, pos)
}

func (ps *Position) getPosition(ctx context.Context, e positionEventBase) entities.Position {
	mID := entities.NewMarketID(e.MarketID())
	pID := entities.NewPartyID(e.PartyID())

	position, err := ps.store.GetByMarketAndParty(ctx, mID, pID)
	if errors.Is(err, sqlstore.ErrPositionNotFound) {
		return entities.NewEmptyPosition(mID, pID)
	}

	if err != nil {
		// TODO: Can we do something less drastic here? If we can't get existing positions
		//       things are a bit screwed as we'll start writing down wrong aggregates.
		panic("unable to query for existing position")
	}

	return position
}

func (ps *Position) updatePosition(ctx context.Context, pos entities.Position) error {
	pos.VegaTime = ps.vegaTime

	err := ps.store.Add(ctx, pos)
	return errors.Wrap(err, "error updating position")
}
