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

type TransferEvent interface {
	events.Event
	TransferFunds() eventspb.Transfer
}

type TransferFeesEvent interface {
	events.Event
	TransferFees() eventspb.TransferFees
}

type TransferFeesDiscountUpdateEvent interface {
	events.Event
	TransferFeesDiscount() eventspb.TransferFeesDiscount
}

type TransferStore interface {
	Upsert(ctx context.Context, transfer *entities.Transfer) error
	UpsertFees(ctx context.Context, tf *entities.TransferFees) error
	UpsertFeesDiscount(ctx context.Context, tfd *entities.TransferFeesDiscount) error
}

type AccountSource interface {
	Obtain(ctx context.Context, a *entities.Account) error
	GetByID(ctx context.Context, id entities.AccountID) (entities.Account, error)
}

type Transfer struct {
	subscriber
	store         TransferStore
	accountSource AccountSource
}

func NewTransfer(store TransferStore, accountSource AccountSource) *Transfer {
	return &Transfer{
		store:         store,
		accountSource: accountSource,
	}
}

func (rf *Transfer) Types() []events.Type {
	return []events.Type{
		events.TransferEvent,
		events.TransferFeesEvent,
	}
}

func (rf *Transfer) Push(ctx context.Context, evt events.Event) error {
	switch te := evt.(type) {
	case TransferEvent:
		return rf.consume(ctx, te)
	case TransferFeesEvent:
		return rf.handleFees(ctx, te)
	case TransferFeesDiscountUpdateEvent:
		return rf.handleDiscount(ctx, te)
	}
	return errors.New("unsupported event")
}

func (rf *Transfer) consume(ctx context.Context, event TransferEvent) error {
	transfer := event.TransferFunds()
	record, err := entities.TransferFromProto(ctx, &transfer, entities.TxHash(event.TxHash()), rf.vegaTime, rf.accountSource)
	if err != nil {
		return errors.Wrap(err, "converting transfer proto to database entity failed")
	}

	return errors.Wrap(rf.store.Upsert(ctx, record), "inserting transfer into to SQL store failed")
}

func (rf *Transfer) handleFees(ctx context.Context, e TransferFeesEvent) error {
	tf := e.TransferFees()
	rec := entities.TransferFeesFromProto(&tf, rf.vegaTime)
	if err := rf.store.UpsertFees(ctx, rec); err != nil {
		return errors.Wrap(err, "inserting transfer fee into SQL store failed")
	}

	// TODO karel - update the discount table by adding a new version with de-ducted fees
	return nil
}

func (rf *Transfer) handleDiscount(ctx context.Context, e TransferFeesDiscountUpdateEvent) error {
	tf := e.TransferFeesDiscount()
	discount := entities.TransferFeesDiscountFromProto(&tf, rf.vegaTime)
	return errors.Wrap(rf.store.UpsertFeesDiscount(ctx, discount), "inserting transfer fee into SQL store failed")
}
