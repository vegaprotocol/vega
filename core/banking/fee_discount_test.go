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

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/builtin"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestBankingTransactionFeeDiscount(t *testing.T) {
	party := "party-1"
	asset := assets.NewAsset(builtin.New("vega", &types.AssetDetails{
		Name:    "vega",
		Symbol:  "vega",
		Quantum: num.DecimalFromFloat(10),
	}))
	assetID := asset.Type().ID

	t.Run("decay amount", func(t *testing.T) {
		eng := getTestEngine(t)

		ctx := context.Background()
		eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
		eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
		eng.assets.EXPECT().Get(gomock.Any()).Return(asset, nil).AnyTimes()

		eng.OnTransferFeeDiscountDecayFractionUpdate(context.Background(), num.DecimalFromFloat(0.5))
		eng.OnTransferFeeDiscountMinimumTrackedAmountUpdate(context.Background(), num.DecimalFromFloat(1))

		asset2 := "asset-2"
		party2 := "party-2"

		assert.Equal(t, "0", eng.AvailableFeeDiscount(assetID, party).String())
		assert.Equal(t, "0", eng.AvailableFeeDiscount(asset2, party2).String())
		eng.RegisterTradingFees(ctx, assetID, map[string]*num.Uint{party: num.NewUint(25)})
		eng.RegisterTradingFees(ctx, assetID, map[string]*num.Uint{party: num.NewUint(25)})
		eng.RegisterTradingFees(ctx, asset2, map[string]*num.Uint{party2: num.NewUint(20)})
		eng.OnEpoch(ctx, types.Epoch{Action: proto.EpochAction_EPOCH_ACTION_END})
		assert.Equal(t, "50", eng.AvailableFeeDiscount(assetID, party).String())
		assert.Equal(t, "20", eng.AvailableFeeDiscount(asset2, party2).String())
		eng.RegisterTradingFees(ctx, assetID, nil)
		eng.OnEpoch(ctx, types.Epoch{Action: proto.EpochAction_EPOCH_ACTION_END})
		// decay by half
		assert.Equal(t, "25", eng.AvailableFeeDiscount(assetID, party).String())
		assert.Equal(t, "10", eng.AvailableFeeDiscount(asset2, party2).String())
		eng.RegisterTradingFees(ctx, assetID, nil)
		eng.OnEpoch(ctx, types.Epoch{Action: proto.EpochAction_EPOCH_ACTION_END})
		// decay by half
		assert.Equal(t, "12", eng.AvailableFeeDiscount(assetID, party).String())
		assert.Equal(t, "0", eng.AvailableFeeDiscount(asset2, party2).String())
		eng.RegisterTradingFees(ctx, assetID, nil)
		eng.OnEpoch(ctx, types.Epoch{Action: proto.EpochAction_EPOCH_ACTION_END})

		// decay by half but it's 0 because decayed amount (6) is less then
		// asset quantum x TransferFeeDiscountMinimumTrackedAmount (10 x 1)
		assert.Equal(t, "0", eng.AvailableFeeDiscount(assetID, party).String())
	})

	t.Run("apply fee discount", func(t *testing.T) {
		eng := getTestEngine(t)

		ctx := context.Background()
		eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
		eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
		eng.assets.EXPECT().Get(gomock.Any()).Return(asset, nil).AnyTimes()

		eng.OnTransferFeeDiscountDecayFractionUpdate(context.Background(), num.DecimalFromFloat(0.5))

		assert.Equal(t, "0", eng.AvailableFeeDiscount(assetID, party).String())

		// expect the whole fee to be paid
		discountedFee, discount := eng.ApplyFeeDiscount(ctx, assetID, party, num.NewUint(5))
		assert.Equal(t, "5", discountedFee.String())
		assert.Equal(t, "0", discount.String())
		eng.RegisterTradingFees(ctx, assetID, map[string]*num.Uint{party: num.NewUint(10)})
		// move to another epoch
		eng.OnEpoch(ctx, types.Epoch{Action: proto.EpochAction_EPOCH_ACTION_END})

		assert.Equal(t, "10", eng.AvailableFeeDiscount(assetID, party).String())

		// expect discount of 10 to be applied
		discountedFee, discount = eng.ApplyFeeDiscount(ctx, assetID, party, num.NewUint(15))
		assert.Equal(t, "5", discountedFee.String())
		assert.Equal(t, "10", discount.String())
		eng.RegisterTradingFees(ctx, assetID, map[string]*num.Uint{party: num.NewUint(20)})
		// move to another epoch
		eng.OnEpoch(ctx, types.Epoch{Action: proto.EpochAction_EPOCH_ACTION_END})

		assert.Equal(t, "20", eng.AvailableFeeDiscount(assetID, party).String())

		// expect discount of 3 to be applied
		discountedFee, discount = eng.ApplyFeeDiscount(ctx, assetID, party, num.NewUint(3))
		assert.Equal(t, "0", discountedFee.String())
		assert.Equal(t, "3", discount.String())

		assert.Equal(t, "17", eng.AvailableFeeDiscount(assetID, party).String())

		eng.RegisterTradingFees(ctx, assetID, map[string]*num.Uint{party: num.NewUint(5)})
		// move to another epoch
		eng.OnEpoch(ctx, types.Epoch{Action: proto.EpochAction_EPOCH_ACTION_END})

		// it's 13 because 9 was decayed and extra 5 added = 17-8+5
		assert.Equal(t, "13", eng.AvailableFeeDiscount(assetID, party).String())

		// expect discount of 4 to be applied
		discountedFee, discount = eng.ApplyFeeDiscount(ctx, assetID, party, num.NewUint(4))
		assert.Equal(t, "0", discountedFee.String())
		assert.Equal(t, "4", discount.String())
	})
}
