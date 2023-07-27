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

package assets_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	snp "code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func getEngineAndSnapshotEngine(t *testing.T, vegaPath paths.Paths) (*testService, *snp.Engine) {
	t.Helper()
	as := getTestService(t)
	as.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	now := time.Now()
	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snp.DefaultConfig()
	snapshotEngine, err := snp.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	snapshotEngine.AddProviders(as)
	return as, snapshotEngine
}

func TestSnapshotRoundTripViaEngine(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)

	vegaPath := paths.New(t.TempDir())

	assetEngine1, snapshotEngine1 := getEngineAndSnapshotEngine(t, vegaPath)
	closeSnapshotEngine1 := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer closeSnapshotEngine1()

	require.NoError(t, snapshotEngine1.Start(context.Background()))

	for i := 0; i < 3; i++ {
		assetName := vgrand.RandomStr(5)

		_, err := assetEngine1.NewAsset(ctx, assetName, &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)

		err = assetEngine1.Enable(ctx, assetName)
		require.Nil(t, err)
	}

	snapshotHash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	closeSnapshotEngine1()

	// Reload the engine using the previous snapshot.

	_, snapshotEngine2 := getEngineAndSnapshotEngine(t, vegaPath)
	defer snapshotEngine2.Close()

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, snapshotEngine2.Start(ctx))

	snapshotHash2, _, _ := snapshotEngine2.Info()

	require.Equal(t, snapshotHash1, snapshotHash2)
}

// test round trip of active snapshot hash and serialisation.
func TestActiveSnapshotRoundTrip(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)

	activeKey := (&types.PayloadActiveAssets{}).Key()
	for i := 0; i < 10; i++ {
		as := getTestService(t)
		as.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		_, err := as.NewAsset(ctx, "asset1", &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)
		err = as.Enable(ctx, "asset1")
		require.Nil(t, err)
		_, err = as.NewAsset(ctx, "asset2", &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)
		err = as.Enable(ctx, "asset2")
		require.Nil(t, err)

		// get the serialised state
		state, _, err := as.GetState(activeKey)
		require.Nil(t, err)

		// verify state is consistent in the absence of change
		stateNoChange, _, err := as.GetState(activeKey)
		require.Nil(t, err)

		require.True(t, bytes.Equal(state, stateNoChange))

		// reload the state
		var active snapshot.Payload
		require.NoError(t, proto.Unmarshal(state, &active))
		payload := types.PayloadFromProto(&active)

		_, err = as.LoadState(context.Background(), payload)
		require.Nil(t, err)
		statePostReload, _, _ := as.GetState(activeKey)
		require.True(t, bytes.Equal(state, statePostReload))
	}
}

// test round trip of active snapshot serialisation.
func TestPendingSnapshotRoundTrip(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)
	pendingKey := (&types.PayloadPendingAssets{}).Key()

	for i := 0; i < 10; i++ {
		as := getTestService(t)
		as.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		_, err := as.NewAsset(ctx, "asset1", &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)
		assetID2, err := as.NewAsset(ctx, "asset2", &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)

		// set asset 2 as pending_listing
		require.NoError(t, as.SetPendingListing(ctx, assetID2))

		// get the serialised state
		state, _, err := as.GetState(pendingKey)
		require.Nil(t, err)

		// verify state is consistent in the absence of change
		stateNoChange, _, err := as.GetState(pendingKey)
		require.Nil(t, err)
		require.True(t, bytes.Equal(state, stateNoChange))

		// reload the state
		var pending snapshot.Payload
		require.NoError(t, proto.Unmarshal(state, &pending))
		payload := types.PayloadFromProto(&pending)

		_, err = as.LoadState(context.Background(), payload)
		require.Nil(t, err)
		statePostReload, _, _ := as.GetState(pendingKey)
		require.True(t, bytes.Equal(state, statePostReload))
	}
}
