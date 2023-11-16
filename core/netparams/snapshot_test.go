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

package netparams_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

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

func TestPatchConfirmationsTo64(t *testing.T) {
	//ctx := vgtest.VegaContext("chainid", 100)

	engine1 := getTestNetParams(t)
	engine1.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	v := vega.EthereumConfig{}
	err := engine1.GetJSONStruct(netparams.BlockchainsEthereumConfig, &v)
	require.NoError(t, err)
	assert.NotEqual(t, uint32(64), v.Confirmations)

	// get a snapshot
	b, _, err := engine1.GetState("all")
	require.NoError(t, err)

	// new engine
	engine2 := getTestNetParams(t)
	engine2.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	ctx := vgcontext.WithSnapshotInfo(context.Background(), "v0.73.4", true)

	snap := &snapshotpb.Payload{}
	err = proto.Unmarshal(b, snap)
	require.Nil(t, err)

	_, err = engine2.LoadState(ctx, types.PayloadFromProto(snap))
	require.NoError(t, err)

	v = vega.EthereumConfig{}
	err = engine2.GetJSONStruct(netparams.BlockchainsEthereumConfig, &v)
	require.NoError(t, err)
	assert.Equal(t, uint32(64), v.Confirmations)

}
