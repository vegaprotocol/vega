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

package service

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/libs/slice"
	"code.vegaprotocol.io/vega/logging"
)

type tradeStore interface {
	Flush(ctx context.Context) ([]*entities.Trade, error)
	Add(t *entities.Trade) error
	List(context.Context, []entities.MarketID, []entities.PartyID, []entities.OrderID, entities.CursorPagination, entities.DateRange) ([]entities.Trade, entities.PageInfo, error)
	GetLastTradeByMarket(ctx context.Context, market string) ([]entities.Trade, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Trade, error)
}

type Trade struct {
	store    tradeStore
	observer utils.Observer[*entities.Trade]
}

func NewTrade(store tradeStore, log *logging.Logger) *Trade {
	return &Trade{
		store:    store,
		observer: utils.NewObserver[*entities.Trade]("trade", log, 0, 0),
	}
}

func (t *Trade) Flush(ctx context.Context) error {
	flushed, err := t.store.Flush(ctx)
	if err != nil {
		return err
	}
	t.observer.Notify(flushed)
	return nil
}

func (t *Trade) Add(trade *entities.Trade) error {
	return t.store.Add(trade)
}

func (t *Trade) List(ctx context.Context,
	marketIDs []entities.MarketID,
	partyIDs []entities.PartyID,
	orderIDs []entities.OrderID,
	pagination entities.CursorPagination,
	dateRange entities.DateRange,
) ([]entities.Trade, entities.PageInfo, error) {
	return t.store.List(ctx, marketIDs, partyIDs, orderIDs, pagination, dateRange)
}

func (t *Trade) GetLastTradeByMarket(ctx context.Context, market string) ([]entities.Trade, error) {
	return t.store.GetLastTradeByMarket(ctx, market)
}

func (t *Trade) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Trade, error) {
	return t.store.GetByTxHash(ctx, txHash)
}

func (t *Trade) Observe(ctx context.Context, retries int, marketIDs []string, partyIDs []string) (<-chan []*entities.Trade, uint64) {
	ch, ref := t.observer.Observe(ctx,
		retries,
		func(trade *entities.Trade) bool {
			// match market filter if any, or if no filter is provided
			marketsOk := len(marketIDs) == 0 || slice.Contains(marketIDs, trade.MarketID.String())
			// match party filter if any, or if no filter is provided
			partiesOk := len(partyIDs) == 0 || slice.Contains(partyIDs, trade.Buyer.String()) || slice.Contains(partyIDs, trade.Seller.String())

			return marketsOk && partiesOk
		})
	return ch, ref
}
