package sqlsubscribers

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
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

func (ts *TradeSubscriber) Type() events.Type {
	return events.TradeEvent
}

func (ts *TradeSubscriber) Push(evt events.Event) {

	switch e := evt.(type) {
	case TimeUpdateEvent:
		ts.sequenceNum = evt.Sequence()
		ts.vegaTime = e.Time()
		ts.store.OnTimeUpdateEvent(evt.Context())
	case TradeEvent:
		ts.sequenceNum = evt.Sequence()
		ts.consume(e)
	default:
		ts.log.Panic("Unknown event type in trade subscriber",
			logging.String("Type", e.Type().String()))
	}
}

func (ts *TradeSubscriber) consume(ae TradeEvent) error {
	trade := ae.Trade()
	err := ts.addTrade(&trade, ts.vegaTime, ts.sequenceNum)

	if err != nil {
		return fmt.Errorf("failed to consume trade:%w", err)
	}

	return nil
}

func (ts *TradeSubscriber) addTrade(t *types.Trade, vegaTime time.Time, blockSeqNumber uint64) error {
	trade, err := entities.TradeFromProto(t, vegaTime, blockSeqNumber)
	if err != nil {
		return fmt.Errorf("converting event to trade:%w", err)
	}

	err = ts.store.Add(trade)
	if err != nil {
		return fmt.Errorf("adding trade to store: %w", err)
	}

	return nil
}
