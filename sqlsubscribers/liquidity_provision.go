package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

type LiquidityProvisionEvent interface {
	events.Event
	LiquidityProvision() *vega.LiquidityProvision
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/liquidity_provision_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers LiquidityProvisionStore
type LiquidityProvisionStore interface {
	Upsert(*entities.LiquidityProvision) error
}

type LiquidityProvision struct {
	store    LiquidityProvisionStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewLiquidityProvision(store LiquidityProvisionStore, log *logging.Logger) *LiquidityProvision {
	return &LiquidityProvision{
		store: store,
		log:   log,
	}
}

func (lp *LiquidityProvision) Type() events.Type {
	return events.LiquidityProvisionEvent
}

func (lp *LiquidityProvision) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		lp.vegaTime = e.Time()
	case LiquidityProvisionEvent:
		lp.consume(e)
	}
}

func (lp *LiquidityProvision) consume(event LiquidityProvisionEvent) {
	provision := event.LiquidityProvision()
	entity, err := entities.LiquidityProvisionFromProto(provision, lp.vegaTime)
	if err != nil {
		lp.log.Error("converting liquidity provision to database entity failed", logging.Error(err))
		return
	}

	if err = lp.store.Upsert(entity); err != nil {
		lp.log.Error("inserting liquidity provision to SQL store failed", logging.Error(err))
	}
}
