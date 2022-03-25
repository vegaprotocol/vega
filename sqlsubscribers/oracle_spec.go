package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
)

type OracleSpecEvent interface {
	events.Event
	OracleSpec() oraclespb.OracleSpec
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracle_spec_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers OracleSpecStore
type OracleSpecStore interface {
	Upsert(*entities.OracleSpec) error
}

type OracleSpec struct {
	store    OracleSpecStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewOracleSpec(store OracleSpecStore, log *logging.Logger) *OracleSpec {
	return &OracleSpec{
		store: store,
		log:   log,
	}
}

func (od *OracleSpec) Type() events.Type {
	return events.OracleSpecEvent
}

func (od *OracleSpec) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		od.vegaTime = e.Time()
	case OracleSpecEvent:
		od.consume(e)
	}
}

func (od *OracleSpec) consume(event OracleSpecEvent) {
	spec := event.OracleSpec()
	entity, err := entities.OracleSpecFromProto(&spec, od.vegaTime)
	if err != nil {
		od.log.Error("converting oracle spec to database entity failed", logging.Error(err))
		return
	}

	if err = od.store.Upsert(entity); err != nil {
		od.log.Error("inserting oracle spec to SQL store failed", logging.Error(err))
	}
}
