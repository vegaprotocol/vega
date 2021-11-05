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
		topology.AddValidator(row.id(), row.pubKey())

		amt, _ := num.UintFromString(row.stakingAccountBalance(), 10)
		stakingAcountStub.IncrementBalance(row.pubKey(), amt)
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
	}, []string{
		"pub_key",
	})
}

type validatorRow struct {
	row RowWrapper
}

func (r validatorRow) pubKey() string {
	pk, ok := r.row.StrB("pub_key")
	if !ok {
		return r.id()
	}
	return pk
}

func (r validatorRow) id() string {
	return r.row.MustStr("id")
}

func (r validatorRow) stakingAccountBalance() string {
	return r.row.MustStr("staking account balance")
}
