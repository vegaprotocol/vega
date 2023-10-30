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
	"context"

	"code.vegaprotocol.io/vega/libs/num"
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/delegation"
)

func PartiesDelegateTheFollowingStake(
	engine *delegation.Engine,
	table *godog.Table,
) error {
	for _, r := range parseDelegationTable(table) {
		row := newDelegationRow(r)
		err := engine.Delegate(context.Background(), row.Party(), row.NodeID(), num.NewUint(row.Amount()))
		if err := checkExpectedError(row, err, nil); err != nil {
			return err
		}
	}
	return nil
}

func parseDelegationTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"node id",
		"amount",
	}, []string{
		"reference",
		"error",
	})
}

type delegationRow struct {
	row RowWrapper
}

func newDelegationRow(r RowWrapper) delegationRow {
	row := delegationRow{
		row: r,
	}
	return row
}

func (r delegationRow) Party() string {
	return r.row.MustStr("party")
}

func (r delegationRow) NodeID() string {
	return r.row.MustStr("node id")
}

func (r delegationRow) Amount() uint64 {
	return r.row.MustU64("amount")
}

func (r delegationRow) Error() string {
	return r.row.Str("error")
}

func (r delegationRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r delegationRow) Reference() string {
	return r.row.MustStr("reference")
}
