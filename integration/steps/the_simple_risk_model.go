package steps

import (
	"github.com/cucumber/godog"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/integration/steps/market"
)

func TheSimpleRiskModel(config *market.Config, name string, table *godog.Table) error {
	row := simpleRiskModelRow{row: parseSimpleRiskModelTable(table)}

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

func parseSimpleRiskModelTable(table *godog.Table) RowWrapper {
	return StrictParseFirstRow(table, []string{
		"probability of trading",
		"long",
		"short",
		"max move up",
		"min move down",
	}, []string{})
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
