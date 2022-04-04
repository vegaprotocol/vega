package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type OracleDataEvent interface {
	events.Event
	OracleData() oraclespb.OracleData
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracle_data_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers OracleDataStore
type OracleDataStore interface {
	Add(*entities.OracleData) error
}

type OracleData struct {
	store    OracleDataStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewOracleData(store OracleDataStore, log *logging.Logger) *OracleData {
	return &OracleData{
		store: store,
		log:   log,
	}
}

func (od *OracleData) Types() []events.Type {
	return []events.Type{events.OracleDataEvent}
}

func (od *OracleData) Push(evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		od.vegaTime = e.Time()
	case OracleDataEvent:
		return od.consume(e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}

	return nil
}

func (od *OracleData) consume(event OracleDataEvent) error {
	data := event.OracleData()
	entity, err := entities.OracleDataFromProto(&data, od.vegaTime)
	if err != nil {
		errors.Wrap(err, "converting oracle data proto to database entity failed")
	}

	return errors.Wrap(od.store.Add(entity), "inserting oracle data proto to SQL store failed")
}
