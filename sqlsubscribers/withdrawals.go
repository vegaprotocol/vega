package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
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

func (w *Withdrawal) Type() events.Type {
	return events.WithdrawalEvent
}

func (w *Withdrawal) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		w.vegaTime = e.Time()
	case WithdrawalEvent:
		w.consume(e)
	}
}

func (w *Withdrawal) consume(event WithdrawalEvent) {
	withdrawal := event.Withdrawal()
	record, err := entities.WithdrawalFromProto(&withdrawal, w.vegaTime)
	if err != nil {
		w.log.Error("converting withdrawal proto to database entity failed", logging.Error(err))
	}

	if err = w.store.Upsert(record); err != nil {
		w.log.Error("Inserting withdrawal to SQL store failed", logging.Error(err))
	}
}
