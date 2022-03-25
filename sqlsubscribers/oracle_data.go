package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
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

func (od *OracleData) Type() events.Type {
	return events.OracleDataEvent
}

func (od *OracleData) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		od.vegaTime = e.Time()
	case OracleDataEvent:
		od.consume(e)
	}
}

func (od *OracleData) consume(event OracleDataEvent) {
	data := event.OracleData()
	entity, err := entities.OracleDataFromProto(&data, od.vegaTime)
	if err != nil {
		od.log.Error("converting oracle data proto to database entity failed", logging.Error(err))
		return
	}

	if err = od.store.Add(entity); err != nil {
		od.log.Error("inserting oracle data proto to SQL store failed", logging.Error(err))
	}
}
