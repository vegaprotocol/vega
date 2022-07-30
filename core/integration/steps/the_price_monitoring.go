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
	"fmt"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/integration/steps/market"
	types "code.vegaprotocol.io/vega/protos/vega"
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
