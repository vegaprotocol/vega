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
	"code.vegaprotocol.io/vega/libs/slice"
	"code.vegaprotocol.io/vega/logging"
)

//nolint:interfacebloat
type orderStore interface {
	Flush(ctx context.Context) ([]entities.Order, error)
	Add(o entities.Order) error
	GetAll(ctx context.Context) ([]entities.Order, error)
	GetOrder(ctx context.Context, orderID string, version *int32) (entities.Order, error)
	GetByMarketAndID(ctx context.Context, marketIDstr string, orderIDs []string) ([]entities.Order, error)
	GetAllVersionsByOrderID(ctx context.Context, id string, p entities.OffsetPagination) ([]entities.Order, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Order, error)
	GetLiveOrders(ctx context.Context) ([]entities.Order, error)
	ListOrderVersions(ctx context.Context, orderIDStr string, p entities.CursorPagination) ([]entities.Order, entities.PageInfo, error)
	ListOrders(ctx context.Context, p entities.CursorPagination, filter entities.OrderFilter) ([]entities.Order, entities.PageInfo, error)
}

type Order struct {
	store    orderStore
	observer utils.Observer[entities.Order]
}

func NewOrder(store orderStore, log *logging.Logger) *Order {
	return &Order{
		store:    store,
		observer: utils.NewObserver[entities.Order]("order", log, 0, 0),
	}
}

func (o *Order) ObserveOrders(ctx context.Context, retries int, markets []string, parties []string, includeLiquidity bool) (<-chan []entities.Order, uint64) {
	ch, ref := o.observer.Observe(ctx,
		retries,
		func(o entities.Order) bool {
			marketOk := slice.Contains(markets, o.MarketID.String())
			partyOk := slice.Contains(parties, o.PartyID.String())
			liqOrder := (o.LpID != nil && includeLiquidity) || !includeLiquidity
			return marketOk && partyOk && liqOrder
		})
	return ch, ref
}

func (o *Order) Flush(ctx context.Context) error {
	flushed, err := o.store.Flush(ctx)
	if err != nil {
		return err
	}
	o.observer.Notify(flushed)
	return nil
}

func (o *Order) Add(order entities.Order) error {
	return o.store.Add(order)
}

func (o *Order) GetAll(ctx context.Context) ([]entities.Order, error) {
	return o.store.GetAll(ctx)
}

func (o *Order) GetOrder(ctx context.Context, orderID string, version *int32) (entities.Order, error) {
	return o.store.GetOrder(ctx, orderID, version)
}

func (o *Order) GetByMarketAndID(ctx context.Context, marketIDstr string, orderIDs []string) ([]entities.Order, error) {
	return o.store.GetByMarketAndID(ctx, marketIDstr, orderIDs)
}

func (o *Order) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Order, error) {
	return o.store.GetByTxHash(ctx, txHash)
}

func (o *Order) GetAllVersionsByOrderID(ctx context.Context, id string, p entities.OffsetPagination) ([]entities.Order, error) {
	return o.store.GetAllVersionsByOrderID(ctx, id, p)
}

func (o *Order) GetLiveOrders(ctx context.Context) ([]entities.Order, error) {
	return o.store.GetLiveOrders(ctx)
}

func (o *Order) ListOrders(ctx context.Context, p entities.CursorPagination, filter entities.OrderFilter,
) ([]entities.Order, entities.PageInfo, error) {
	return o.store.ListOrders(ctx, p, filter)
}

func (o *Order) ListOrderVersions(ctx context.Context, orderID string, p entities.CursorPagination) ([]entities.Order, entities.PageInfo, error) {
	return o.store.ListOrderVersions(ctx, orderID, p)
}
