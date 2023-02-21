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

type OrderStore interface {
	Add(entities.Order) error
	Flush(ctx context.Context) error
	GetByMarketAndID(ctx context.Context, marketIDstr string, orderIDs []string) ([]entities.Order, error)
}

type Order struct {
	subscriber
	store        OrderStore
	depthService MarketDepthService
}

func NewOrder(store OrderStore, depthService MarketDepthService) *Order {
	return &Order{
		store:        store,
		depthService: depthService,
	}
}

func (os *Order) Types() []events.Type {
	return []events.Type{
		events.OrderEvent,
		events.ExpiredOrdersEvent,
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
	}
	return nil
}

func (os *Order) Flush(ctx context.Context) error {
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

		if err := os.store.Add(o); err != nil {
			return errors.Wrap(os.store.Add(o), "adding order to database")
		}
		// the next order will be insterted as though it was the next event on the bus, with a new sequence number:
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

	return errors.Wrap(os.store.Add(order), "adding order to database")
}

func (os *Order) consumeEndBlock() {
	os.depthService.PublishAtEndOfBlock()
}
