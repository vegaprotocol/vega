// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
)

type KeyRotationEvent interface {
	events.Event
	KeyRotation() eventspb.KeyRotation
}

type KeyRotationStore interface {
	Upsert(context.Context, *entities.KeyRotation) error
}

type KeyRotation struct {
	subscriber
	store KeyRotationStore
}

func NewKeyRotation(store KeyRotationStore) *KeyRotation {
	return &KeyRotation{
		store: store,
	}
}

func (kr *KeyRotation) Types() []events.Type {
	return []events.Type{events.KeyRotationEvent}
}

func (kr *KeyRotation) Push(ctx context.Context, evt events.Event) error {
	return kr.consume(ctx, evt.(KeyRotationEvent))
}

func (kr *KeyRotation) consume(ctx context.Context, event KeyRotationEvent) error {
	keyRotation := event.KeyRotation()
	record, err := entities.KeyRotationFromProto(&keyRotation, entities.TxHash(event.TxHash()), kr.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting key rotation proto to database entity failed")
	}

	return errors.Wrap(kr.store.Upsert(ctx, record), "Inserting key rotation to SQL store failed")
}
