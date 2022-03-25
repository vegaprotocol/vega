package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

type DepositEvent interface {
	events.Event
	Deposit() vega.Deposit
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/deposits_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers DepositStore
type DepositStore interface {
	Upsert(*entities.Deposit) error
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

func (d *Deposit) Type() events.Type {
	return events.DepositEvent
}

func (d *Deposit) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		d.vegaTime = e.Time()
	case DepositEvent:
		d.consume(e)
	}
}

func (d *Deposit) consume(event DepositEvent) {
	deposit := event.Deposit()
	record, err := entities.DepositFromProto(&deposit, d.vegaTime)
	if err != nil {
		d.log.Error("converting deposit proto to database entity failed", logging.Error(err))
		return
	}

	if err = d.store.Upsert(record); err != nil {
		d.log.Error("Inserting deposit to SQL store failed", logging.Error(err))
	}
}
