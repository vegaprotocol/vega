package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type KeyRotationEvent interface {
	events.Event
	KeyRotation() eventspb.KeyRotation
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/key_rotation_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers KeyRotationStore
type KeyRotationStore interface {
	Upsert(context.Context, *entities.KeyRotation) error
}

type KeyRotation struct {
	subscriber
	store KeyRotationStore
	log   *logging.Logger
}

func NewKeyRotation(store KeyRotationStore, log *logging.Logger) *KeyRotation {
	return &KeyRotation{
		store: store,
		log:   log,
	}
}

func (kr *KeyRotation) Types() []events.Type {
	return []events.Type{events.KeyRotationEvent}
}

func (kr *KeyRotation) Push(ctx context.Context, evt events.Event) error {
	return kr.consume(ctx, evt.(KeyRotationEvent))
}

func (kr *KeyRotation) consume(ctx context.Context, event KeyRotationEvent) error {
	key_rotation := event.KeyRotation()
	record, err := entities.KeyRotationFromProto(&key_rotation, kr.vegaTime)

	if err != nil {
		return errors.Wrap(err, "converting key rotation proto to database entity failed")
	}

	return errors.Wrap(kr.store.Upsert(ctx, record), "Inserting key rotation to SQL store failed")
}
