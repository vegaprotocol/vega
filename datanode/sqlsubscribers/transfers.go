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

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"github.com/pkg/errors"
)

type TransferEvent interface {
	events.Event
	TransferFunds() eventspb.TransferInstruction
}

type TransferStore interface {
	Upsert(ctx context.Context, transfer *entities.TransferInstruction) error
}

type AccountSource interface {
	Obtain(ctx context.Context, a *entities.Account) error
	GetByID(id int64) (entities.Account, error)
}

type Transfer struct {
	subscriber
	store         TransferStore
	accountSource AccountSource
	log           *logging.Logger
}

func NewTransfer(store TransferStore, accountSource AccountSource, log *logging.Logger) *Transfer {
	return &Transfer{
		store:         store,
		accountSource: accountSource,
		log:           log,
	}
}

func (rf *Transfer) Types() []events.Type {
	return []events.Type{events.TransferInstructionEvent}
}

func (rf *Transfer) Push(ctx context.Context, evt events.Event) error {
	return rf.consume(ctx, evt.(TransferEvent))
}

func (rf *Transfer) consume(ctx context.Context, event TransferEvent) error {
	transfer := event.TransferFunds()
	record, err := entities.TransferInstructionFromProto(ctx, &transfer, entities.TxHash(event.TxHash()), rf.vegaTime, rf.accountSource)
	if err != nil {
		return errors.Wrap(err, "converting transfer proto to database entity failed")
	}

	return errors.Wrap(rf.store.Upsert(ctx, record), "inserting transfer into to SQL store failed")
}
