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

package vesting_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/vesting"
	"code.vegaprotocol.io/vega/core/vesting/mocks"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testSnapshotEngine struct {
	*vesting.SnapshotEngine

	ctrl   *gomock.Controller
	col    *mocks.MockCollateral
	asvm   *mocks.MockActivityStreakVestingMultiplier
	broker *mocks.MockBroker
	assets *mocks.MockAssets
}

func getTestSnapshotEngine(t *testing.T) *testSnapshotEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	col := mocks.NewMockCollateral(ctrl)
	asvm := mocks.NewMockActivityStreakVestingMultiplier(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	assets := mocks.NewMockAssets(ctrl)

	return &testSnapshotEngine{
		SnapshotEngine: vesting.NewSnapshotEngine(
			logging.NewTestLogger(), col, asvm, broker, assets,
		),
		ctrl:   ctrl,
		col:    col,
		asvm:   asvm,
		broker: broker,
		assets: assets,
	}
}

func TestSnapshot(t *testing.T) {
	v1 := getTestSnapshotEngine(t)
	setDefaults(t, v1)

	// set couple of rewards
	v1.AddReward("party1", "eth", num.NewUint(100), 4)
	v1.AddReward("party1", "btc", num.NewUint(150), 1)
	v1.AddReward("party1", "eth", num.NewUint(200), 0)
	v1.AddReward("party2", "btc", num.NewUint(100), 2)
	v1.AddReward("party3", "btc", num.NewUint(100), 0)
	v1.AddReward("party4", "eth", num.NewUint(100), 1)
	v1.AddReward("party5", "doge", num.NewUint(100), 0)
	v1.AddReward("party5", "btc", num.NewUint(1420), 1)
	v1.AddReward("party6", "doge", num.NewUint(100), 3)
	v1.AddReward("party7", "eth", num.NewUint(100), 2)
	v1.AddReward("party8", "vega", num.NewUint(100), 10)

	state1, _, err := v1.GetState(vesting.VestingKey)
	assert.NoError(t, err)
	assert.NotNil(t, state1)

	ppayload := &snapshotpb.Payload{}
	err = proto.Unmarshal(state1, ppayload)
	assert.NoError(t, err)

	v2 := getTestSnapshotEngine(t)
	setDefaults(t, v2)
	_, err = v2.LoadState(context.Background(), types.PayloadFromProto(ppayload))
	assert.NoError(t, err)

	// now assert the v2 produce the same state
	state2, _, err := v2.GetState(vesting.VestingKey)
	assert.NoError(t, err)
	assert.NotNil(t, state2)

	assert.Equal(t, state1, state2)

	// now move a couple of epoch for good measure
	epochsForward(t, v1)
	epochsForward(t, v2)

	// now assert the v2 produce the same state
	state1, _, err = v1.GetState(vesting.VestingKey)
	assert.NoError(t, err)
	assert.NotNil(t, state1)
	state2, _, err = v2.GetState(vesting.VestingKey)
	assert.NoError(t, err)
	assert.NotNil(t, state2)

	assert.Equal(t, state1, state2)
}

func epochsForward(t *testing.T, v *testSnapshotEngine) {
	t.Helper()

	// expect at least 3 transfers and events call, 1 per epoch move
	v.col.EXPECT().TransferVestedRewards(gomock.Any(), gomock.Any()).Times(3).Return(nil, nil)
	v.col.EXPECT().GetAllVestingQuantumBalance(gomock.Any()).AnyTimes().Return(num.UintZero())
	v.broker.EXPECT().Send(gomock.Any()).Times(6)

	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})
	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})
	v.OnEpochEvent(context.Background(), types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
	})
}

func setDefaults(t *testing.T, v *testSnapshotEngine) {
	t.Helper()
	v.OnRewardVestingBaseRateUpdate(context.Background(), num.MustDecimalFromString("0.9"))
	v.OnRewardVestingMinimumTransferUpdate(context.Background(), num.MustDecimalFromString("1"))
	v.asvm.EXPECT().GetRewardsVestingMultiplier(gomock.Any()).AnyTimes().Return(num.MustDecimalFromString("1"))
	v.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(
		assets.NewAsset(dummyAsset{quantum: 10}), nil,
	)
}
