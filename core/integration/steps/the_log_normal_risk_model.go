// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
