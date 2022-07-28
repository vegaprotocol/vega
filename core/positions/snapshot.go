// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package positions

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/num"
	"code.vegaprotocol.io/vega/logging"

	"code.vegaprotocol.io/vega/libs/proto"
)

type SnapshotEngine struct {
	*Engine
	pl      types.Payload
	data    []byte
	changed bool
	stopped bool
}

func NewSnapshotEngine(
	log *logging.Logger, config Config, marketID string, broker Broker,
) *SnapshotEngine {
	return &SnapshotEngine{
		Engine:  New(log, config, marketID, broker),
		pl:      types.Payload{},
		changed: true,
		stopped: false,
	}
}

func (e *SnapshotEngine) Changed() bool {
	return e.changed
}

// StopSnapshots is called when the engines respective market no longer exists. We need to stop
// taking snapshots and communicate to the snapshot engine to remove us as a provider.
func (e *SnapshotEngine) StopSnapshots() {
	e.log.Debug("market has been cleared, stopping snapshot production", logging.String("marketid", e.marketID))
	e.stopped = true
}

func (e *SnapshotEngine) RegisterOrder(ctx context.Context, order *types.Order) *MarketPosition {
	e.changed = true
	return e.Engine.RegisterOrder(ctx, order)
}

func (e *SnapshotEngine) UnregisterOrder(ctx context.Context, order *types.Order) *MarketPosition {
	e.changed = true
	return e.Engine.UnregisterOrder(ctx, order)
}

func (e *SnapshotEngine) AmendOrder(ctx context.Context, originalOrder, newOrder *types.Order) *MarketPosition {
	e.changed = true
	return e.Engine.AmendOrder(ctx, originalOrder, newOrder)
}

func (e *SnapshotEngine) UpdateNetwork(ctx context.Context, trade *types.Trade) []events.MarketPosition {
	e.changed = true
	return e.Engine.UpdateNetwork(ctx, trade)
}

func (e *SnapshotEngine) Update(ctx context.Context, trade *types.Trade) []events.MarketPosition {
	e.changed = true
	return e.Engine.Update(ctx, trade)
}

func (e *SnapshotEngine) RemoveDistressed(parties []events.MarketPosition) []events.MarketPosition {
	e.changed = true
	return e.Engine.RemoveDistressed(parties)
}

func (e *SnapshotEngine) UpdateMarkPrice(markPrice *num.Uint) []events.MarketPosition {
	e.changed = true
	return e.Engine.UpdateMarkPrice(markPrice)
}

func (e *SnapshotEngine) Namespace() types.SnapshotNamespace {
	return types.PositionsSnapshot
}

func (e *SnapshotEngine) Keys() []string {
	return []string{e.marketID}
}

func (e *SnapshotEngine) Stopped() bool {
	return e.stopped
}

func (e *SnapshotEngine) HasChanged(k string) bool {
	return true
	// return e.changed
}

func (e *SnapshotEngine) GetState(k string) ([]byte, []types.StateProvider, error) {
	if k != e.marketID {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	state, err := e.serialise()
	return state, nil, err
}

func (e *SnapshotEngine) LoadState(_ context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	var err error
	switch pl := payload.Data.(type) {
	case *types.PayloadMarketPositions:

		// Check the payload is for this market
		if e.marketID != pl.MarketPositions.MarketID {
			return nil, types.ErrUnknownSnapshotType
		}
		e.log.Debug("loading snapshot", logging.Int("positions", len(pl.MarketPositions.Positions)))
		for _, p := range pl.MarketPositions.Positions {
			pos := NewMarketPosition(p.PartyID)
			pos.price = p.Price
			pos.buy = p.Buy
			pos.sell = p.Sell
			pos.size = p.Size
			pos.vwBuyPrice = p.VwBuy
			pos.vwSellPrice = p.VwSell

			e.positionsCpy = append(e.positionsCpy, pos)
			e.positions[p.PartyID] = pos
		}
		e.data, err = proto.Marshal(payload.IntoProto())
		e.changed = false
		return nil, err

	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

// serialise marshal the snapshot state, populating the data field
// with updated values.
func (e *SnapshotEngine) serialise() ([]byte, error) {
	if e.stopped {
		return nil, nil
	}

	if !e.HasChanged(e.pl.Key()) {
		return e.data, nil // we already have what we need
	}

	e.log.Debug("serilaising snapshot", logging.Int("positions", len(e.positionsCpy)))
	positions := make([]*types.MarketPosition, 0, len(e.positionsCpy))

	for _, evt := range e.positionsCpy {
		pos := &types.MarketPosition{
			PartyID: evt.Party(),
			Price:   evt.Price(),
			Buy:     evt.Buy(),
			Sell:    evt.Sell(),
			Size:    evt.Size(),
			VwBuy:   evt.VWBuy(),
			VwSell:  evt.VWSell(),
		}
		positions = append(positions, pos)
	}
	e.pl.Data = &types.PayloadMarketPositions{
		MarketPositions: &types.MarketPositions{
			MarketID:  e.marketID,
			Positions: positions,
		},
	}

	var err error
	e.data, err = proto.Marshal(e.pl.IntoProto())
	if err != nil {
		return nil, err
	}

	e.changed = false

	return e.data, nil
}
