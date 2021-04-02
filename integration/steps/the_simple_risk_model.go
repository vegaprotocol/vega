package steps

import (
	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/steps/market"
	types "code.vegaprotocol.io/vega/proto"
)

func TheSimpleRiskModel(config *market.Config, name string, table *gherkin.DataTable) error {
	r, err := GetFirstRow(*table)
	if err != nil {
		return err
	}

	row := simpleRiskModelRow{row: r}

	return config.RiskModels.AddSimple(name, &types.TradableInstrument_SimpleRiskModel{
		SimpleRiskModel: &types.SimpleRiskModel{
			Params: &types.SimpleModelParams{
				FactorLong:           row.long(),
				FactorShort:          row.short(),
				MaxMoveUp:            row.maxMoveUp(),
				MinMoveDown:          row.minMoveDown(),
				ProbabilityOfTrading: row.probabilityOfTrading(),
			},
		},
	})
}

type simpleRiskModelRow struct {
	row RowWrapper
}

func (r simpleRiskModelRow) probabilityOfTrading() float64 {
	return r.row.MustF64("probability of trading")
}

func (r simpleRiskModelRow) long() float64 {
	return r.row.MustF64("long")
}

func (r simpleRiskModelRow) short() float64 {
	return r.row.MustF64("short")
}

func (r simpleRiskModelRow) maxMoveUp() float64 {
	return r.row.MustF64("max move up")
}

func (r simpleRiskModelRow) minMoveDown() float64 {
	return r.row.MustF64("min move down")
}
