// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/assets/mocks"
	"code.vegaprotocol.io/vega/integration/stubs"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	snp "code.vegaprotocol.io/vega/snapshot"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func testAssets(t *testing.T) *assets.Service {
	t.Helper()
	conf := assets.NewDefaultConfig()
	logger := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	ts := mocks.NewMockTimeService(ctrl)
	ts.EXPECT().NotifyOnTick(gomock.Any()).Times(1)
	as := assets.New(logger, conf, nil, nil, ts, true)
	return as
}

func getEngineAndSnapshotEngine(t *testing.T) (*assets.Service, *snp.Engine) {
	t.Helper()
	as := testAssets(t)
	now := time.Now()
	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig(), "", "")
	config := snp.NewDefaultConfig()
	config.Storage = "memory"
	snapshotEngine, _ := snp.New(context.Background(), &paths.DefaultPaths{}, config, log, timeService, statsData.Blockchain)
	snapshotEngine.AddProviders(as)
	snapshotEngine.ClearAndInitialise()
	return as, snapshotEngine
}

func TestSnapshotRoundtripViaEngine(t *testing.T) {
	ctx := vgcontext.WithTraceID(vgcontext.WithBlockHeight(context.Background(), 100), "0xDEADBEEF")
	ctx = vgcontext.WithChainID(ctx, "chainid")

	as, snapshotEngine := getEngineAndSnapshotEngine(t)
	defer snapshotEngine.Close()

	_, err := as.NewAsset("asset1", &types.AssetDetails{
		Source: &types.AssetDetailsBuiltinAsset{},
	})
	require.Nil(t, err)
	err = as.Enable("asset1")
	require.Nil(t, err)
	_, err = as.NewAsset("asset2", &types.AssetDetails{
		Source: &types.AssetDetailsBuiltinAsset{},
	})
	require.Nil(t, err)
	err = as.Enable("asset2")
	require.Nil(t, err)

	_, err = as.NewAsset("asset3", &types.AssetDetails{
		Source: &types.AssetDetailsBuiltinAsset{},
	})
	require.Nil(t, err)
	_, err = as.NewAsset("asset4", &types.AssetDetails{
		Source: &types.AssetDetailsBuiltinAsset{},
	})
	require.Nil(t, err)

	_, err = snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	snaps, err := snapshotEngine.List()
	require.NoError(t, err)
	snap1 := snaps[0]

	asLoad, snapshotEngineLoad := getEngineAndSnapshotEngine(t)
	snapshotEngineLoad.ReceiveSnapshot(snap1)
	snapshotEngineLoad.ApplySnapshot(ctx)
	snapshotEngineLoad.CheckLoaded()
	defer snapshotEngineLoad.Close()

	err = as.Enable("asset3")
	require.Nil(t, err)

	err = asLoad.Enable("asset3")
	require.Nil(t, err)

	as.NewAsset("asset5", &types.AssetDetails{
		Source: &types.AssetDetailsBuiltinAsset{},
	})

	asLoad.NewAsset("asset5", &types.AssetDetails{
		Source: &types.AssetDetailsBuiltinAsset{},
	})

	b, err := snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	bLoad, err := snapshotEngineLoad.Snapshot(ctx)
	require.NoError(t, err)
	require.True(t, bytes.Equal(b, bLoad))
}

// test round trip of active snapshot hash and serialisation.
func TestActiveSnapshotRoundTrip(t *testing.T) {
	activeKey := (&types.PayloadActiveAssets{}).Key()
	for i := 0; i < 10; i++ {
		as := testAssets(t)
		_, err := as.NewAsset("asset1", &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)
		err = as.Enable("asset1")
		require.Nil(t, err)
		_, err = as.NewAsset("asset2", &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)
		err = as.Enable("asset2")
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
		proto.Unmarshal(state, &active)
		payload := types.PayloadFromProto(&active)

		_, err = as.LoadState(context.Background(), payload)
		require.Nil(t, err)
		statePostReload, _, _ := as.GetState(activeKey)
		require.True(t, bytes.Equal(state, statePostReload))
	}
}

// test round trip of active snapshot serialisation.
func TestPendingSnapshotRoundTrip(t *testing.T) {
	pendingKey := (&types.PayloadPendingAssets{}).Key()

	for i := 0; i < 10; i++ {
		as := testAssets(t)
		_, err := as.NewAsset("asset1", &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)
		_, err = as.NewAsset("asset2", &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)

		// get the serialised state
		state, _, err := as.GetState(pendingKey)
		require.Nil(t, err)

		// verify state is consistent in the absence of change
		stateNoChange, _, err := as.GetState(pendingKey)
		require.Nil(t, err)
		require.True(t, bytes.Equal(state, stateNoChange))

		// reload the state
		var pending snapshot.Payload
		proto.Unmarshal(state, &pending)
		payload := types.PayloadFromProto(&pending)

		_, err = as.LoadState(context.Background(), payload)
		require.Nil(t, err)
		statePostReload, _, _ := as.GetState(pendingKey)
		require.True(t, bytes.Equal(state, statePostReload))
	}
}
