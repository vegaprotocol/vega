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
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/cucumber/godog"
)

func RegisterAsset(tbl *godog.Table, asset *stubs.AssetStub, col *collateral.Engine) error {
	rows := parseAssetsTable(tbl)
	for _, row := range rows {
		aRow := assetRow{row: row}
		aid := row.MustStr("id")
		asset.Register(
			aid,
			row.MustU64("decimal places"),
			aRow.maybeQuantum(),
		)
		err := col.EnableAsset(context.Background(), types.Asset{
			ID: aid,
			Details: &types.AssetDetails{
				Quantum: aRow.quantum(),
				Symbol:  aid,
			},
		})
		if err != nil {
			if err == collateral.ErrAssetAlreadyEnabled {
				return fmt.Errorf("asset %s was already enabled, perhaps when defining markets, order of steps should be swapped", aid)
			}
			return fmt.Errorf("couldn't enable asset(%s): %v", aid, err)
		}
	}
	return nil
}

func parseAssetsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"decimal places",
	}, []string{
		"quantum",
	})
}

type assetRow struct {
	row RowWrapper
}

func (r assetRow) quantum() num.Decimal {
	if !r.row.HasColumn("quantum") {
		return num.DecimalOne()
	}
	return r.row.MustDecimal("quantum")
}

func (r assetRow) maybeQuantum() *num.Decimal {
	if !r.row.HasColumn("quantum") {
		return nil
	}

	return ptr.From(r.row.MustDecimal("quantum"))
}
