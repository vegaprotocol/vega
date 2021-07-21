package steps

import (
	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/types"
)

func PartiesUndelegateTheFollowingStake(
	exec *execution.Engine,
	table *gherkin.DataTable,
) error {
	for _, r := range parseUndelegationTable(table) {
		row := newDelegationRow(r)

		undelegateStake := types.UndelegateAtEpochEnd{
			NodeID: row.NodeID(),
			Amount: row.Amount(),
		}

		_ = undelegateStake

		/*		resp, err := exec.Undelegate(context.Background(), &undelegateStake)
				if err := checkExpectedError(row, err); err != nil {
					return err
				}*/
	}
	return nil
}

func parseUndelegationTable(table *gherkin.DataTable) []RowWrapper {
	return StrictParseTable(table, []string{
		"node id",
		"amount",
	}, nil)
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

func (r undelegationRow) NodeID() string {
	return r.row.MustStr("node id")
}

func (r undelegationRow) Amount() uint64 {
	return r.row.MustU64("amount")
}
