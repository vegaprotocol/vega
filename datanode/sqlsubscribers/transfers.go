// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type TransferEvent interface {
	events.Event
	TransferFunds() eventspb.Transfer
}

type TransferStore interface {
	Upsert(ctx context.Context, transfer *entities.Transfer) error
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
	return []events.Type{events.TransferEvent}
}

func (rf *Transfer) Push(ctx context.Context, evt events.Event) error {
	return rf.consume(ctx, evt.(TransferEvent))
}

func (rf *Transfer) consume(ctx context.Context, event TransferEvent) error {
	transfer := event.TransferFunds()
	record, err := entities.TransferFromProto(ctx, &transfer, entities.TxHash(event.TxHash()), rf.vegaTime, rf.accountSource)
	if err != nil {
		return errors.Wrap(err, "converting transfer proto to database entity failed")
	}

	return errors.Wrap(rf.store.Upsert(ctx, record), "inserting transfer into to SQL store failed")
}
