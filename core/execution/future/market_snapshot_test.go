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

package future_test

import (
	"context"
	"errors"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/collateral"
	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/execution/future"
	"code.vegaprotocol.io/vega/core/fee"
	fmock "code.vegaprotocol.io/vega/core/fee/mocks"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/settlement"
	"code.vegaprotocol.io/vega/core/types"
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreSettledMarket(t *testing.T) {
	tm := getSettledMarket(t)
	em := tm.market.GetState()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	oracleEngine := mocks.NewMockOracleEngine(ctrl)

	var unsubs uint64
	unsubscribe := func(_ context.Context, id spec.SubscriptionID) { unsubs++ }
	oracleEngine.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(spec.SubscriptionID(1), unsubscribe, nil)
	oracleEngine.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(spec.SubscriptionID(2), unsubscribe, nil)

	snap, err := newMarketFromSnapshot(t, context.Background(), ctrl, em, oracleEngine)
	require.NoError(t, err)
	require.NotEmpty(t, snap)

	// check the market is restored settled and that we have unsubscribed the two oracles
	assert.Equal(t, types.MarketStateSettled, snap.State())
	assert.Equal(t, uint64(2), unsubs)
	closed := snap.OnTick(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()), time.Now())
	assert.True(t, closed)
}

func TestRestoreClosedMarket(t *testing.T) {
	tm := getActiveMarket(t)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	changes := &types.MarketStateUpdateConfiguration{
		MarketID:        tm.mktCfg.ID,
		SettlementPrice: num.UintOne(),
		UpdateType:      types.MarketStateUpdateTypeTerminate,
	}
	tm.market.UpdateMarketState(ctx, changes)
	em := tm.market.GetState()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	oracleEngine := mocks.NewMockOracleEngine(ctrl)

	var unsubs uint64
	unsubscribe := func(_ context.Context, id spec.SubscriptionID) { unsubs++ }
	oracleEngine.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(spec.SubscriptionID(1), unsubscribe, nil)
	oracleEngine.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(spec.SubscriptionID(2), unsubscribe, nil)

	snap, err := newMarketFromSnapshot(t, context.Background(), ctrl, em, oracleEngine)
	require.NoError(t, err)
	require.NotEmpty(t, snap)

	// check the market is restored settled and that we have unsubscribed the two oracles
	assert.Equal(t, types.MarketStateClosed, snap.State())
	assert.Equal(t, uint64(2), unsubs)
	closed := snap.OnTick(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()), time.Now())
	assert.True(t, closed)
}

func TestRestoreTerminatedMarket(t *testing.T) {
	tm := getTerminatedMarket(t)
	em := tm.market.GetState()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	oracleEngine := mocks.NewMockOracleEngine(ctrl)

	var termUnsub bool
	unsubscribe := func(_ context.Context, id spec.SubscriptionID) {
		if id == spec.SubscriptionID(2) {
			termUnsub = true
		}
	}
	oracleEngine.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(spec.SubscriptionID(1), unsubscribe, nil)
	oracleEngine.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(spec.SubscriptionID(2), unsubscribe, nil)

	snap, err := newMarketFromSnapshot(t, context.Background(), ctrl, em, oracleEngine)
	require.NoError(t, err)
	require.NotEmpty(t, snap)

	// check the market is restored terminated and that we have unsubscribed one oracles
	assert.Equal(t, types.MarketStateTradingTerminated, snap.State())
	assert.True(t, termUnsub)
	closed := snap.OnTick(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()), time.Now())
	assert.False(t, closed)
}

func TestRestoreNilLastTradedPrice(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)
	defer tm.ctrl.Finish()

	em := tm.market.GetState()
	assert.Nil(t, em.LastTradedPrice)
	assert.Nil(t, em.CurrentMarkPrice)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	oracleEngine := mocks.NewMockOracleEngine(ctrl)

	unsubscribe := func(_ context.Context, id spec.SubscriptionID) {
	}
	oracleEngine.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(spec.SubscriptionID(1), unsubscribe, nil)
	oracleEngine.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(spec.SubscriptionID(2), unsubscribe, nil)

	snap, err := newMarketFromSnapshot(t, context.Background(), ctrl, em, oracleEngine)
	require.NoError(t, err)
	require.NotEmpty(t, snap)

	em2 := snap.GetState()
	assert.Nil(t, em2.LastTradedPrice)
	assert.Nil(t, em2.CurrentMarkPrice)
}

func getTerminatedMarket(t *testing.T) *testMarket {
	t.Helper()
	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
	}

	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)
	defer tm.ctrl.Finish()

	// terminate the market
	err := tm.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
		Signers: pubKeys,
		Data: map[string]string{
			"trading.terminated": "true",
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.MarketStateTradingTerminated, tm.market.State())

	return tm
}

func getSettledMarket(t *testing.T) *testMarket {
	t.Helper()

	tm := getTerminatedMarket(t)

	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
	}

	err := tm.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
		Signers: pubKeys,
		Data: map[string]string{
			"prices.ETH.value": "100",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, types.MarketStateSettled, tm.market.State())

	return tm
}

func getActiveMarket(t *testing.T) *testMarket {
	t.Helper()

	esm := newEquityShareMarket(t)
	matchingPrice := uint64(900000)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	esm.WithSubmittedOrder(t, "some-id-1", "party1", types.SideSell, matchingPrice+1).
		WithSubmittedOrder(t, "some-id-2", "party2", types.SideBuy, matchingPrice-1).
		WithSubmittedOrder(t, "some-id-3", "party1", types.SideSell, matchingPrice).
		WithSubmittedOrder(t, "some-id-4", "party2", types.SideBuy, matchingPrice).
		WithSubmittedLiquidityProvision(t, "party1", "lp-id-1", 2000000, "0.5").
		WithSubmittedLiquidityProvision(t, "party2", "lp-id-2", 1000000, "0.5")

	// end opening auction
	esm.tm.market.OnTick(ctx, esm.Now.Add(2*time.Second))
	return esm.tm
}

// newMarketFromSnapshot is a wrapper for NewMarketFromSnapshot with a lot of defaults handled.
func newMarketFromSnapshot(t *testing.T, ctx context.Context, ctrl *gomock.Controller, em *types.ExecMarket, oracleEngine products.OracleEngine) (*future.Market, error) {
	t.Helper()
	var (
		riskConfig       = risk.NewDefaultConfig()
		positionConfig   = positions.NewDefaultConfig()
		settlementConfig = settlement.NewDefaultConfig()
		matchingConfig   = matching.NewDefaultConfig()
		feeConfig        = fee.NewDefaultConfig()
		liquidityConfig  = liquidity.NewDefaultConfig()
	)
	log := logging.NewTestLogger()

	assets, err := em.Market.GetAssets()
	require.NoError(t, err)
	cfgAsset := NewAssetStub(assets[0], em.Market.DecimalPlaces)

	epochEngine := mocks.NewMockEpochEngine(ctrl)
	epochEngine.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).Times(1)
	teams := mocks.NewMockTeams(ctrl)
	bc := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)

	broker.EXPECT().Stage(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	timeService := mocks.NewMockTimeService(ctrl)
	timeService.EXPECT().GetTimeNow().AnyTimes()
	collateralEngine := collateral.New(log, collateral.NewDefaultConfig(), timeService, broker)

	marketActivityTracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, bc, broker, collateralEngine)
	epochEngine.NotifyOnEpoch(marketActivityTracker.OnEpochEvent, marketActivityTracker.OnEpochRestore)

	positionConfig.StreamPositionVerbose = true
	referralDiscountReward := fmock.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscount := fmock.NewMockVolumeDiscountService(ctrl)
	volumeRebate := fmock.NewMockVolumeRebateService(ctrl)
	referralDiscountReward.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscount.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	referralDiscountReward.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID(""), errors.New("not a referrer")).AnyTimes()
	banking := mocks.NewMockBanking(ctrl)
	parties := mocks.NewMockParties(ctrl)

	return future.NewMarketFromSnapshot(ctx, log, em, riskConfig, positionConfig, settlementConfig, matchingConfig,
		feeConfig, liquidityConfig, collateralEngine, oracleEngine, timeService, broker, stubs.NewStateVar(), cfgAsset, marketActivityTracker,
		peggedOrderCounterForTest, referralDiscountReward, volumeDiscount, volumeRebate, banking, parties)
}
