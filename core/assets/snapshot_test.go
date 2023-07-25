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
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func getEngineAndSnapshotEngine(t *testing.T) (*testService, *snp.Engine) {
	t.Helper()
	as := getTestService(t)
	now := time.Now()
	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snp.NewDefaultConfig()
	config.Storage = "memory"
	snapshotEngine, _ := snp.New(context.Background(), &paths.DefaultPaths{}, config, log, timeService, statsData.Blockchain)
	snapshotEngine.AddProviders(as)
	require.NoError(t, snapshotEngine.ClearAndInitialise())
	return as, snapshotEngine
}

func TestSnapshotRoundtripViaEngine(t *testing.T) {
	ctx := vgcontext.WithTraceID(vgcontext.WithBlockHeight(context.Background(), 100), "0xDEADBEEF")
	ctx = vgcontext.WithChainID(ctx, "chainid")

	as, snapshotEngine := getEngineAndSnapshotEngine(t)
	defer snapshotEngine.Close()
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

	_, err = as.NewAsset(ctx, "asset3", &types.AssetDetails{
		Source: &types.AssetDetailsBuiltinAsset{},
	})
	require.Nil(t, err)
	_, err = as.NewAsset(ctx, "asset4", &types.AssetDetails{
		Source: &types.AssetDetailsBuiltinAsset{},
	})
	require.Nil(t, err)

	_, err = snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	snaps, err := snapshotEngine.List()
	require.NoError(t, err)
	snap1 := snaps[0]

	asLoad, snapshotEngineLoad := getEngineAndSnapshotEngine(t)
	asLoad.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	require.NoError(t, snapshotEngineLoad.ReceiveSnapshot(snap1))
	require.NoError(t, snapshotEngineLoad.ApplySnapshot(ctx))
	_, err = snapshotEngineLoad.CheckLoaded()
	require.NoError(t, err)
	defer snapshotEngineLoad.Close()

	err = as.Enable(ctx, "asset3")
	require.Nil(t, err)

	err = asLoad.Enable(ctx, "asset3")
	require.Nil(t, err)

	_, err = as.NewAsset(ctx, "asset5", &types.AssetDetails{
		Source: &types.AssetDetailsBuiltinAsset{},
	})
	require.NoError(t, err)

	_, err = asLoad.NewAsset(ctx, "asset5", &types.AssetDetails{
		Source: &types.AssetDetailsBuiltinAsset{},
	})
	require.NoError(t, err)

	b, err := snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	bLoad, err := snapshotEngineLoad.Snapshot(ctx)
	require.NoError(t, err)
	require.True(t, bytes.Equal(b, bLoad))
}

// test round trip of active snapshot hash and serialisation.
func TestActiveSnapshotRoundTrip(t *testing.T) {
	ctx := vgcontext.WithTraceID(vgcontext.WithBlockHeight(context.Background(), 100), "0xDEADBEEF")
	ctx = vgcontext.WithChainID(ctx, "chainid")

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
	ctx := vgcontext.WithTraceID(vgcontext.WithBlockHeight(context.Background(), 100), "0xDEADBEEF")
	ctx = vgcontext.WithChainID(ctx, "chainid")
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
