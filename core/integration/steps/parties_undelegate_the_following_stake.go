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

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/delegation"
	"code.vegaprotocol.io/vega/libs/num"
)

func PartiesUndelegateTheFollowingStake(
	engine *delegation.Engine,
	table *godog.Table,
) error {
	for _, r := range parseUndelegationTable(table) {
		row := newUndelegationRow(r)

		if row.When() == "now" {
			err := engine.UndelegateNow(context.Background(), row.Party(), row.NodeID(), num.NewUint(row.Amount()))

			if err := checkExpectedError(row, err, nil); err != nil {
				return err
			}
		} else {
			err := engine.UndelegateAtEndOfEpoch(context.Background(), row.Party(), row.NodeID(), num.NewUint(row.Amount()))

			if err := checkExpectedError(row, err, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseUndelegationTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"node id",
		"amount",
		"when",
	}, []string{
		"reference",
		"error",
	})
}

type undelegationRow struct {
	row RowWrapper
}

func newUndelegationRow(r RowWrapper) undelegationRow {
	row := undelegationRow{
		row: r,
	}
	return row
}

func (r undelegationRow) When() string {
	return r.row.MustStr("when")
}

func (r undelegationRow) Party() string {
	return r.row.MustStr("party")
}

func (r undelegationRow) NodeID() string {
	return r.row.MustStr("node id")
}

func (r undelegationRow) Amount() uint64 {
	return r.row.MustU64("amount")
}

func (r undelegationRow) Error() string {
	return r.row.Str("error")
}

func (r undelegationRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r undelegationRow) Reference() string {
	return r.row.MustStr("reference")
}
