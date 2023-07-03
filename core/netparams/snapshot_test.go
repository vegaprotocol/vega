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

package netparams_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotRestoreDependentNetParams(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)

	engine1 := getTestNetParams(t)
	engine1.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	now := time.Now()
	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	vegaPath := paths.New(t.TempDir())
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snapshot.DefaultConfig()

	snapshotEngine1, err := snapshot.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	snapshotEngine1.AddProviders(engine1)
	snapshotEngine1CloseFn := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer snapshotEngine1CloseFn()

	require.NoError(t, snapshotEngine1.Start(ctx))

	require.NoError(t, engine1.Update(ctx, netparams.MarketAuctionMinimumDuration, "1s"))
	marketAuctionMinimumDurationV1, err := engine1.Get(netparams.MarketAuctionMinimumDuration)
	require.NoError(t, err)
	require.Equal(t, "1s", marketAuctionMinimumDurationV1)

	require.NoError(t, engine1.Update(ctx, netparams.DelegationMinAmount, "100"))
	delegationMinAmountV1, err := engine1.Get(netparams.DelegationMinAmount)
	require.NoError(t, err)
	require.Equal(t, "100", delegationMinAmountV1)

	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	require.NoError(t, engine1.Update(ctx, netparams.GovernanceProposalMarketMinClose, "2h"))

	state1 := map[string][]byte{}
	for _, key := range engine1.Keys() {
		state, additionalProvider, err := engine1.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state1[key] = state
	}

	snapshotEngine1CloseFn()

	engine2 := getTestNetParams(t)
	engine2.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	snapshotEngine2, err := snapshot.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	defer snapshotEngine2.Close()

	snapshotEngine2.AddProviders(engine2)

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, snapshotEngine2.Start(ctx))

	// Comparing the hash after restoration, to ensure it produces the same result.
	hash2, _, _ := snapshotEngine2.Info()
	require.Equal(t, hash1, hash2)

	marketAuctionMinimumDurationV2, err := engine2.Get(netparams.MarketAuctionMinimumDuration)
	require.NoError(t, err)
	delegationMinAmountV2, err := engine2.Get(netparams.DelegationMinAmount)
	require.NoError(t, err)

	require.Equal(t, marketAuctionMinimumDurationV1, marketAuctionMinimumDurationV2)
	require.Equal(t, delegationMinAmountV1, delegationMinAmountV2)

	require.NoError(t, engine2.Update(ctx, netparams.GovernanceProposalMarketMinClose, "2h"))

	state2 := map[string][]byte{}
	for _, key := range engine2.Keys() {
		state, additionalProvider, err := engine2.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state2[key] = state
	}

	for key := range state1 {
		assert.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}
