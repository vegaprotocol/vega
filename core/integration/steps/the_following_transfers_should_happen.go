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

package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func TheFollowingTransfersShouldHappen(
	broker *stubs.BrokerStub,
	exec Execution,
	table *godog.Table,
) error {
	transfers := broker.GetTransfers(true)

	for _, r := range parseTransferTable(table) {
		row := transferRow{row: r}
		if row.IsAMM() {
			found := false
			if id, ok := exec.GetAMMSubAccountID(row.From()); ok {
				row.row.values["from"] = id
				found = true
			}
			if id, ok := exec.GetAMMSubAccountID(row.To()); ok {
				row.row.values["to"] = id
				found = true
			}
			if !found {
				return fmt.Errorf("no AMM aliases found for from (%s) or to (%s)", row.From(), row.To())
			}
		}

		matched, divergingAmounts := matchTransfers(transfers, row)

		if matched {
			continue
		}

		if len(divergingAmounts) == 0 {
			return errMissingTransfer(row)
		}
		return errTransferFoundButNotRightAmount(row, divergingAmounts)
	}

	broker.ResetType(events.LedgerMovementsEvent)

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

func matchTransfers(ledgerEntries []*vegapb.LedgerEntry, row transferRow) (bool, []uint64) {
	divergingAmounts := []uint64{}
	for _, transfer := range ledgerEntries {
		if transfer.FromAccount.ID() == row.FromAccountID() && transfer.ToAccount.ID() == row.ToAccountID() {
			if row.Type() != "" && transfer.Type != vegapb.TransferType(vegapb.TransferType_value[row.Type()]) {
				continue
			}
			if stringToU64(transfer.Amount) == row.Amount() {
				return true, nil
			}
			divergingAmounts = append(divergingAmounts, stringToU64(transfer.Amount))
		}
	}
	return false, divergingAmounts
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
	}, []string{
		"type",
		"is amm",
	})
}

type transferRow struct {
	row RowWrapper
}

func (r transferRow) From() string {
	return r.row.MustStr("from")
}

func (r transferRow) FromAccount() vegapb.AccountType {
	return r.row.MustAccount("from account")
}

func (r transferRow) FromAccountID() string {
	return AccountID(r.MarketID(), r.From(), r.Asset(), r.FromAccount())
}

func (r transferRow) To() string {
	return r.row.MustStr("to")
}

func (r transferRow) Type() string {
	return r.row.Str("type")
}

func (r transferRow) ToAccount() vegapb.AccountType {
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

func (r transferRow) IsAMM() bool {
	if !r.row.HasColumn("is amm") {
		return false
	}
	return r.row.MustBool("is amm")
}
