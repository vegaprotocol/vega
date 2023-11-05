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
	"code.vegaprotocol.io/vega/core/integration/steps/market"
	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func TheMarginCalculator(config *market.Config, name string, table *godog.Table) error {
	row := marginCalculatorRow{row: parseMarginCalculatorTable(table)}

	return config.MarginCalculators.Add(name, &types.MarginCalculator{
		ScalingFactors: &types.ScalingFactors{
			SearchLevel:       row.searchLevelFactor(),
			InitialMargin:     row.initialMarginFactor(),
			CollateralRelease: row.collateralReleaseFactor(),
		},
	})
}

func parseMarginCalculatorTable(table *godog.Table) RowWrapper {
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
