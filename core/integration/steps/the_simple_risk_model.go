// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/core/integration/steps/market"
	types "code.vegaprotocol.io/vega/protos/vega"
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
