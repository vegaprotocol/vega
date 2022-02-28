package sqlsubscribers

import (
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
}

type MarketData struct {
	log       *logging.Logger
	store     MarketDataStore
	dbTimeout time.Duration
	vegaTime  time.Time
	seqNum    int
}

func (md *MarketData) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		md.seqNum = 0
		md.vegaTime = e.Time()
	case MarketDataEvent:
		md.seqNum++
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
	md.log.Debug("Received MarketData Event",
		logging.Int64("block", event.BlockNr()),
		logging.String("market", event.MarketData().Market),
	)

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

	record.VegaTime = md.vegaTime
	record.SeqNum = md.seqNum

	return record, nil
}
