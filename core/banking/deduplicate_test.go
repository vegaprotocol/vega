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

package banking_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/erc20"
	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"

	"github.com/stretchr/testify/require"
)

func TestAssetActionDeduplication(t *testing.T) {
	ctx := context.Background()

	eng := getTestEngine(t)
	eng.OnPrimaryEthChainIDUpdated("1")

	id1 := vgrand.RandomStr(5)
	txHash1 := vgrand.RandomStr(5)
	assetID1 := vgrand.RandomStr(5)
	assetList1 := &types.ERC20AssetList{
		VegaAssetID: assetID1,
	}
	erc20Asset, err := erc20.New(assetID1, &types.AssetDetails{
		Source: &types.AssetDetailsErc20{
			ERC20: &types.ERC20{
				ChainID:           "",
				ContractAddress:   "",
				LifetimeLimit:     nil,
				WithdrawThreshold: nil,
			},
		},
	}, nil, nil)
	require.NoError(t, err)
	asset1 := assets.NewAsset(erc20Asset)

	t.Run("Generate asset list", func(t *testing.T) {
		eng.tsvc.EXPECT().GetTimeNow().Times(1).Return(time.Now())
		eng.assets.EXPECT().Get(assetID1).Times(1).Return(asset1, nil)
		require.NoError(t, eng.EnableERC20(ctx, assetList1, id1, 1000, 1000, txHash1, ""))

		// Validate the asset list.
		eng.witness.f(eng.witness.r, true)

		// These expectations shows the asset action is processed.
		eng.assets.EXPECT().Get(assetID1).Times(1).Return(asset1, nil)
		eng.assets.EXPECT().Enable(ctx, assetID1).Times(1).Return(nil)
		eng.col.EXPECT().EnableAsset(ctx, *asset1.ToAssetType()).Times(1).Return(nil)

		// Trigger processing of asset actions, and deduplication.
		eng.OnTick(ctx, time.Now())
	})

	t.Run("Generate duplicated asset list and ", func(t *testing.T) {
		eng.tsvc.EXPECT().GetTimeNow().Times(1).Return(time.Now())
		eng.assets.EXPECT().Get(assetID1).Times(1).Return(asset1, nil)
		require.NoError(t, eng.EnableERC20(ctx, assetList1, id1, 1000, 1000, txHash1, ""))

		// Validate the asset list.
		eng.witness.f(eng.witness.r, true)

		// We expect nothing as the asset action should be deduplicated.

		// Trigger processing of asset actions, and deduplication.
		eng.OnTick(ctx, time.Now())
	})

	// This covers the scenario where the event is replayed but with the chain ID
	// set, which might happen with the introduction of the second bridge. We have
	// to ensure the event is acknowledge as a duplicate.
	t.Run("Generate a duplicated event but updated with the chain ID", func(t *testing.T) {
		eng.tsvc.EXPECT().GetTimeNow().Times(1).Return(time.Now())
		eng.assets.EXPECT().Get(assetID1).Times(1).Return(asset1, nil)
		require.NoError(t, eng.EnableERC20(ctx, assetList1, id1, 1000, 1000, txHash1, "1"))

		// Validate the asset list.
		eng.witness.f(eng.witness.r, true)

		// We expect nothing as the asset action should be deduplicated.

		// Trigger processing of asset actions, and deduplication.
		eng.OnTick(ctx, time.Now())
	})
}
