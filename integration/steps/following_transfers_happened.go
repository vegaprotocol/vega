package steps

import (
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func TheFollowingTransfersHappened(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	transfers := getTransfers(broker)

	for _, r := range TableWrapper(*table).Parse() {
		row := transferRow{row: r}

		matched, divergingAmounts := matchTransfers(transfers, row)

		if matched {
			continue
		}

		if len(divergingAmounts) == 0 {
			return fmt.Errorf("missing transfers between %v and %v for amount %v",
				row.fromAccountID(), row.toAccountID(), row.amount(),
			)
		} else {
			return fmt.Errorf("invalid amount for transfer from %v to %v, expected(%d) got(%v)",
				row.fromAccountID(), row.toAccountID(), row.amount(), divergingAmounts,
			)
		}
	}

	broker.ResetType(events.TransferResponses)

	return nil
}

func matchTransfers(transfers []*types.LedgerEntry, row transferRow) (bool, []uint64) {
	divergingAmounts := []uint64{}
	for _, transfer := range transfers {
		if transfer.FromAccount == row.fromAccountID() && transfer.ToAccount == row.toAccountID() {
			if transfer.Amount == row.amount() {
				return true, nil
			}
			divergingAmounts = append(divergingAmounts, transfer.Amount)
		}
	}
	return false, divergingAmounts
}

func getTransfers(broker *stubs.BrokerStub) []*types.LedgerEntry {
	transferEvents := broker.GetTransferResponses()
	transfers := []*types.LedgerEntry{}
	for _, e := range transferEvents {
		for _, response := range e.TransferResponses() {
			transfers = append(transfers, response.GetTransfers()...)
		}
	}
	return transfers
}

type transferRow struct {
	row RowWrapper
}

func (r transferRow) from() string {
	return r.row.Str("from")
}

func (r transferRow) to() string {
	return r.row.Str("to")
}

func (r transferRow) fromAccount() types.AccountType {
	return account(r.row.Str("from account"))
}

func (r transferRow) fromAccountID() string {
	return accountID(r.marketID(), r.from(), r.asset(), r.fromAccount())
}

func (r transferRow) toAccount() types.AccountType {
	return account(r.row.Str("to account"))
}

func (r transferRow) toAccountID() string {
	return accountID(r.marketID(), r.to(), r.asset(), r.toAccount())
}

func (r transferRow) marketID() string {
	return r.row.Str("market id")
}

func (r transferRow) amount() uint64 {
	value, err := r.row.U64("amount")
	if err != nil {
		panic(err)
	}
	return value
}

func (r transferRow) asset() string {
	return r.row.Str("asset")
}
