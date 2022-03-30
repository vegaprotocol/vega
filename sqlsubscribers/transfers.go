package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
)

type TransferEvent interface {
	events.Event
	TransferFunds() eventspb.Transfer
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_store_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers TransferStore
type TransferStore interface {
	Upsert(transfer *entities.Transfer) error
}

type AccountSource interface {
	Obtain(a *entities.Account) error
	GetByID(id int64) (entities.Account, error)
}

type Transfer struct {
	store         TransferStore
	accountSource AccountSource
	log           *logging.Logger
	vegaTime      time.Time
}

func NewTransfer(store TransferStore, accountSource AccountSource, log *logging.Logger) *Transfer {
	return &Transfer{
		store:         store,
		accountSource: accountSource,
		log:           log,
	}
}

func (rf *Transfer) Types() []events.Type {
	return []events.Type{events.TransferEvent}
}

func (rf *Transfer) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		rf.vegaTime = e.Time()
	case TransferEvent:
		rf.consume(e)
	}
}

func (rf *Transfer) consume(event TransferEvent) {

	transfer := event.TransferFunds()
	record, err := entities.TransferFromProto(&transfer, rf.vegaTime, rf.accountSource)
	if err != nil {
		rf.log.Error("converting transfer proto to database entity failed", logging.Error(err))
	}

	if err = rf.store.Upsert(record); err != nil {
		rf.log.Error("Inserting transfer into to SQL store failed", logging.Error(err))
	}
}
