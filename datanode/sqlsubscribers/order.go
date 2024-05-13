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
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/pkg/errors"
)

type MarketDepthService interface {
	AddOrder(order *types.Order, vegaTime time.Time, sequenceNumber uint64)
	PublishAtEndOfBlock()
}

type OrderEvent interface {
	events.Event
	Order() *vega.Order
}

type ExpiredOrdersEvent interface {
	events.Event
	MarketID() string
	OrderIDs() []string
}

type CancelledOrdersEvent interface {
	ExpiredOrdersEvent
	PartyID() string
}

type OrderStore interface {
	Add(entities.Order) error
	Flush(ctx context.Context) error
	GetByMarketAndID(ctx context.Context, marketIDstr string, orderIDs []string) ([]entities.Order, error)
}

type Order struct {
	subscriber
	store        OrderStore
	depthService MarketDepthService
	// the store uses the batcher type which could be used as a cache, provided we know the
	// version and vegatime type for the orders we need to persist. This isn't the case
	// but orders are ingested here, so we can cache them here, of course.
	cache map[entities.OrderID]entities.Order
}

func NewOrder(store OrderStore, depthService MarketDepthService) *Order {
	return &Order{
		store:        store,
		depthService: depthService,
		cache:        map[entities.OrderID]entities.Order{},
	}
}

func (os *Order) Types() []events.Type {
	return []events.Type{
		events.OrderEvent,
		events.ExpiredOrdersEvent,
		events.EndBlockEvent,
		events.CancelledOrdersEvent,
	}
}

func (os *Order) Push(ctx context.Context, evt events.Event) error {
	switch evt.Type() {
	case events.OrderEvent:
		return os.consume(evt.(OrderEvent), evt.Sequence())
	case events.ExpiredOrdersEvent:
		return os.expired(ctx, evt.(ExpiredOrdersEvent), evt.Sequence())
	case events.EndBlockEvent:
		os.consumeEndBlock()
	case events.CancelledOrdersEvent:
		return os.cancelled(ctx, evt.(CancelledOrdersEvent), evt.Sequence())
	}
	return nil
}

func (os *Order) Flush(ctx context.Context) error {
	// clear cache
	os.cache = map[entities.OrderID]entities.Order{}
	return os.store.Flush(ctx)
}

func (os *Order) expired(ctx context.Context, eo ExpiredOrdersEvent, seqNum uint64) error {
	orders, err := os.store.GetByMarketAndID(ctx, eo.MarketID(), eo.OrderIDs())
	if err != nil {
		return err
	}
	txHash := entities.TxHash(eo.TxHash())
	for _, o := range orders {
		o.Status = entities.OrderStatusExpired
		o.SeqNum = seqNum
		o.UpdatedAt = os.vegaTime
		o.VegaTime = os.vegaTime
		o.TxHash = txHash

		// to the depth service
		torder, err := types.OrderFromProto(o.ToProto())
		if err != nil {
			panic(err)
		}
		os.depthService.AddOrder(torder, os.vegaTime, seqNum)

		if err := os.persist(o); err != nil {
			return errors.Wrap(os.store.Add(o), "adding order to database")
		}
		// the next order will be insterted as though it was the next event on the bus, with a new sequence number:
		seqNum++
	}
	return nil
}

func (os *Order) cancelled(ctx context.Context, co CancelledOrdersEvent, seqNum uint64) error {
	allIDs := co.OrderIDs()
	ids := make([]string, 0, len(allIDs))
	orders := make([]entities.Order, 0, len(allIDs))
	for _, id := range allIds {
		k := entities.OrderID(id)
		if o, ok := os.cache[k]; ok {
			orders = append(orders, o)
		} else {
			ids = append(ids, id)
		}
	}
	ncOrders, err := os.store.GetByMarketAndID(ctx, co.MarketID(), ids)
	if err != nil {
		return err
	}
	orders = append(orders, ncOrders...)
	txHash := entities.TxHash(co.TxHash())
	for _, o := range orders {
		o.Status = entities.OrderStatusCancelled
		o.SeqNum = seqNum
		o.UpdatedAt = os.vegaTime
		o.VegaTime = os.vegaTime
		o.TxHash = txHash

		torder, err := types.OrderFromProto(o.ToProto())
		if err != nil {
			panic(err)
		}
		os.depthService.AddOrder(torder, os.vegaTime, seqNum)

		if err := os.persist(o); err != nil {
			return errors.Wrap(err, "adding order to database")
		}
		seqNum++
	}
	return nil
}

func (os *Order) consume(oe OrderEvent, seqNum uint64) error {
	protoOrder := oe.Order()

	order, err := entities.OrderFromProto(protoOrder, seqNum, entities.TxHash(oe.TxHash()))
	if err != nil {
		return errors.Wrap(err, "deserializing order")
	}
	order.VegaTime = os.vegaTime

	// then publish to the market depthService
	torder, err := types.OrderFromProto(oe.Order())
	if err != nil {
		panic(err)
	}
	os.depthService.AddOrder(torder, os.vegaTime, seqNum)

	return errors.Wrap(os.persist(order), "adding order to database")
}

func (os *Order) consumeEndBlock() {
	os.depthService.PublishAtEndOfBlock()
}

func (os *Order) persist(o entities.Order) error {
	os.cache[o.ID] = o
	return os.store.Add(o)
}
