package steps

import (
	"github.com/cucumber/godog"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/integration/steps/market"
)

func TheFeesConfiguration(config *market.Config, name string, table *godog.Table) error {
	row := feesConfigRow{row: parseFeesConfigTable(table)}

	return config.FeesConfig.Add(name, &types.Fees{
		Factors: &types.FeeFactors{
			InfrastructureFee: row.infrastructureFee(),
			MakerFee:          row.makerFee(),
		},
	})
}

func parseFeesConfigTable(table *godog.Table) RowWrapper {
	return StrictParseFirstRow(table, []string{
		"maker fee",
		"infrastructure fee",
	}, []string{})
}

type feesConfigRow struct {
	row RowWrapper
}

func (r feesConfigRow) makerFee() string {
	return r.row.MustStr("maker fee")
}

func (r feesConfigRow) infrastructureFee() string {
	return r.row.MustStr("infrastructure fee")
}
