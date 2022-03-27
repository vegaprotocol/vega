package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

type MarketDataEvent interface {
	events.Event
	MarketData() types.MarketData
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_data_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers MarketDataStore
type MarketDataStore interface {
	Add(*entities.MarketData) error
	OnTimeUpdateEvent(context.Context) error
}

type MarketData struct {
	log       *logging.Logger
	store     MarketDataStore
	dbTimeout time.Duration
	vegaTime  time.Time
	seqNum    uint64
}

func (md *MarketData) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		md.vegaTime = e.Time()
		md.store.OnTimeUpdateEvent(e.Context())
	case MarketDataEvent:
		md.seqNum = e.Sequence()
		md.consume(e)
	default:
		md.log.Error("Unknown event type in transfer response subscriber",
			logging.String("type", e.Type().String()))
	}
}

func (md *MarketData) Type() events.Type {
	return events.MarketDataEvent
}

func NewMarketData(store MarketDataStore, log *logging.Logger, dbTimeout time.Duration) *MarketData {
	return &MarketData{
		log:       log,
		store:     store,
		dbTimeout: dbTimeout,
	}
}

func (md *MarketData) consume(event MarketDataEvent) {
	var record *entities.MarketData
	var err error
	mdProto := event.MarketData()

	if record, err = md.convertMarketDataProto(&mdProto); err != nil {
		md.log.Error("Converting market data proto for persistence failed", logging.Error(err))
		return
	}

	if err := md.store.Add(record); err != nil {
		md.log.Error("Inserting market data to SQL store failed.", logging.Error(err))
	}
}

func (md *MarketData) convertMarketDataProto(data *types.MarketData) (*entities.MarketData, error) {
	record, err := entities.MarketDataFromProto(data)
	if err != nil {
		return nil, err
	}

	record.SyntheticTime = md.vegaTime.Add(time.Duration(record.SeqNum) * time.Microsecond)
	record.VegaTime = md.vegaTime
	record.SeqNum = md.seqNum

	return record, nil
}
