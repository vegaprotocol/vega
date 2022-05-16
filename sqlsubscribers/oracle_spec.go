package sqlsubscribers

import (
	"context"

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
	Upsert(context.Context, *entities.OracleSpec) error
}

type OracleSpec struct {
	subscriber
	store OracleSpecStore
	log   *logging.Logger
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

func (od *OracleSpec) Push(ctx context.Context, evt events.Event) error {
	return od.consume(ctx, evt.(OracleSpecEvent))
}

func (od *OracleSpec) consume(ctx context.Context, event OracleSpecEvent) error {
	spec := event.OracleSpec()
	entity, err := entities.OracleSpecFromProto(&spec, od.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting oracle spec to database entity failed")
	}

	return errors.Wrap(od.store.Upsert(ctx, entity), "inserting oracle spec to SQL store failed")
}
