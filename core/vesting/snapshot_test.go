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
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/paths"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotEngine(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)

	vegaPath := paths.New(t.TempDir())
	now := time.Now()

	te1 := newEngine(t)
	snapshotEngine1 := newSnapshotEngine(t, vegaPath, now, te1.engine)
	closeSnapshotEngine1 := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer closeSnapshotEngine1()

	require.NoError(t, snapshotEngine1.Start(ctx))

	setupMocks(t, te1)
	setupNetParams(ctx, t, te1)

	te1.engine.AddReward(context.Background(), "party1", "eth", num.NewUint(100), 4)
	te1.engine.AddReward(context.Background(), "party1", "btc", num.NewUint(150), 1)
	te1.engine.AddReward(context.Background(), "party1", "eth", num.NewUint(200), 0)

	nextEpoch(ctx, t, te1, time.Now())

	te1.engine.AddReward(context.Background(), "party2", "btc", num.NewUint(100), 2)
	te1.engine.AddReward(context.Background(), "party3", "btc", num.NewUint(100), 0)

	nextEpoch(ctx, t, te1, time.Now())

	te1.engine.AddReward(context.Background(), "party4", "eth", num.NewUint(100), 1)
	te1.engine.AddReward(context.Background(), "party5", "doge", num.NewUint(100), 0)

	// Take a snapshot.
	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)
	snapshottedEpoch := te1.currentEpoch

	// This is what must be replayed after snapshot restoration.
	replayFn := func(te *testSnapshotEngine) {
		te.engine.AddReward(context.Background(), "party6", "doge", num.NewUint(100), 3)

		nextEpoch(ctx, t, te, time.Now())

		te.engine.AddReward(context.Background(), "party7", "eth", num.NewUint(100), 2)
		te.engine.AddReward(context.Background(), "party8", "vega", num.NewUint(100), 10)

		nextEpoch(ctx, t, te, time.Now())
	}

	replayFn(te1)

	state1 := map[string][]byte{}
	for _, key := range te1.engine.Keys() {
		state, additionalProvider, err := te1.engine.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state1[key] = state
	}

	closeSnapshotEngine1()

	// Reload the engine using the previous snapshot.

	te2 := newEngine(t)
	snapshotEngine2 := newSnapshotEngine(t, vegaPath, now, te2.engine)
	defer snapshotEngine2.Close()

	setupMocks(t, te2)
	setupNetParams(ctx, t, te2)

	// Ensure the engine's epoch (and test helpers) starts at the same epoch the
	// first engine has been snapshotted.
	te2.currentEpoch = snapshottedEpoch
	te2.engine.OnEpochRestore(ctx, types.Epoch{
		Seq:    snapshottedEpoch,
		Action: vegapb.EpochAction_EPOCH_ACTION_START,
	})

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, snapshotEngine2.Start(ctx))

	// Comparing the hash after restoration, to ensure it produces the same result.
	hash2, _, _ := snapshotEngine2.Info()
	require.Equal(t, hash1, hash2)

	// Replaying the same commands after snapshot has been taken with first engine.
	replayFn(te2)

	state2 := map[string][]byte{}
	for _, key := range te2.engine.Keys() {
		state, additionalProvider, err := te2.engine.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state2[key] = state
	}

	for key := range state1 {
		assert.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}

func setupNetParams(ctx context.Context, t *testing.T, te *testSnapshotEngine) {
	t.Helper()

	require.NoError(t, te.engine.OnBenefitTiersUpdate(ctx, &vegapb.VestingBenefitTiers{
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
	}))

	require.NoError(t, te.engine.OnRewardVestingBaseRateUpdate(ctx, num.MustDecimalFromString("0.9")))
	require.NoError(t, te.engine.OnRewardVestingMinimumTransferUpdate(ctx, num.MustDecimalFromString("1")))
}

func setupMocks(t *testing.T, te *testSnapshotEngine) {
	t.Helper()

	te.asvm.EXPECT().GetRewardsVestingMultiplier(gomock.Any()).AnyTimes().Return(num.MustDecimalFromString("1"))
	te.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(dummyAsset{quantum: 10}), nil)
	te.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	te.parties.EXPECT().RelatedKeys(gomock.Any()).AnyTimes()
}
