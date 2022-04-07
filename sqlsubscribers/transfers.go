package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type TransferEvent interface {
	events.Event
	TransferFunds() eventspb.Transfer
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_store_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers TransferStore
type TransferStore interface {
	Upsert(ctx context.Context, transfer *entities.Transfer) error
}

type AccountSource interface {
	Obtain(ctx context.Context, a *entities.Account) error
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

func (rf *Transfer) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		rf.vegaTime = e.Time()
	case TransferEvent:
		return rf.consume(ctx, e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}

	return nil
}

func (rf *Transfer) consume(ctx context.Context, event TransferEvent) error {

	transfer := event.TransferFunds()
	record, err := entities.TransferFromProto(ctx, &transfer, rf.vegaTime, rf.accountSource)
	if err != nil {
		return errors.Wrap(err, "converting transfer proto to database entity failed")
	}

	return errors.Wrap(rf.store.Upsert(ctx, record), "inserting transfer into to SQL store failed")
}
