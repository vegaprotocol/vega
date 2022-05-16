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

type DepositEvent interface {
	events.Event
	Deposit() vega.Deposit
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/deposits_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers DepositStore
type DepositStore interface {
	Upsert(context.Context, *entities.Deposit) error
}

type Deposit struct {
	store    DepositStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewDeposit(store DepositStore, log *logging.Logger) *Deposit {
	return &Deposit{
		store: store,
		log:   log,
	}
}

func (d *Deposit) Types() []events.Type {
	return []events.Type{events.DepositEvent}
}

func (d *Deposit) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		d.vegaTime = e.Time()
	case DepositEvent:
		return d.consume(ctx, e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}

	return nil
}

func (d *Deposit) consume(ctx context.Context, event DepositEvent) error {
	deposit := event.Deposit()
	record, err := entities.DepositFromProto(&deposit, d.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting deposit proto to database entity failed")
	}

	return errors.Wrap(d.store.Upsert(ctx, record), "inserting deposit to SQL store failed")
}
