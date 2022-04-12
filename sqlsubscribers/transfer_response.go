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
	Add(*entities.LedgerEntry) error
	Flush(ctx context.Context) error
}

type AccountStore interface {
	Obtain(a *entities.Account) error
}

type BalanceStore interface {
	Add(b entities.Balance) error
	Flush(ctx context.Context) error
}

type TransferResponseEvent interface {
	events.Event
	TransferResponses() []*vega.TransferResponse
}

type TransferResponse struct {
	ledger   Ledger
	accounts AccountStore
	parties  PartyStore
	vegaTime time.Time
	balances BalanceStore
	log      *logging.Logger
}

func NewTransferResponse(
	ledger Ledger,
	accounts AccountStore,
	balances BalanceStore,
	parties PartyStore,
	log *logging.Logger,
) *TransferResponse {
	return &TransferResponse{
		ledger:   ledger,
		accounts: accounts,
		balances: balances,
		parties:  parties,
		log:      log,
	}
}

func (t *TransferResponse) Types() []events.Type {
	return []events.Type{events.TransferResponses}
}

func (t *TransferResponse) Push(evt events.Event) error {
	ctx := context.Background()
	switch e := evt.(type) {
	case TimeUpdateEvent:
		t.vegaTime = e.Time()
		err := t.ledger.Flush(ctx)
		if err != nil {
			return errors.Wrap(err, "flushing ledgers")
		}
		err = t.balances.Flush(ctx)
		return errors.Wrap(err, "flushing balances")
	case TransferResponseEvent:
		return t.consume(e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}
}

func (t *TransferResponse) consume(e TransferResponseEvent) error {

	var errs strings.Builder
	for _, tr := range e.TransferResponses() {
		for _, vle := range tr.Transfers {
			if err := t.addLedgerEntry(vle, t.vegaTime); err != nil {
				errs.WriteString(fmt.Sprintf("couldn't add ledger entry: %v, error:%s\n", vle, err))
			}
		}
		for _, vb := range tr.Balances {
			if err := t.addBalance(vb, t.vegaTime); err != nil {
				errs.WriteString(fmt.Sprintf("couldn't add balance: %v, error:%s\n", vb, err))
			}
		}
	}

	if errs.Len() != 0 {
		return errors.Errorf("processing transfer response:%s", errs.String())
	}

	return nil
}

func (t *TransferResponse) addBalance(vb *vega.TransferBalance, vegaTime time.Time) error {
	acc, err := t.obtainAccountWithProto(vb.Account, vegaTime)
	if err != nil {
		return errors.Wrap(err, "obtaining account")
	}

	balance, err := decimal.NewFromString(vb.Balance)
	if err != nil {
		return errors.Wrap(err, "parsing account balance")
	}

	b := entities.Balance{
		AccountID: acc.ID,
		Balance:   balance,
		VegaTime:  vegaTime,
	}

	err = t.balances.Add(b)
	if err != nil {
		return errors.Wrap(err, "adding balance to store")
	}
	return nil
}

func (t *TransferResponse) addLedgerEntry(vle *vega.LedgerEntry, vegaTime time.Time) error {
	accFrom, err := t.obtainAccountWithID(vle.FromAccount, vegaTime)
	if err != nil {
		return errors.Wrap(err, "obtaining 'from' account")
	}

	accTo, err := t.obtainAccountWithID(vle.ToAccount, vegaTime)
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

	err = t.ledger.Add(&le)
	if err != nil {
		return errors.Wrap(err, "adding to store")
	}
	return nil
}

// Parse the vega account ID; if that account already exists in the db, fetch it; else create it.
func (t *TransferResponse) obtainAccountWithID(id string, vegaTime time.Time) (entities.Account, error) {
	a, err := entities.AccountFromAccountID(id)
	if err != nil {
		return entities.Account{}, errors.Wrapf(err, "parsing account id: %s", id)
	}
	a.VegaTime = vegaTime
	err = t.accounts.Obtain(&a)
	if err != nil {
		return entities.Account{}, errors.Wrapf(err, "obtaining account for id: %s", id)
	}
	return a, nil
}

func (t *TransferResponse) obtainAccountWithProto(va *vega.Account, vegaTime time.Time) (entities.Account, error) {
	a, err := entities.AccountFromProto(*va)
	if err != nil {
		return entities.Account{}, errors.Wrap(err, "obtaining account for balance")
	}

	a.VegaTime = vegaTime
	err = t.accounts.Obtain(&a)
	if err != nil {
		return entities.Account{}, errors.Wrap(err, "obtaining account")
	}
	return a, nil
}
