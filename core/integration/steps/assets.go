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

	"github.com/cucumber/godog"
)

func RegisterAsset(tbl *godog.Table, asset *stubs.AssetStub, col *collateral.Engine) error {
	rows := StrictParseTable(tbl, []string{
		"id",
		"decimal places",
	}, nil)
	toEnable := []string{}
	for _, row := range rows {
		aid := row.MustStr("id")
		asset.Register(
			aid,
			row.MustU64("decimal places"),
		)
		toEnable = append(toEnable, aid)
	}
	return enableAssets(toEnable, col)
}

func enableAssets(ids []string, collateralEngine *collateral.Engine) error {
	for _, assetToEnable := range ids {
		err := collateralEngine.EnableAsset(context.Background(), types.Asset{
			ID: assetToEnable,
			Details: &types.AssetDetails{
				Quantum: num.DecimalOne(),
				Symbol:  assetToEnable,
			},
		})
		if err != nil && err != collateral.ErrAssetAlreadyEnabled {
			return fmt.Errorf("couldn't enable asset(%s): %v", assetToEnable, err)
		}
	}
	return nil
}
