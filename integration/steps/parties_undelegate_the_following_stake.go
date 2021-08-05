package steps

import (
	"code.vegaprotocol.io/vega/types/num"
	"github.com/cucumber/godog"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/types"
)

func PartiesUndelegateTheFollowingStake(
	exec *execution.Engine,
	table *godog.Table,
) error {
	for _, r := range parseUndelegationTable(table) {
		row := newUndelegationRow(r)

		undelegateStake := types.Undelegate{
			NodeID: row.NodeID(),
			Amount: num.NewUint(row.Amount()),
			Method: commandspb.UndelegateSubmission_METHOD_AT_END_OF_EPOCH.String(),
		}

		_ = undelegateStake

		/*resp, err := exec.Undelegate(context.Background(), &undelegateStake)
		if err := checkExpectedError(row, err); err != nil {
			return err
		}*/
	}
	return nil
}

func parseUndelegationTable(table *godog.Table) []RowWrapper {
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
