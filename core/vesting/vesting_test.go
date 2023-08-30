// Copyright (c) 2023 Gobalsky Labs Limited
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

package vesting_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/common"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/vesting"
	"code.vegaprotocol.io/vega/core/vesting/mocks"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testEngine struct {
	*vesting.Engine

	ctrl   *gomock.Controller
	col    *mocks.MockCollateral
	asvm   *mocks.MockActivityStreakVestingMultiplier
	broker *mocks.MockBroker
	assets *mocks.MockAssets
}

func getTestEngine(t *testing.T) *testEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	col := mocks.NewMockCollateral(ctrl)
	asvm := mocks.NewMockActivityStreakVestingMultiplier(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	assets := mocks.NewMockAssets(ctrl)

	return &testEngine{
		Engine: vesting.New(
			logging.NewTestLogger(), col, asvm, broker, assets,
		),
		ctrl:   ctrl,
		col:    col,
		asvm:   asvm,
		broker: broker,
		assets: assets,
	}
}

func TestRewardMultiplier(t *testing.T) {
	v := getTestEngine(t)

	// set benefits tiers
	err := v.OnBenefitTiersUpdate(context.Background(), &vegapb.VestingBenefitTiers{
		Tiers: []*vegapb.VestingBenefitTier{
			{
				MinimumQuantumBalance: "10000",
				RewardMultiplier:      "1.5",
			},
			{
				MinimumQuantumBalance: "100000",
				RewardMultiplier:      "2",
			},
			{
				MinimumQuantumBalance: "500000",
				RewardMultiplier:      "2.5",
			},
		},
	})

	assert.NoError(t, err)

	v.col.EXPECT().GetAllVestingQuantumBalance("party1").Times(1).Return(num.UintZero())
	assert.Equal(t, num.DecimalOne(), v.GetRewardsBonusMultiplier("party1"))

	v.col.EXPECT().GetAllVestingQuantumBalance("party1").Times(1).Return(num.NewUint(10001))
	assert.Equal(t, num.MustDecimalFromString("1.5"), v.GetRewardsBonusMultiplier("party1"))

	v.col.EXPECT().GetAllVestingQuantumBalance("party1").Times(1).Return(num.NewUint(100001))
	assert.Equal(t, num.MustDecimalFromString("2"), v.GetRewardsBonusMultiplier("party1"))

	v.col.EXPECT().GetAllVestingQuantumBalance("party1").Times(1).Return(num.NewUint(500001))
	assert.Equal(t, num.MustDecimalFromString("2.5"), v.GetRewardsBonusMultiplier("party1"))
}

func TestDistributeAfterDelay(t *testing.T) {
	v := getTestEngine(t)

	// distribute 90% as the base rate,
	// so first we distribute some, then we get under the minimum value, and all the rest
	// is distributed
	v.OnRewardVestingBaseRateUpdate(context.Background(), num.MustDecimalFromString("0.9"))
	// this is multiplied by the quantume, so it will make it 100% of the quantum
	v.OnRewardVestingMinimumTransferUpdate(context.Background(), num.MustDecimalFromString("1"))

	// set the asvm to return always 1
	v.asvm.EXPECT().Get(gomock.Any()).AnyTimes().Return(num.MustDecimalFromString("1"))

	// set asset to return proper quantum
	v.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(dummyAsset{quantum: 10}), nil)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// Add a reward to be locked for 3 epochs then
	// we add a 100 of the reward.
	// it will be paid in 2 times, first 90,
	// then the remain 10,
	// and it'll be all
	v.AddReward("party1", "eth", num.NewUint(100), 3)
	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})
	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})
	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// now we expect 1 call to the collateral for the transfer of 90, for the transfer of the 90
	v.col.EXPECT().TransferVestedRewards(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, transfers []*types.Transfer) ([]*types.LedgerMovements, error) {
			assert.Len(t, transfers, 1)
			assert.Equal(t, int(transfers[0].Amount.Amount.Uint64()), 90)
			assert.Equal(t, transfers[0].Owner, "party1")
			assert.Equal(t, int(transfers[0].MinAmount.Uint64()), 90)
			assert.Equal(t, transfers[0].Amount.Asset, "eth")
			return nil, nil
		},
	)
	// one call to the broker
	v.broker.EXPECT().Send(gomock.Any()).Times(1)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// now we expect 1 call to the collateral for the transfer of 10, for the transfer of the 90, which is the whole remaining thing
	v.col.EXPECT().TransferVestedRewards(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, transfers []*types.Transfer) ([]*types.LedgerMovements, error) {
			assert.Len(t, transfers, 1)
			assert.Equal(t, int(transfers[0].Amount.Amount.Uint64()), 10)
			assert.Equal(t, transfers[0].Owner, "party1")
			assert.Equal(t, int(transfers[0].MinAmount.Uint64()), 10)
			assert.Equal(t, transfers[0].Amount.Asset, "eth")
			return nil, nil
		},
	)
	// one call to the broker
	v.broker.EXPECT().Send(gomock.Any()).Times(1)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// try it again and nothing happen
	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})
}

func TestDistributeWithNoDelay(t *testing.T) {
	v := getTestEngine(t)

	// distribute 90% as the base rate,
	// so first we distribute some, then we get under the minimum value, and all the rest
	// is distributed
	v.OnRewardVestingBaseRateUpdate(context.Background(), num.MustDecimalFromString("0.9"))
	// this is multiplied by the quantume, so it will make it 100% of the quantum
	v.OnRewardVestingMinimumTransferUpdate(context.Background(), num.MustDecimalFromString("1"))

	// set the asvm to return always 1
	v.asvm.EXPECT().Get(gomock.Any()).AnyTimes().Return(num.MustDecimalFromString("1"))

	// set asset to return proper quantum
	v.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(dummyAsset{quantum: 10}), nil)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// we add a 100 of the reward.
	// it will be paid in 2 times, first 90,
	// then the remain 10,
	// and it'll be all
	v.AddReward("party1", "eth", num.NewUint(100), 0)

	// now we expect 1 call to the collateral for the transfer of 90, for the transfer of the 90
	v.col.EXPECT().TransferVestedRewards(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, transfers []*types.Transfer) ([]*types.LedgerMovements, error) {
			assert.Len(t, transfers, 1)
			assert.Equal(t, int(transfers[0].Amount.Amount.Uint64()), 90)
			assert.Equal(t, transfers[0].Owner, "party1")
			assert.Equal(t, int(transfers[0].MinAmount.Uint64()), 90)
			assert.Equal(t, transfers[0].Amount.Asset, "eth")
			return nil, nil
		},
	)
	// one call to the broker
	v.broker.EXPECT().Send(gomock.Any()).Times(1)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// now we expect 1 call to the collateral for the transfer of 10, for the transfer of the 90, which is the whole remaining thing
	v.col.EXPECT().TransferVestedRewards(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, transfers []*types.Transfer) ([]*types.LedgerMovements, error) {
			assert.Len(t, transfers, 1)
			assert.Equal(t, int(transfers[0].Amount.Amount.Uint64()), 10)
			assert.Equal(t, transfers[0].Owner, "party1")
			assert.Equal(t, int(transfers[0].MinAmount.Uint64()), 10)
			assert.Equal(t, transfers[0].Amount.Asset, "eth")
			return nil, nil
		},
	)
	// one call to the broker
	v.broker.EXPECT().Send(gomock.Any()).Times(1)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// try it again and nothing happen
	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})
}

func TestDistributeWithStreakRate(t *testing.T) {
	v := getTestEngine(t)

	// distribute 90% as the base rate,
	// so first we distribute some, then we get under the minimum value, and all the rest
	// is distributed
	v.OnRewardVestingBaseRateUpdate(context.Background(), num.MustDecimalFromString("0.9"))
	// this is multiplied by the quantume, so it will make it 100% of the quantum
	v.OnRewardVestingMinimumTransferUpdate(context.Background(), num.MustDecimalFromString("1"))

	// set the asvm to return always 1
	v.asvm.EXPECT().Get(gomock.Any()).AnyTimes().Return(num.MustDecimalFromString("1.1"))

	// set asset to return proper quantum
	v.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(dummyAsset{quantum: 10}), nil)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// Add a reward to be locked for 3 epochs then
	// we add a 100 of the reward.
	// it will be paid in 2 times, first 90,
	// then the remain 10,
	// and it'll be all
	v.AddReward("party1", "eth", num.NewUint(100), 0)

	// now we expect 1 call to the collateral for the transfer of 99, for the transfer of the 99
	// this is 100 * 0.9 + 1.1
	v.col.EXPECT().TransferVestedRewards(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, transfers []*types.Transfer) ([]*types.LedgerMovements, error) {
			assert.Len(t, transfers, 1)
			assert.Equal(t, int(transfers[0].Amount.Amount.Uint64()), 99)
			assert.Equal(t, transfers[0].Owner, "party1")
			assert.Equal(t, int(transfers[0].MinAmount.Uint64()), 99)
			assert.Equal(t, transfers[0].Amount.Asset, "eth")
			return nil, nil
		},
	)
	// one call to the broker
	v.broker.EXPECT().Send(gomock.Any()).Times(1)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// now we expect 1 call to the collateral for the transfer of 10, for the transfer of the 90, which is the whole remaining thing
	v.col.EXPECT().TransferVestedRewards(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, transfers []*types.Transfer) ([]*types.LedgerMovements, error) {
			assert.Len(t, transfers, 1)
			assert.Equal(t, int(transfers[0].Amount.Amount.Uint64()), 1)
			assert.Equal(t, transfers[0].Owner, "party1")
			assert.Equal(t, int(transfers[0].MinAmount.Uint64()), 1)
			assert.Equal(t, transfers[0].Amount.Asset, "eth")
			return nil, nil
		},
	)
	// one call to the broker
	v.broker.EXPECT().Send(gomock.Any()).Times(1)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// try it again and nothing happen
	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})
}

func TestDistributeMultipleAfterDelay(t *testing.T) {
	v := getTestEngine(t)

	// distribute 90% as the base rate,
	// so first we distribute some, then we get under the minimum value, and all the rest
	// is distributed
	v.OnRewardVestingBaseRateUpdate(context.Background(), num.MustDecimalFromString("0.9"))
	// this is multiplied by the quantume, so it will make it 100% of the quantum
	v.OnRewardVestingMinimumTransferUpdate(context.Background(), num.MustDecimalFromString("1"))

	// set the asvm to return always 1
	v.asvm.EXPECT().Get(gomock.Any()).AnyTimes().Return(num.MustDecimalFromString("1"))

	// set asset to return proper quantum
	v.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(dummyAsset{quantum: 10}), nil)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// Add a reward to be locked for 2 epochs then
	// we add a 100 of the reward.
	v.AddReward("party1", "eth", num.NewUint(100), 2)
	// then another for 1 epoch
	v.AddReward("party1", "eth", num.NewUint(100), 1)
	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// now we expect 1 call to the collateral for the transfer of 90, for the transfer of the 90
	v.col.EXPECT().TransferVestedRewards(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, transfers []*types.Transfer) ([]*types.LedgerMovements, error) {
			assert.Len(t, transfers, 1)
			assert.Equal(t, int(transfers[0].Amount.Amount.Uint64()), 90)
			assert.Equal(t, transfers[0].Owner, "party1")
			assert.Equal(t, int(transfers[0].MinAmount.Uint64()), 90)
			assert.Equal(t, transfers[0].Amount.Asset, "eth")
			return nil, nil
		},
	)
	// one call to the broker
	v.broker.EXPECT().Send(gomock.Any()).Times(1)

	// this will deliver 100 more as well ready to be paid
	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// now we expect another transfer of 99 which is 110*0.9
	v.col.EXPECT().TransferVestedRewards(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, transfers []*types.Transfer) ([]*types.LedgerMovements, error) {
			assert.Len(t, transfers, 1)
			assert.Equal(t, int(transfers[0].Amount.Amount.Uint64()), 99)
			assert.Equal(t, transfers[0].Owner, "party1")
			assert.Equal(t, int(transfers[0].MinAmount.Uint64()), 99)
			assert.Equal(t, transfers[0].Amount.Asset, "eth")
			return nil, nil
		},
	)
	// one call to the broker
	v.broker.EXPECT().Send(gomock.Any()).Times(1)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// now we expect another transfer of 9 which is 110*0.9 floored
	// but it's actually defaulting to 10 which is the minimum acceptable transfer
	v.col.EXPECT().TransferVestedRewards(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, transfers []*types.Transfer) ([]*types.LedgerMovements, error) {
			assert.Len(t, transfers, 1)
			assert.Equal(t, int(transfers[0].Amount.Amount.Uint64()), 10)
			assert.Equal(t, transfers[0].Owner, "party1")
			assert.Equal(t, int(transfers[0].MinAmount.Uint64()), 10)
			assert.Equal(t, transfers[0].Amount.Asset, "eth")
			return nil, nil
		},
	)
	// one call to the broker
	v.broker.EXPECT().Send(gomock.Any()).Times(1)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// now we expect another transfer of 1 which is all that is left
	v.col.EXPECT().TransferVestedRewards(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, transfers []*types.Transfer) ([]*types.LedgerMovements, error) {
			assert.Len(t, transfers, 1)
			assert.Equal(t, int(transfers[0].Amount.Amount.Uint64()), 1)
			assert.Equal(t, transfers[0].Owner, "party1")
			assert.Equal(t, int(transfers[0].MinAmount.Uint64()), 1)
			assert.Equal(t, transfers[0].Amount.Asset, "eth")
			return nil, nil
		},
	)
	// one call to the broker
	v.broker.EXPECT().Send(gomock.Any()).Times(1)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})

	// try it again and nothing happen
	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})
}

type dummyAsset struct {
	quantum uint64
}

func (d dummyAsset) Type() *types.Asset {
	return &types.Asset{
		Details: &types.AssetDetails{
			Quantum: num.DecimalFromInt64(int64(d.quantum)),
		},
	}
}

func (dummyAsset) GetAssetClass() common.AssetClass { return common.ERC20 }
func (dummyAsset) IsValid() bool                    { return true }
func (dummyAsset) SetPendingListing()               {}
func (dummyAsset) SetRejected()                     {}
func (dummyAsset) SetEnabled()                      {}
func (dummyAsset) SetValid()                        {}
func (dummyAsset) String() string                   { return "" }
