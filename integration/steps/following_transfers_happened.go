package steps

import (
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func TheFollowingTransfersHappened(
	broker *stubs.BrokerStub,
	table *gherkin.DataTable,
) error {
	transfers := getTransfers(broker)

	for _, r := range TableWrapper(*table).Parse() {
		row := transferRow{row: r}

		matched, divergingAmounts := matchTransfers(transfers, row)

		if matched {
			continue
		}

		if len(divergingAmounts) == 0 {
			return errMissingTransfer(row)
		} else {
			return errTransferFoundButNotRightAmount(row, divergingAmounts)
		}
	}

	broker.ResetType(events.TransferResponses)

	return nil
}

func errTransferFoundButNotRightAmount(row transferRow, divergingAmounts []uint64) error {
	return fmt.Errorf("invalid amount for transfer from %v to %v, expected(%d) got(%v)",
		row.fromAccountID(), row.toAccountID(), row.amount(), divergingAmounts,
	)
}

func errMissingTransfer(row transferRow) error {
	return fmt.Errorf("missing transfers between %v and %v for amount %v",
		row.fromAccountID(), row.toAccountID(), row.amount(),
	)
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

func (r transferRow) fromAccount() types.AccountType {
	return r.row.Account("from account")
}

func (r transferRow) fromAccountID() string {
	return accountID(r.marketID(), r.from(), r.asset(), r.fromAccount())
}

func (r transferRow) to() string {
	return r.row.Str("to")
}

func (r transferRow) toAccount() types.AccountType {
	return r.row.Account("to account")
}

func (r transferRow) toAccountID() string {
	return accountID(r.marketID(), r.to(), r.asset(), r.toAccount())
}

func (r transferRow) marketID() string {
	return r.row.Str("market id")
}

func (r transferRow) amount() uint64 {
	return r.row.U64("amount")
}

func (r transferRow) asset() string {
	return r.row.Str("asset")
}
