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
	liqmocks "code.vegaprotocol.io/vega/core/liquidity/v2/mocks"
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

	snap, err := newMarketFromSnapshot(t, ctrl, em, oracleEngine)
	require.NoError(t, err)
	require.NotEmpty(t, snap)

	// check the market is restored settled and that we have unsubscribed the two oracles
	assert.Equal(t, types.MarketStateSettled, snap.State())
	assert.Equal(t, uint64(2), unsubs)
	closed := tm.market.OnTick(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()), time.Now())
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

	snap, err := newMarketFromSnapshot(t, ctrl, em, oracleEngine)
	require.NoError(t, err)
	require.NotEmpty(t, snap)

	// check the market is restored terminated and that we have unsubscribed one oracles
	assert.Equal(t, types.MarketStateTradingTerminated, snap.State())
	assert.True(t, termUnsub)
	closed := tm.market.OnTick(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()), time.Now())
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

	snap, err := newMarketFromSnapshot(t, ctrl, em, oracleEngine)
	require.NoError(t, err)
	require.NotEmpty(t, snap)

	em2 := tm.market.GetState()
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

// newMarketFromSnapshot is a wrapper for NewMarketFromSnapshot with a lot of defaults handled.
func newMarketFromSnapshot(t *testing.T, ctrl *gomock.Controller, em *types.ExecMarket, oracleEngine products.OracleEngine) (*future.Market, error) {
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
	marketActivityTracker := common.NewMarketActivityTracker(logging.NewTestLogger(), epochEngine, teams, bc)

	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	timeService := mocks.NewMockTimeService(ctrl)
	timeService.EXPECT().GetTimeNow().AnyTimes()
	collateralEngine := collateral.New(log, collateral.NewDefaultConfig(), timeService, broker)

	positionConfig.StreamPositionVerbose = true
	referralDiscountReward := fmock.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscount := fmock.NewMockVolumeDiscountService(ctrl)
	referralDiscountReward.EXPECT().ReferralDiscountFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	volumeDiscount.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	referralDiscountReward.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID(""), errors.New("not a referrer")).AnyTimes()

	epochTime := liqmocks.NewMockEpochTime(ctrl)

	return future.NewMarketFromSnapshot(context.Background(), log, em, riskConfig, positionConfig, settlementConfig, matchingConfig,
		feeConfig, liquidityConfig, collateralEngine, oracleEngine, timeService, broker, stubs.NewStateVar(), cfgAsset, marketActivityTracker,
		peggedOrderCounterForTest, referralDiscountReward, volumeDiscount, epochTime)
}
