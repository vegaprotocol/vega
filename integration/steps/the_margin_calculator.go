package steps

import (
	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/steps/market"
	types "code.vegaprotocol.io/vega/proto"
)

func TheMarginCalculator(config *market.Config, name string, table *gherkin.DataTable) error {
	row := marginCalculatorRow{row: parseMarginCalculatorTable(table)}

	return config.MarginCalculators.Add(name, &types.MarginCalculator{
		ScalingFactors: &types.ScalingFactors{
			SearchLevel:       row.searchLevelFactor(),
			InitialMargin:     row.initialMarginFactor(),
			CollateralRelease: row.collateralReleaseFactor(),
		},
	})
}

func parseMarginCalculatorTable(table *gherkin.DataTable) RowWrapper {
	return StrictParseFirstRow(table, []string{
		"release factor",
		"initial factor",
		"search factor",
	}, []string{})
}

type marginCalculatorRow struct {
	row RowWrapper
}

func (r marginCalculatorRow) collateralReleaseFactor() float64 {
	return r.row.MustF64("release factor")
}

func (r marginCalculatorRow) initialMarginFactor() float64 {
	return r.row.MustF64("initial factor")
}

func (r marginCalculatorRow) searchLevelFactor() float64 {
	return r.row.MustF64("search factor")
}
