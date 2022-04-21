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
	OnTimeUpdateEvent(ctx context.Context) error
}

type TradeSubscriber struct {
	store       TradesStore
	log         *logging.Logger
	vegaTime    time.Time
	sequenceNum uint64
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

func (ts *TradeSubscriber) Push(ctx context.Context, evt events.Event) error {

	switch e := evt.(type) {
	case TimeUpdateEvent:
		ts.sequenceNum = evt.Sequence()
		ts.vegaTime = e.Time()
		ts.store.OnTimeUpdateEvent(ctx)
	case TradeEvent:
		ts.sequenceNum = evt.Sequence()
		return ts.consume(e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}

	return nil
}

func (ts *TradeSubscriber) consume(ae TradeEvent) error {
	trade := ae.Trade()
	return errors.Wrap(ts.addTrade(&trade, ts.vegaTime, ts.sequenceNum), "failed to consume trade")
}

func (ts *TradeSubscriber) addTrade(t *types.Trade, vegaTime time.Time, blockSeqNumber uint64) error {
	trade, err := entities.TradeFromProto(t, vegaTime, blockSeqNumber)
	if err != nil {
		return errors.Wrap(err, "converting event to trade")
	}

	return errors.Wrap(ts.store.Add(trade), "adding trade to store")
}
