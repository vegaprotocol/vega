package steps

import (
	"context"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/delegation"
	"code.vegaprotocol.io/vega/types/num"
)

func PartiesUndelegateTheFollowingStake(
	engine *delegation.Engine,
	table *godog.Table,
) error {
	for _, r := range parseUndelegationTable(table) {
		row := newUndelegationRow(r)

		err := engine.UndelegateAtEndOfEpoch(context.Background(), row.Party(), row.NodeID(), num.NewUint(row.Amount()))

		if err := checkExpectedError(row, err); err != nil {
			return err
		}
	}
	return nil
}

func parseUndelegationTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"node id",
		"amount",
	}, []string{
		"reference",
		"error"})
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
