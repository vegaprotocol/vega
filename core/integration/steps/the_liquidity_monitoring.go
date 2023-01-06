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
