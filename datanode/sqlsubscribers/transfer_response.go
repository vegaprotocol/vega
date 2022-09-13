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
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type Ledger interface {
	AddLedgerEntry(entities.LedgerEntry) error
	AddTransferInstructionResponse(*vega.TransferInstructionResponse)
	Flush(ctx context.Context) error
}

type TransferInstructionResponseEvent interface {
	events.Event
	TransferInstructionResponses() []*vega.TransferInstructionResponse
}

type TransferInstructionResponse struct {
	subscriber
	ledger   Ledger
	accounts AccountService
	log      *logging.Logger
}

func NewTransferInstructionResponse(
	ledger Ledger,
	accounts AccountService,
	log *logging.Logger,
) *TransferInstructionResponse {
	return &TransferInstructionResponse{
		ledger:   ledger,
		accounts: accounts,
		log:      log,
	}
}

func (t *TransferInstructionResponse) Types() []events.Type {
	return []events.Type{events.TransferInstructionResponses}
}

func (t *TransferInstructionResponse) Flush(ctx context.Context) error {
	err := t.ledger.Flush(ctx)
	return errors.Wrap(err, "flushing ledger")
}

func (t *TransferInstructionResponse) Push(ctx context.Context, evt events.Event) error {
	return t.consume(ctx, evt.(TransferInstructionResponseEvent))
}

func (t *TransferInstructionResponse) consume(ctx context.Context, e TransferInstructionResponseEvent) error {
	var errs strings.Builder
	for _, tr := range e.TransferInstructionResponses() {
		t.ledger.AddTransferInstructionResponse(tr)
		for _, vle := range tr.Transfers {
			if err := t.addLedgerEntry(ctx, vle, e.TxHash(), t.vegaTime); err != nil {
				errs.WriteString(fmt.Sprintf("couldn't add ledger entry: %v, error:%s\n", vle, err))
			}
		}
	}

	if errs.Len() != 0 {
		return errors.Errorf("processing transfer response:%s", errs.String())
	}

	return nil
}

func (t *TransferInstructionResponse) addLedgerEntry(ctx context.Context, vle *vega.LedgerEntry, txHash string, vegaTime time.Time) error {
	accFrom, err := t.obtainAccountWithID(ctx, vle.FromAccount, txHash, vegaTime)
	if err != nil {
		return errors.Wrap(err, "obtaining 'from' account")
	}

	accTo, err := t.obtainAccountWithID(ctx, vle.ToAccount, txHash, vegaTime)
	if err != nil {
		return errors.Wrap(err, "obtaining 'to' account")
	}

	quantity, err := decimal.NewFromString(vle.Amount)
	if err != nil {
		return errors.Wrap(err, "parsing amount string")
	}

	le := entities.LedgerEntry{
		AccountFromID: accTo.ID,
		AccountToID:   accFrom.ID,
		Quantity:      quantity,
		TxHash:        entities.TxHash(txHash),
		VegaTime:      vegaTime,
		TransferTime:  time.Unix(0, vle.Timestamp),
		Reference:     vle.Reference,
		Type:          vle.Type,
	}

	err = t.ledger.AddLedgerEntry(le)
	if err != nil {
		return errors.Wrap(err, "adding to store")
	}
	return nil
}

// Parse the vega account ID; if that account already exists in the db, fetch it; else create it.
func (t *TransferInstructionResponse) obtainAccountWithID(ctx context.Context, id string, txHash string, vegaTime time.Time) (entities.Account, error) {
	a, err := entities.AccountFromAccountID(id, entities.TxHash(txHash))
	if err != nil {
		return entities.Account{}, errors.Wrapf(err, "parsing account id: %s", id)
	}
	a.VegaTime = vegaTime
	err = t.accounts.Obtain(ctx, &a)
	if err != nil {
		return entities.Account{}, errors.Wrapf(err, "obtaining account for id: %s", id)
	}
	return a, nil
}
