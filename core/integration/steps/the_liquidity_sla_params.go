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

func TheLiquiditySLAPArams(config *market.Config, name string, table *godog.Table) error {
	row := slaParamRow{row: parseSLAParamsTable(table)[0]}

	return config.LiquiditySLAParams.Add(
		name,
		&types.LiquiditySLAParameters{
			PriceRange:                  row.priceRange(),
			CommitmentMinTimeFraction:   row.commitmentMinTimeFraction(),
			SlaCompetitionFactor:        row.slaCompetitionFactor(),
			PerformanceHysteresisEpochs: uint64(row.performanceHysteresisEpochs()),
		})
}

func parseSLAParamsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"price range",
		"commitment min time fraction",
		"performance hysteresis epochs",
		"sla competition factor",
	}, []string{})
}

type slaParamRow struct {
	row RowWrapper
}

func (r slaParamRow) priceRange() string {
	return r.row.MustStr("price range")
}

func (r slaParamRow) commitmentMinTimeFraction() string {
	return r.row.MustStr("commitment min time fraction")
}

func (r slaParamRow) performanceHysteresisEpochs() int64 {
	return r.row.MustI64("performance hysteresis epochs")
}

func (r slaParamRow) slaCompetitionFactor() string {
	return r.row.MustStr("sla competition factor")
}
