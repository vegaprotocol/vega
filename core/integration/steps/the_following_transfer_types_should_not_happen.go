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

	"code.vegaprotocol.io/vega/core/integration/stubs"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func TheTransfersOfFollowingTypesShouldNotHappen(
	broker *stubs.BrokerStub,
	table *godog.Table,
) error {
	transfers := broker.GetTransfers(true)

	for _, r := range parseNotTransferTable(table) {
		row := transferNotRow{row: r}
		if matchTransferType(transfers, row) {
			return errTransferFound(row)
		}
	}

	return nil
}

type transferNotRow struct {
	row RowWrapper
}

func errTransferFound(row transferNotRow) error {
	return fmt.Errorf("transfer of type '%s' found",
		row.Type(),
	)
}

func matchTransferType(ledgerEntries []*vegapb.LedgerEntry, row transferNotRow) bool {
	for _, transfer := range ledgerEntries {
		if row.Type() != "" && transfer.Type == vegapb.TransferType(vegapb.TransferType_value[row.Type()]) {
			return true
		}
	}
	return false
}

func parseNotTransferTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"type",
	}, []string{})
}

func (r transferNotRow) Type() string {
	return r.row.MustStr("type")
}
