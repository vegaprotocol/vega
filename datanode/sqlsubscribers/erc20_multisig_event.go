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

type ERC20MultiSigSignerAddedEvent interface {
	events.Event
	Proto() eventspb.ERC20MultiSigSignerAdded
}

type ERC20MultiSigSignerRemovedEvent interface {
	events.Event
	Proto() eventspb.ERC20MultiSigSignerRemoved
}

type ERC20MultiSigSignerEventStore interface {
	Add(ctx context.Context, e *entities.ERC20MultiSigSignerEvent) error
}

type ERC20MultiSigSignerEvent struct {
	subscriber
	store ERC20MultiSigSignerEventStore
}

func NewERC20MultiSigSignerEvent(store ERC20MultiSigSignerEventStore) *ERC20MultiSigSignerEvent {
	return &ERC20MultiSigSignerEvent{
		store: store,
	}
}

func (m *ERC20MultiSigSignerEvent) Types() []events.Type {
	return []events.Type{
		events.ERC20MultiSigSignerAddedEvent,
		events.ERC20MultiSigSignerRemovedEvent,
	}
}

func (m *ERC20MultiSigSignerEvent) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case ERC20MultiSigSignerAddedEvent:
		return m.consumeAddedEvent(ctx, e)
	case ERC20MultiSigSignerRemovedEvent:
		return m.consumeRemovedEvent(ctx, e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}
}

func (m *ERC20MultiSigSignerEvent) consumeAddedEvent(ctx context.Context, event ERC20MultiSigSignerAddedEvent) error {
	e := event.Proto()
	record, err := entities.ERC20MultiSigSignerEventFromAddedProto(&e, entities.TxHash(event.TxHash()))
	if err != nil {
		return errors.Wrap(err, "converting signer-added proto to database entity failed")
	}
	return m.store.Add(ctx, record)
}

func (m *ERC20MultiSigSignerEvent) consumeRemovedEvent(ctx context.Context, event ERC20MultiSigSignerRemovedEvent) error {
	e := event.Proto()
	records, err := entities.ERC20MultiSigSignerEventFromRemovedProto(&e, entities.TxHash(event.TxHash()))
	if err != nil {
		return errors.Wrap(err, "converting signer-added proto to database entity failed")
	}
	for _, r := range records {
		m.store.Add(ctx, r)
	}
	return nil
}
