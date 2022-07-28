// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package steps

import (
	"context"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/delegation"
	"code.vegaprotocol.io/vega/core/types/num"
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
