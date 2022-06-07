package sqlsubscribers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type Ledger interface {
	AddLedgerEntry(entities.LedgerEntry) error
	AddTransferResponse(*vega.TransferResponse)
	Flush(ctx context.Context) error
}

type TransferResponseEvent interface {
	events.Event
	TransferResponses() []*vega.TransferResponse
}

type TransferResponse struct {
	subscriber
	ledger   Ledger
	accounts AccountService
	log      *logging.Logger
}

func NewTransferResponse(
	ledger Ledger,
	accounts AccountService,
	log *logging.Logger,
) *TransferResponse {
	return &TransferResponse{
		ledger:   ledger,
		accounts: accounts,
		log:      log,
	}
}

func (t *TransferResponse) Types() []events.Type {
	return []events.Type{events.TransferResponses}
}

func (t *TransferResponse) Flush(ctx context.Context) error {
	err := t.ledger.Flush(ctx)
	return errors.Wrap(err, "flushing ledger")
}

func (t *TransferResponse) Push(ctx context.Context, evt events.Event) error {
	return t.consume(ctx, evt.(TransferResponseEvent))
}

func (t *TransferResponse) consume(ctx context.Context, e TransferResponseEvent) error {
	var errs strings.Builder
	for _, tr := range e.TransferResponses() {
		t.ledger.AddTransferResponse(tr)
		for _, vle := range tr.Transfers {
			if err := t.addLedgerEntry(ctx, vle, t.vegaTime); err != nil {
				errs.WriteString(fmt.Sprintf("couldn't add ledger entry: %v, error:%s\n", vle, err))
			}
		}
	}

	if errs.Len() != 0 {
		return errors.Errorf("processing transfer response:%s", errs.String())
	}

	return nil
}

func (t *TransferResponse) addLedgerEntry(ctx context.Context, vle *vega.LedgerEntry, vegaTime time.Time) error {
	accFrom, err := t.obtainAccountWithID(ctx, vle.FromAccount, vegaTime)
	if err != nil {
		return errors.Wrap(err, "obtaining 'from' account")
	}

	accTo, err := t.obtainAccountWithID(ctx, vle.ToAccount, vegaTime)
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
func (t *TransferResponse) obtainAccountWithID(ctx context.Context, id string, vegaTime time.Time) (entities.Account, error) {
	a, err := entities.AccountFromAccountID(id)
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
