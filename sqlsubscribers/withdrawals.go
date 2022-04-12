package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type WithdrawalEvent interface {
	events.Event
	Withdrawal() vega.Withdrawal
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/withdrawals_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers WithdrawalStore
type WithdrawalStore interface {
	Upsert(*entities.Withdrawal) error
}

type Withdrawal struct {
	store    WithdrawalStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewWithdrawal(store WithdrawalStore, log *logging.Logger) *Withdrawal {
	return &Withdrawal{
		store: store,
		log:   log,
	}
}

func (w *Withdrawal) Types() []events.Type {
	return []events.Type{events.WithdrawalEvent}
}

func (w *Withdrawal) Push(evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		w.vegaTime = e.Time()
	case WithdrawalEvent:
		return w.consume(e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}

	return nil
}

func (w *Withdrawal) consume(event WithdrawalEvent) error {
	withdrawal := event.Withdrawal()
	record, err := entities.WithdrawalFromProto(&withdrawal, w.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting withdrawal proto to database entity failed")
	}

	return errors.Wrap(w.store.Upsert(record), "inserting withdrawal to SQL store failed")
}
