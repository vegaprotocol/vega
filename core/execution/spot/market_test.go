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

package spot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/execution/spot"
	"code.vegaprotocol.io/vega/core/fee"
	fmocks "code.vegaprotocol.io/vega/core/fee/mocks"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/monitor"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type testMarket struct {
	market           *spot.Market
	log              *logging.Logger
	ctrl             *gomock.Controller
	collateralEngine *collateral.Engine
	broker           *bmocks.MockBroker
	timeService      *mocks.MockTimeService
	banking          *mocks.MockBanking
	now              time.Time
	baseAsset        string
	quoteAsset       string
	mas              *monitor.AuctionState
	eventCount       uint64
	orderEventCount  uint64
	events           []events.Event
	orderEvents      []events.Event
	mktCfg           *types.Market
	stateVar         *stubs.StateVarStub
}

var (
	MAXMOVEUP   = num.DecimalFromFloat(1000)
	MINMOVEDOWN = num.DecimalFromFloat(500)
)

func peggedOrderCounterForTest(int64) {}

var defaultCollateralAssets = []types.Asset{
	{
		ID: "ETH",
		Details: &types.AssetDetails{
			Symbol:  "ETH",
			Quantum: num.DecimalOne(),
		},
	},
	{
		ID: "BTC",
		Details: &types.AssetDetails{
			Symbol:  "BTC",
			Quantum: num.DecimalOne(),
		},
	},
	{
		ID: "VOTE",
		Details: &types.AssetDetails{
			Name:     "VOTE",
			Symbol:   "VOTE",
			Decimals: 5,
			Quantum:  num.DecimalOne(),
			Source: &types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{},
			},
		},
	},
}

var defaultPriceMonitorSettings = &types.PriceMonitoringSettings{
	Parameters: &types.PriceMonitoringParameters{
		Triggers: []*types.PriceMonitoringTrigger{
			{
				Horizon:          600,
				HorizonDec:       num.MustDecimalFromString("600"),
				Probability:      num.DecimalFromFloat(0.99),
				AuctionExtension: 120,
			},
		},
	},
}

func getMarketWithDP(base, quote string, pMonitorSettings *types.PriceMonitoringSettings, openingAuctionDuration *types.AuctionDuration, quoteDecimalPlaces uint64, positionDP int64) types.Market {
	mkt := types.Market{
		ID:                    crypto.RandomHash(),
		DecimalPlaces:         quoteDecimalPlaces,
		PositionDecimalPlaces: positionDP,
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				InfrastructureFee: num.DecimalFromFloat(0.001),
				MakerFee:          num.DecimalFromFloat(0.004),
			},
			LiquidityFeeSettings: &types.LiquidityFeeSettings{
				Method: vega.LiquidityFeeSettings_METHOD_MARGINAL_COST,
			},
		},
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				ID:   "Crypto/Base-Quote/Spot",
				Code: "CRYPTO:Base-Quote",
				Name: "Base-Quote spot",
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:spot/crypto",
						"product:spot",
					},
				},
				Product: &types.InstrumentSpot{
					Spot: &types.Spot{
						BaseAsset:  base,
						QuoteAsset: quote,
						Name:       base + "/" + quote,
					},
				},
			},
			RiskModel: &types.TradableInstrumentSimpleRiskModel{
				SimpleRiskModel: &types.SimpleRiskModel{
					Params: &types.SimpleModelParams{
						FactorLong:           num.DecimalFromFloat(0.15),
						FactorShort:          num.DecimalFromFloat(0.25),
						MaxMoveUp:            MAXMOVEUP,
						MinMoveDown:          MINMOVEDOWN,
						ProbabilityOfTrading: num.DecimalFromFloat(0.1),
					},
				},
			},
		},
		OpeningAuction:          openingAuctionDuration,
		PriceMonitoringSettings: pMonitorSettings,
		LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
			TargetStakeParameters: &types.TargetStakeParameters{
				TimeWindow:    3600, // seconds = 1h
				ScalingFactor: num.DecimalFromFloat(10),
			},
		},
		LiquiditySLAParams: &types.LiquiditySLAParams{
			PriceRange:                  num.DecimalFromFloat(0.05),
			CommitmentMinTimeFraction:   num.DecimalFromFloat(0.5),
			PerformanceHysteresisEpochs: 1,
			SlaCompetitionFactor:        num.DecimalFromFloat(0.5),
		},
		TickSize: num.UintOne(),
	}

	return mkt
}

func newTestMarket(
	t *testing.T,
	pMonitorSettings *types.PriceMonitoringSettings,
	openingAuctionDuration *types.AuctionDuration,
	now time.Time,
) *testMarket {
	t.Helper()
	return newTestMarketWithAllowedSellers(t, pMonitorSettings, openingAuctionDuration, now, nil)
}

func newTestMarketWithAllowedSellers(
	t *testing.T,
	pMonitorSettings *types.PriceMonitoringSettings,
	openingAuctionDuration *types.AuctionDuration,
	now time.Time,
	allowedSellers []string,
) *testMarket {
	t.Helper()
	base := "BTC"
	quote := "ETH"
	quoteDP := uint64(0)
	baseDP := uint64(0)
	positionDP := int64(0)
	log := logging.NewDevLogger()
	ctrl := gomock.NewController(t)
	ts := mocks.NewMockTimeService(ctrl)
	ts.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return now
		}).AnyTimes()
	broker := bmocks.NewMockBroker(ctrl)
	collateral := collateral.New(log, collateral.NewDefaultConfig(), ts, broker)
	ctx := context.Background()

	statevarEngine := stubs.NewStateVar()
	mkt := getMarketWithDP(base, quote, pMonitorSettings, openingAuctionDuration, quoteDP, positionDP)
	mkt.AllowedSellers = allowedSellers

	as := monitor.NewAuctionState(&mkt, now)
	epoch := mocks.NewMockEpochEngine(ctrl)
	epoch.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).AnyTimes()

	teams := mocks.NewMockTeams(ctrl)
	bc := mocks.NewMockAccountBalanceChecker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	mat := common.NewMarketActivityTracker(log, teams, bc, broker, collateral)
	epoch.NotifyOnEpoch(mat.OnEpochEvent, mat.OnEpochRestore)

	baseAsset := NewAssetStub(base, baseDP)
	quoteAsset := NewAssetStub(quote, quoteDP)

	referralDiscountReward := fmocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscount := fmocks.NewMockVolumeDiscountService(ctrl)
	volumeRebate := fmocks.NewMockVolumeRebateService(ctrl)
	referralDiscountReward.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID(""), errors.New("no referrer")).AnyTimes()
	referralDiscountReward.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	referralDiscountReward.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscount.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeRebate.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	banking := mocks.NewMockBanking(ctrl)
	vaultService := mocks.NewMockVaultService(ctrl)
	vaultService.EXPECT().GetVaultOwner(gomock.Any()).Return(nil).AnyTimes()

	market, _ := spot.NewMarket(log, matching.NewDefaultConfig(), fee.NewDefaultConfig(), liquidity.NewDefaultConfig(), collateral, &mkt, ts, broker, as, statevarEngine, mat, baseAsset, quoteAsset, peggedOrderCounterForTest, referralDiscountReward, volumeDiscount, volumeRebate, banking, vaultService)

	tm := &testMarket{
		market:           market,
		log:              log,
		ctrl:             ctrl,
		broker:           broker,
		timeService:      ts,
		banking:          banking,
		baseAsset:        base,
		quoteAsset:       quote,
		mas:              as,
		now:              now,
		collateralEngine: collateral,
		mktCfg:           &mkt,
		stateVar:         statevarEngine,
	}

	// eventFn records and count events and orderEvents
	eventFn := func(evt events.Event) {
		if evt.Type() == events.OrderEvent {
			tm.orderEventCount++
			tm.orderEvents = append(tm.orderEvents, evt)
		}
		tm.eventCount++
		tm.events = append(tm.events, evt)
	}
	// eventsFn is the same as eventFn above but handles []event
	eventsFn := func(evts []events.Event) {
		for _, evt := range evts {
			eventFn(evt)
		}
	}
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(eventFn)
	tm.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(eventsFn)

	assets := defaultCollateralAssets
	for _, a := range assets {
		err := collateral.EnableAsset(ctx, a)
		require.NoError(t, err)
	}

	tm.collateralEngine.CreateSpotMarketAccounts(ctx, tm.market.GetID(), quote)

	return tm
}

func addAccountWithAmount(market *testMarket, party string, amnt uint64, asset string) *types.LedgerMovement {
	r, _ := market.collateralEngine.Deposit(context.Background(), party, asset, num.NewUint(amnt))
	return r
}

func getGTCLimitOrder(tm *testMarket,
	now time.Time,
	id string,
	side types.Side,
	partyID string,
	size uint64,
	price uint64,
) *types.Order {
	order := &types.Order{
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		Status:      types.OrderStatusActive,
		ID:          id,
		Side:        side,
		Party:       partyID,
		MarketID:    tm.market.GetID(),
		Size:        size,
		Price:       num.NewUint(price),
		Remaining:   size,
		CreatedAt:   now.UnixNano(),
		Reference:   "marketorder",
	}
	return order
}

//nolint:unparam
func getStopOrderSubmission(tm *testMarket,
	now time.Time,
	id string,
	side1 types.Side,
	side2 types.Side,
	partyID string,
	size uint64,
	price uint64,
) *types.StopOrdersSubmission {
	return &types.StopOrdersSubmission{
		RisesAbove: &types.StopOrderSetup{
			OrderSubmission: &types.OrderSubmission{
				Type:        types.OrderTypeLimit,
				TimeInForce: types.OrderTimeInForceGTC,
				Side:        side1,
				MarketID:    tm.market.GetID(),
				Size:        size,
				Price:       num.NewUint(price),
				Reference:   "marketorder",
			},
			Expiry: &types.StopOrderExpiry{
				ExpiryStrategy: ptr.From(types.StopOrderExpiryStrategyCancels),
			},
			Trigger:             types.NewTrailingStopOrderTrigger(types.StopOrderTriggerDirectionRisesAbove, num.DecimalFromFloat(0.9)),
			SizeOverrideSetting: types.StopOrderSizeOverrideSettingNone,
			SizeOverrideValue:   nil,
		},
		FallsBelow: &types.StopOrderSetup{
			OrderSubmission: &types.OrderSubmission{
				Type:        types.OrderTypeLimit,
				TimeInForce: types.OrderTimeInForceGTC,
				Side:        side2,
				MarketID:    tm.market.GetID(),
				Size:        size,
				Price:       num.NewUint(price),
				Reference:   "marketorder",
			},
			Expiry: &types.StopOrderExpiry{
				ExpiryStrategy: ptr.From(types.StopOrderExpiryStrategyCancels),
			},
			Trigger:             types.NewTrailingStopOrderTrigger(types.StopOrderTriggerDirectionRisesAbove, num.DecimalFromFloat(0.9)),
			SizeOverrideSetting: types.StopOrderSizeOverrideSettingNone,
			SizeOverrideValue:   nil,
		},
	}
}
