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

type MarketCreatedEvent interface {
	events.Event
	Market() vega.Market
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/markets_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers MarketsStore
type MarketsStore interface {
	Upsert(context.Context, *entities.Market) error
}

type MarketCreated struct {
	store    MarketsStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewMarketCreated(store MarketsStore, log *logging.Logger) *MarketCreated {
	return &MarketCreated{
		store: store,
		log:   log,
	}
}

func (m *MarketCreated) Types() []events.Type {
	return []events.Type{events.MarketCreatedEvent}
}

func (m *MarketCreated) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		m.vegaTime = e.Time()
	case MarketCreatedEvent:
		return m.consume(ctx, e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}

	return nil
}

func (m *MarketCreated) consume(ctx context.Context, event MarketCreatedEvent) error {
	market := event.Market()
	record, err := entities.NewMarketFromProto(&market, m.vegaTime)

	if err != nil {
		return errors.Wrap(err, "converting market proto to database entity failed")
	}

	return errors.Wrap(m.store.Upsert(ctx, record), "inserting market to SQL store failed:%w")
}
