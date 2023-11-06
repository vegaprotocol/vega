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
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/steps/market"
	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func ThePriceMonitoring(config *market.Config, name string, table *godog.Table) error {
	rows := parsePriceMonitoringTable(table)
	triggers := make([]*types.PriceMonitoringTrigger, 0, len(rows))
	for _, r := range rows {
		row := priceMonitoringRow{row: r}
		p := &types.PriceMonitoringTrigger{
			Horizon:          row.horizon(),
			Probability:      fmt.Sprintf("%0.16f", row.probability()),
			AuctionExtension: row.auctionExtension(),
		}
		triggers = append(triggers, p)
	}

	return config.PriceMonitoring.Add(
		name,
		&types.PriceMonitoringSettings{
			Parameters: &types.PriceMonitoringParameters{
				Triggers: triggers,
			},
		},
	)
}

func parsePriceMonitoringTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"horizon",
		"probability",
		"auction extension",
	}, []string{})
}

type priceMonitoringRow struct {
	row RowWrapper
}

func (r priceMonitoringRow) horizon() int64 {
	return r.row.MustI64("horizon")
}

func (r priceMonitoringRow) probability() float64 {
	return r.row.MustF64("probability")
}

func (r priceMonitoringRow) auctionExtension() int64 {
	return r.row.MustI64("auction extension")
}
