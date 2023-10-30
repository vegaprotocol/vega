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

func TheLiquidityMonitoring(config *market.Config, table *godog.Table) error {
	rows := parseLiquidityMonitoringTable(table)
	for _, row := range rows {
		r := liquidityMonitoringRow{row: row}
		p := &types.LiquidityMonitoringParameters{
			TargetStakeParameters: &types.TargetStakeParameters{
				TimeWindow:    r.timeWindow(),
				ScalingFactor: r.scalingFactor(),
			},
			TriggeringRatio:  r.triggeringRatio(),
			AuctionExtension: r.auctionExtension(),
		}
		if err := config.LiquidityMonitoring.Add(r.name(), p); err != nil {
			return err
		}
	}
	return nil
}

func parseLiquidityMonitoringTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"name",
		"time window",
		"scaling factor",
		"triggering ratio",
	}, []string{
		"auction extension",
	})
}

type liquidityMonitoringRow struct {
	row RowWrapper
}

func (r liquidityMonitoringRow) name() string {
	return r.row.MustStr("name")
}

func (r liquidityMonitoringRow) timeWindow() int64 {
	tw := r.row.MustDurationStr("time window")
	return int64(tw.Seconds())
}

func (r liquidityMonitoringRow) scalingFactor() float64 {
	return r.row.MustF64("scaling factor")
}

func (r liquidityMonitoringRow) triggeringRatio() string {
	return r.row.MustStr("triggering ratio")
}

func (r liquidityMonitoringRow) auctionExtension() int64 {
	if !r.row.HasColumn("auction extension") {
		return 0
	}
	return r.row.MustI64("auction extension")
}
