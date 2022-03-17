package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
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

func (m *MarketUpdated) Type() events.Type {
	return events.MarketUpdatedEvent
}

func (m *MarketUpdated) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		m.vegaTime = e.Time()
	case MarketUpdatedEvent:
		m.consume(e)
	}
}

func (m *MarketUpdated) consume(event MarketUpdatedEvent) {
	market := event.Market()
	record, err := entities.NewMarketFromProto(&market, m.vegaTime)

	if err != nil {
		m.log.Error("Converting market proto to database entity failed", logging.Error(err))
		return
	}

	if err = m.store.Upsert(record); err != nil {
		m.log.Error("Updating market to SQL store failed", logging.Error(err))
	}
}
