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

type RiskFactorEvent interface {
	events.Event
	RiskFactor() vega.RiskFactor
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_factor_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers RiskFactorStore
type RiskFactorStore interface {
	Upsert(context.Context, *entities.RiskFactor) error
}

type RiskFactor struct {
	store    RiskFactorStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewRiskFactor(store RiskFactorStore, log *logging.Logger) *RiskFactor {
	return &RiskFactor{
		store: store,
		log:   log,
	}
}

func (rf *RiskFactor) Types() []events.Type {
	return []events.Type{events.RiskFactorEvent}
}

func (rf *RiskFactor) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		rf.vegaTime = e.Time()
	case RiskFactorEvent:
		return rf.consume(ctx, e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}

	return nil
}

func (rf *RiskFactor) consume(ctx context.Context, event RiskFactorEvent) error {
	riskFactor := event.RiskFactor()
	record, err := entities.RiskFactorFromProto(&riskFactor, rf.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting risk factor proto to database entity failed")
	}

	return errors.Wrap(rf.store.Upsert(ctx, record), "inserting risk factor to SQL store failed")
}
