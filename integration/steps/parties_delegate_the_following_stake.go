package steps

import (
	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/types"
)

func PartiesDelegateTheFollowingStake(
	exec *execution.Engine,
	table *gherkin.DataTable,
) error {
	for _, r := range parseDelegationTable(table) {
		row := newDelegationRow(r)

		delegateStake := types.Delegate{
			NodeID: row.NodeID(),
			Amount: row.Amount(),
		}

		_ = delegateStake
		/*		resp, err := exec.Delegate(context.Background(), &delegateStake)
				if err := checkExpectedError(row, err); err != nil {
					return err
				}*/
	}
	return nil
}

func parseDelegationTable(table *gherkin.DataTable) []RowWrapper {
	return StrictParseTable(table, []string{
		"node id",
		"amount",
	}, nil)
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

func (r delegationRow) NodeID() string {
	return r.row.MustStr("node id")
}

func (r delegationRow) Amount() uint64 {
	return r.row.MustU64("amount")
}
