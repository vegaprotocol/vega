// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlsubscribers

import (
	"context"

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
	Add(ctx context.Context, e *entities.ERC20MultiSigSignerEvent) error
}

type ERC20MultiSigSignerEvent struct {
	subscriber
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
	record, err := entities.ERC20MultiSigSignerEventFromAddedProto(&e)
	if err != nil {
		return errors.Wrap(err, "converting signer-added proto to database entity failed")
	}
	return m.store.Add(ctx, record)
}

func (m *ERC20MultiSigSignerEvent) consumeRemovedEvent(ctx context.Context, event ERC20MultiSigSignerRemovedEvent) error {
	e := event.Proto()
	records, err := entities.ERC20MultiSigSignerEventFromRemovedProto(&e)
	if err != nil {
		return errors.Wrap(err, "converting signer-added proto to database entity failed")
	}
	for _, r := range records {
		m.store.Add(ctx, r)
	}
	return nil
}
