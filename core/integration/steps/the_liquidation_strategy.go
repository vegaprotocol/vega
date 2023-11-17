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
	"time"

	"code.vegaprotocol.io/vega/core/integration/steps/market"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

func TheLiquidationStrategies(config *market.Config, table *godog.Table) error {
	rows := parseLiquidationStrategyTable(table)
	for _, row := range rows {
		lsr := lsRow{
			r: row,
		}
		config.LiquidationStrat.Add(lsr.name(), lsr.liquidationStrategy().IntoProto())
	}
	return nil
}

func parseLiquidationStrategyTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"name",
		"disposal step",
		"disposal fraction",
		"full disposal size",
		"max fraction consumed",
	}, nil)
}

type lsRow struct {
	r RowWrapper
}

func (l lsRow) liquidationStrategy() *types.LiquidationStrategy {
	return &types.LiquidationStrategy{
		DisposalTimeStep:    l.disposalStep(),
		DisposalFraction:    l.disposalFraction(),
		FullDisposalSize:    l.fullDisposalSize(),
		MaxFractionConsumed: l.maxFraction(),
	}
}

func (l lsRow) name() string {
	return l.r.MustStr("name")
}

func (l lsRow) disposalStep() time.Duration {
	i := l.r.MustI64("disposal step")
	return time.Duration(i) * time.Second
}

func (l lsRow) disposalFraction() num.Decimal {
	return l.r.MustDecimal("disposal fraction")
}

func (l lsRow) fullDisposalSize() uint64 {
	return l.r.MustU64("full disposal size")
}

func (l lsRow) maxFraction() num.Decimal {
	return l.r.MustDecimal("max fraction consumed")
}
