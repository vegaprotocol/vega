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

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/core/integration/steps/market"
)

func TheFeesConfiguration(config *market.Config, name string, table *godog.Table) error {
	row := feesConfigRow{row: parseFeesConfigTable(table)}

	return config.FeesConfig.Add(name, &types.Fees{
		Factors: &types.FeeFactors{
			InfrastructureFee: row.infrastructureFee(),
			MakerFee:          row.makerFee(),
		},
	})
}

func parseFeesConfigTable(table *godog.Table) RowWrapper {
	return StrictParseFirstRow(table, []string{
		"maker fee",
		"infrastructure fee",
	}, []string{})
}

type feesConfigRow struct {
	row RowWrapper
}

func (r feesConfigRow) makerFee() string {
	return r.row.MustStr("maker fee")
}

func (r feesConfigRow) infrastructureFee() string {
	return r.row.MustStr("infrastructure fee")
}
