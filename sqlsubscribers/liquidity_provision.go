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

type LiquidityProvisionEvent interface {
	events.Event
	LiquidityProvision() *vega.LiquidityProvision
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/liquidity_provision_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers LiquidityProvisionStore
type LiquidityProvisionStore interface {
	Upsert(entities.LiquidityProvision) error
	Flush(ctx context.Context) error
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

func (lp *LiquidityProvision) Types() []events.Type {
	return []events.Type{events.LiquidityProvisionEvent}
}

func (lp *LiquidityProvision) Push(evt events.Event) error {
	ctx := context.Background()
	switch e := evt.(type) {
	case TimeUpdateEvent:
		lp.vegaTime = e.Time()
		err := lp.store.Flush(ctx)
		return errors.Wrap(err, "flushing liquidity provisions")
	case LiquidityProvisionEvent:
		return lp.consume(e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}
}

func (lp *LiquidityProvision) consume(event LiquidityProvisionEvent) error {
	provision := event.LiquidityProvision()
	entity, err := entities.LiquidityProvisionFromProto(provision, lp.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting liquidity provision to database entity failed")
	}

	return errors.Wrap(lp.store.Upsert(entity), "inserting liquidity provision to SQL store failed")
}
