package sqlsubscribers

import (
	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type ERC20MultiSigSignerAddedEvent interface {
	events.Event
	Proto() eventspb.ERC20MultiSigSignerAdded
}

type ERC20MultiSigSignerRemovedEvent interface {
	events.Event
	Proto() eventspb.ERC20MultiSigSignerRemoved
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/withdrawals_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers WithdrawalStore
type ERC20MultiSigSignerEventStore interface {
	Add(e *entities.ERC20MultiSigSignerEvent) error
}

type ERC20MultiSigSignerEvent struct {
	store ERC20MultiSigSignerEventStore
	log   *logging.Logger
}

func NewERC20MultiSigSignerEvent(store ERC20MultiSigSignerEventStore, log *logging.Logger) *ERC20MultiSigSignerEvent {
	return &ERC20MultiSigSignerEvent{
		store: store,
		log:   log,
	}
}

func (t *ERC20MultiSigSignerEvent) Types() []events.Type {
	return []events.Type{
		events.ERC20MultiSigSignerAddedEvent,
		events.ERC20MultiSigSignerRemovedEvent,
	}
}

func (m *ERC20MultiSigSignerEvent) Push(evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		return nil // we have a timestamp in the event so we don't need to do anything here
	case ERC20MultiSigSignerAddedEvent:
		return m.consumeAddedEvent(e)
	case ERC20MultiSigSignerRemovedEvent:
		return m.consumeRemovedEvent(e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}
}

func (m *ERC20MultiSigSignerEvent) consumeAddedEvent(event ERC20MultiSigSignerAddedEvent) error {
	e := event.Proto()
	record, err := entities.ERC20MultiSigSignerEventFromAddedProto(&e)
	if err != nil {
		return errors.Wrap(err, "converting signer-added proto to database entity failed")
	}
	return m.store.Add(record)
}

func (m *ERC20MultiSigSignerEvent) consumeRemovedEvent(event ERC20MultiSigSignerRemovedEvent) error {
	e := event.Proto()
	records, err := entities.ERC20MultiSigSignerEventFromRemovedProto(&e)
	if err != nil {
		return errors.Wrap(err, "converting signer-added proto to database entity failed")
	}
	for _, r := range records {
		m.store.Add(r)
	}
	return nil
}
