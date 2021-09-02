package steps

import (
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/delegation"

	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/types/num"
)

func TheValidators(
	topology *stubs.TopologyStub,
	stakingAcountStub *stubs.StakingAccountStub,
	delegtionEngine *delegation.Engine,
	table *godog.Table,
) error {
	for _, r := range parseTable(table) {
		row := newValidatorRow(r)
		topology.AddValidator(row.id())

		amt, _ := num.UintFromString(row.stakingAccountBalance(), 10)
		stakingAcountStub.IncrementBalance(row.id(), amt, "")
	}

	return nil
}

func newValidatorRow(r RowWrapper) validatorRow {
	row := validatorRow{
		row: r,
	}
	return row
}

func parseTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"staking account balance",
	}, nil)
}

type validatorRow struct {
	row RowWrapper
}

func (r validatorRow) id() string {
	return r.row.MustStr("id")
}

func (r validatorRow) stakingAccountBalance() string {
	return r.row.MustStr("staking account balance")
}
