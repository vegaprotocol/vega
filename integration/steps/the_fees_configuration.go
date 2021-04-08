package steps

import (
	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/steps/market"
	types "code.vegaprotocol.io/vega/proto"
)

func TheFeesConfiguration(config *market.Config, name string, table *gherkin.DataTable) error {
	r, err := GetFirstRow(*table)
	if err != nil {
		return err
	}

	row := feesConfigRow{row: r}

	return config.FeesConfig.Add(name, &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      row.liquidityFee(),
			InfrastructureFee: row.infrastructureFee(),
			MakerFee:          row.makerFee(),
		},
	})
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

func (r feesConfigRow) liquidityFee() string {
	return r.row.MustStr("liquidity fee")
}
