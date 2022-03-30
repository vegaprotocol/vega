package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
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

func (od *OracleSpec) Types() []events.Type {
	return []events.Type{events.OracleSpecEvent}
}

func (od *OracleSpec) Push(evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		od.vegaTime = e.Time()
	case OracleSpecEvent:
		return od.consume(e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}

	return nil
}

func (od *OracleSpec) consume(event OracleSpecEvent) error {
	spec := event.OracleSpec()
	entity, err := entities.OracleSpecFromProto(&spec, od.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting oracle spec to database entity failed")
	}

	return errors.Wrap(od.store.Upsert(entity), "inserting oracle spec to SQL store failed")
}
