package steps

import (
	"context"

	"code.vegaprotocol.io/vega/types/num"
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/delegation"
)

func PartiesDelegateTheFollowingStake(
	engine *delegation.Engine,
	table *godog.Table,
) error {
	for _, r := range parseDelegationTable(table) {
		row := newDelegationRow(r)
		err := engine.Delegate(context.Background(), row.Party(), row.NodeID(), num.NewUint(row.Amount()))
		if err := checkExpectedError(row, err); err != nil {
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
