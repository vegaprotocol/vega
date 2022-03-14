package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

type MarketCreatedEvent interface {
	events.Event
	Market() vega.Market
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/markets_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers MarketsStore
type MarketsStore interface {
	Upsert(*entities.Market) error
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

func (m *MarketCreated) Type() events.Type {
	return events.MarketCreatedEvent
}

func (m *MarketCreated) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		m.vegaTime = e.Time()
	case MarketCreatedEvent:
		m.consume(e)
	}
}

func (m *MarketCreated) consume(event MarketCreatedEvent) {
	m.log.Debug("Received MarketCreatedEvent",
		logging.Int64("block", event.BlockNr()),
		logging.String("market-id", event.Market().Id),
	)

	market := event.Market()
	record, err := entities.NewMarketFromProto(&market, m.vegaTime)

	if err != nil {
		m.log.Error("Converting market proto to database entity failed", logging.Error(err))
		return
	}

	if err = m.store.Upsert(record); err != nil {
		m.log.Error("Inserting market to SQL store failed", logging.Error(err))
	}
}
