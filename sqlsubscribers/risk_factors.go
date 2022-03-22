package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

type RiskFactorEvent interface {
	events.Event
	RiskFactor() vega.RiskFactor
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_factor_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers RiskFactorStore
type RiskFactorStore interface {
	Upsert(*entities.RiskFactor) error
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

func (rf *RiskFactor) Type() events.Type {
	return events.RiskFactorEvent
}

func (rf *RiskFactor) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		rf.vegaTime = e.Time()
	case RiskFactorEvent:
		rf.consume(e)
	}
}

func (rf *RiskFactor) consume(event RiskFactorEvent) {
	riskFactor := event.RiskFactor()
	record, err := entities.RiskFactorFromProto(&riskFactor, rf.vegaTime)
	if err != nil {
		rf.log.Error("converting risk factor proto to database entity failed", logging.Error(err))
	}

	if err = rf.store.Upsert(record); err != nil {
		rf.log.Error("Inserting risk factor to SQL store failed", logging.Error(err))
	}
}
