package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type MarketUpdatedEvent interface {
	events.Event
	Market() vega.Market
}

type MarketUpdated struct {
	store    MarketsStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewMarketUpdated(store MarketsStore, log *logging.Logger) *MarketUpdated {
	return &MarketUpdated{
		store: store,
		log:   log,
	}
}

func (m *MarketUpdated) Types() []events.Type {
	return []events.Type{events.MarketUpdatedEvent}
}

func (m *MarketUpdated) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		m.vegaTime = e.Time()
	case MarketUpdatedEvent:
		return m.consume(ctx, e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}

	return nil
}

func (m *MarketUpdated) consume(ctx context.Context, event MarketUpdatedEvent) error {
	market := event.Market()
	record, err := entities.NewMarketFromProto(&market, m.vegaTime)

	if err != nil {
		return errors.Wrap(err, "converting market proto to database entity failed")
	}

	return errors.Wrap(m.store.Upsert(ctx, record), "updating market to SQL store failed")
}
