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

package steps

import (
	"fmt"

	"github.com/cucumber/godog"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheFollowingTransfersShouldHappen(
	broker *stubs.BrokerStub,
	table *godog.Table,
) error {
	transfers := getTransfers(broker)

	for _, r := range parseTransferTable(table) {
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
	return formatDiff(
		fmt.Sprintf("invalid amount for transfer from %s to %s", row.FromAccountID(), row.ToAccountID()),
		map[string]string{
			"amount": u64ToS(row.Amount()),
		},
		map[string]string{
			"amount": u64SToS(divergingAmounts),
		},
	)
}

func errMissingTransfer(row transferRow) error {
	return fmt.Errorf("missing transfers between %v and %v for amount %v",
		row.FromAccountID(), row.ToAccountID(), row.Amount(),
	)
}

func matchTransfers(transfers []*types.LedgerEntry, row transferRow) (bool, []uint64) {
	divergingAmounts := []uint64{}
	for _, transfer := range transfers {
		if transfer.FromAccount == row.FromAccountID() && transfer.ToAccount == row.ToAccountID() {
			if stringToU64(transfer.Amount) == row.Amount() {
				return true, nil
			}
			divergingAmounts = append(divergingAmounts, stringToU64(transfer.Amount))
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

func parseTransferTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"from",
		"from account",
		"to",
		"to account",
		"market id",
		"amount",
		"asset",
	}, []string{})
}

type transferRow struct {
	row RowWrapper
}

func (r transferRow) From() string {
	return r.row.MustStr("from")
}

func (r transferRow) FromAccount() types.AccountType {
	return r.row.MustAccount("from account")
}

func (r transferRow) FromAccountID() string {
	return AccountID(r.MarketID(), r.From(), r.Asset(), r.FromAccount())
}

func (r transferRow) To() string {
	return r.row.MustStr("to")
}

func (r transferRow) ToAccount() types.AccountType {
	return r.row.MustAccount("to account")
}

func (r transferRow) ToAccountID() string {
	return AccountID(r.MarketID(), r.To(), r.Asset(), r.ToAccount())
}

func (r transferRow) MarketID() string {
	return r.row.MustStr("market id")
}

func (r transferRow) Amount() uint64 {
	return r.row.MustU64("amount")
}

func (r transferRow) Asset() string {
	return r.row.MustStr("asset")
}
