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
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/cucumber/godog"
)

func UpdateAsset(tbl *godog.Table, asset *stubs.AssetStub, col *collateral.Engine) error {
	rows := parseAssetsTable(tbl)
	for _, row := range rows {
		aRow := assetRow{row: row}
		aid := row.MustStr("id")
		asset.Register(
			aid,
			row.MustU64("decimal places"),
			aRow.maybeQuantum(),
		)
		err := col.PropagateAssetUpdate(context.Background(), types.Asset{
			ID: aid,
			Details: &types.AssetDetails{
				Quantum: aRow.quantum(),
				Symbol:  aid,
			},
		})
		if err != nil {
			if err == collateral.ErrAssetHasNotBeenEnabled {
				return fmt.Errorf("asset %q has not been enabled", aid)
			}
			return fmt.Errorf("couldn't enable asset %q: %w", aid, err)
		}
	}
	return nil
}

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
			return fmt.Errorf("couldn't enable asset %q: %w", aid, err)
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

func CreateNetworkTreasuryAccount(col *collateral.Engine, asset string) error {
	_ = col.GetOrCreateNetworkTreasuryAccount(context.Background(), asset)
	return nil
}
