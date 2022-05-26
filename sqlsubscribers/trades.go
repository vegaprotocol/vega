package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type TradeEvent interface {
	events.Event
	Trade() types.Trade
}

type TradesStore interface {
	Add(*entities.Trade) error
	Flush(ctx context.Context) error
}

type TradeSubscriber struct {
	subscriber
	store TradesStore
	log   *logging.Logger
}

func NewTradesSubscriber(store TradesStore, log *logging.Logger) *TradeSubscriber {
	return &TradeSubscriber{
		store: store,
		log:   log,
	}
}

func (ts *TradeSubscriber) Types() []events.Type {
	return []events.Type{events.TradeEvent}
}

func (ts *TradeSubscriber) Flush(ctx context.Context) error {
	return ts.store.Flush(ctx)
}

func (ts *TradeSubscriber) Push(ctx context.Context, evt events.Event) error {
	return ts.consume(evt.(TradeEvent))
}

func (ts *TradeSubscriber) consume(ae TradeEvent) error {
	trade := ae.Trade()
	return errors.Wrap(ts.addTrade(&trade, ts.vegaTime, ae.Sequence()), "failed to consume trade")
}

func (ts *TradeSubscriber) addTrade(t *types.Trade, vegaTime time.Time, blockSeqNumber uint64) error {
	trade, err := entities.TradeFromProto(t, vegaTime, blockSeqNumber)
	if err != nil {
		return errors.Wrap(err, "converting event to trade")
	}

	return errors.Wrap(ts.store.Add(trade), "adding trade to store")
}
