// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package steps

import (
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/integration/steps/market"
	types "code.vegaprotocol.io/vega/protos/vega"
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
