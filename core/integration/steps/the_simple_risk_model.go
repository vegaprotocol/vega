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
