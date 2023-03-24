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

package service

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/logging"
)

type tradeStore interface {
	Flush(ctx context.Context) ([]*entities.Trade, error)
	Add(t *entities.Trade) error
	List(context.Context, entities.MarketID, entities.PartyID, entities.OrderID, entities.CursorPagination, entities.DateRange) ([]entities.Trade, entities.PageInfo, error)
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

func (t *Trade) Observe(ctx context.Context, retries int, marketID *string, partyID *string) (<-chan []*entities.Trade, uint64) {
	ch, ref := t.observer.Observe(ctx,
		retries,
		func(trade *entities.Trade) bool {
			return (marketID == nil || *marketID == trade.MarketID.String()) &&
				(partyID == nil || *partyID == trade.Buyer.String() || *partyID == trade.Seller.String())
		})
	return ch, ref
}
