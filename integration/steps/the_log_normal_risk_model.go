package steps

import (
	"github.com/cucumber/godog"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/integration/steps/market"
)

func TheLogNormalRiskModel(config *market.Config, name string, table *godog.Table) error {
	row := logNormalRiskModelRow{row: parseLogNormalRiskModelTable(table)}

	return config.RiskModels.AddLogNormal(name, &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: row.riskAversion(),
			Tau:                   row.tau(),
			Params: &types.LogNormalModelParams{
				Mu:    row.mu(),
				R:     row.r(),
				Sigma: row.sigma(),
			},
		},
	})
}

func parseLogNormalRiskModelTable(table *godog.Table) RowWrapper {
	return StrictParseFirstRow(table, []string{
		"risk aversion",
		"tau",
		"mu",
		"r",
		"sigma",
	}, []string{})
}

type logNormalRiskModelRow struct {
	row RowWrapper
}

func (r logNormalRiskModelRow) riskAversion() float64 {
	return r.row.MustF64("risk aversion")
}

func (r logNormalRiskModelRow) tau() float64 {
	return r.row.MustF64("tau")
}

func (r logNormalRiskModelRow) mu() float64 {
	return r.row.MustF64("mu")
}

func (r logNormalRiskModelRow) r() float64 {
	return r.row.MustF64("r")
}

func (r logNormalRiskModelRow) sigma() float64 {
	return r.row.MustF64("sigma")
}
