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

package execution_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"
	"time"

	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/core/assets"
	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/datasource"
	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/signedoracle"
	"code.vegaprotocol.io/vega/core/datasource/spec"

	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type engineFake struct {
	*execution.Engine
	ctrl       *gomock.Controller
	broker     *bmocks.MockBroker
	timeSvc    *mocks.MockTimeService
	collateral *mocks.MockCollateral
	oracle     *mocks.MockOracleEngine
	statevar   *mocks.MockStateVarEngine
	epoch      *mocks.MockEpochEngine
	asset      *mocks.MockAssets
}

func getMockedEngine(t *testing.T) *engineFake {
	t.Helper()
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	execConfig := execution.NewDefaultConfig()
	broker := bmocks.NewMockBroker(ctrl)
	// broker.EXPECT().Send(gomock.Any()).AnyTimes()
	// broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	timeService := mocks.NewMockTimeService(ctrl)
	// timeService.EXPECT().GetTimeNow().AnyTimes()

	collateralService := mocks.NewMockCollateral(ctrl)
	// collateralService.EXPECT().AssetExists(gomock.Any()).AnyTimes().Return(true)
	// collateralService.EXPECT().CreateMarketAccounts(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	oracleService := mocks.NewMockOracleEngine(ctrl)
	// oracleService.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	statevar := mocks.NewMockStateVarEngine(ctrl)
	// statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	// statevar.EXPECT().NewEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	epochEngine := mocks.NewMockEpochEngine(ctrl)
	epochEngine.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).Times(1)
	asset := mocks.NewMockAssets(ctrl)

	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	exec := execution.NewEngine(log, execConfig, timeService, collateralService, oracleService, broker, statevar, common.NewMarketActivityTracker(log, epochEngine, teams, balanceChecker), asset)
	return &engineFake{
		Engine:     exec,
		ctrl:       ctrl,
		broker:     broker,
		timeSvc:    timeService,
		collateral: collateralService,
		oracle:     oracleService,
		statevar:   statevar,
		epoch:      epochEngine,
		asset:      asset,
	}
}

func createEngine(t *testing.T) (*execution.Engine, *gomock.Controller) {
	t.Helper()
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	executionConfig := execution.NewDefaultConfig()
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	timeService := mocks.NewMockTimeService(ctrl)
	timeService.EXPECT().GetTimeNow().AnyTimes()

	collateralService := mocks.NewMockCollateral(ctrl)
	collateralService.EXPECT().AssetExists(gomock.Any()).AnyTimes().Return(true)
	collateralService.EXPECT().CreateMarketAccounts(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	collateralService.EXPECT().GetMarketLiquidityFeeAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&types.Account{Balance: num.UintZero()}, nil)
	collateralService.EXPECT().GetInsurancePoolBalance(gomock.Any(), gomock.Any()).AnyTimes().Return(num.UintZero(), true)
	collateralService.EXPECT().CreateSpotMarketAccounts(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	collateralService.EXPECT().GetAssetQuantum("ETH").AnyTimes().Return(num.DecimalFromInt64(1), nil)
	collateralService.EXPECT().GetOrCreatePartyBondAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&types.Account{Balance: num.UintZero()}, nil)
	collateralService.EXPECT().GetOrCreatePartyLiquidityFeeAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	collateralService.EXPECT().BondSpotUpdate(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&types.LedgerMovement{}, nil)
	oracleService := mocks.NewMockOracleEngine(ctrl)
	oracleService.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(spec.SubscriptionID(0), func(_ context.Context, _ spec.SubscriptionID) {}, nil)
	oracleService.EXPECT().Unsubscribe(gomock.Any(), gomock.Any()).AnyTimes()

	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statevar.EXPECT().NewEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statevar.EXPECT().UnregisterStateVariable(gomock.Any(), gomock.Any()).AnyTimes()

	epochEngine := mocks.NewMockEpochEngine(ctrl)
	epochEngine.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).Times(1)
	asset := mocks.NewMockAssets(ctrl)
	asset.EXPECT().Get(gomock.Any()).AnyTimes().DoAndReturn(func(a string) (*assets.Asset, error) {
		as := NewAssetStub(a, 0)
		return as, nil
	})
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)

	return execution.NewEngine(log, executionConfig, timeService, collateralService, oracleService, broker, statevar, common.NewMarketActivityTracker(log, epochEngine, teams, balanceChecker), asset), ctrl
}

func TestEmptyMarkets(t *testing.T) {
	engine, ctrl := createEngine(t)
	assert.NotNil(t, engine)
	defer ctrl.Finish()

	keys := engine.Keys()
	require.Equal(t, 1, len(keys))
	key := keys[0]

	// Check that the starting state is empty
	bytes, providers, err := engine.GetState(key)
	assert.NoError(t, err)
	assert.NotEmpty(t, bytes)
	assert.Empty(t, providers)
}

func getSpotMarketConfig() *types.Market {
	return &types.Market{
		ID: "SpotMarketID", // ID will be generated
		PriceMonitoringSettings: &types.PriceMonitoringSettings{
			Parameters: &types.PriceMonitoringParameters{
				Triggers: []*types.PriceMonitoringTrigger{
					{
						Horizon:          1000,
						HorizonDec:       num.DecimalFromFloat(1000.0),
						Probability:      num.DecimalFromFloat(0.3),
						AuctionExtension: 10000,
					},
				},
			},
		},
		LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
			TargetStakeParameters: &types.TargetStakeParameters{
				TimeWindow:    101,
				ScalingFactor: num.DecimalFromFloat(1.0),
			},
			TriggeringRatio:  num.DecimalZero(),
			AuctionExtension: 0,
		},
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				MakerFee:          num.DecimalFromFloat(0.1),
				InfrastructureFee: num.DecimalFromFloat(0.1),
				LiquidityFee:      num.DecimalFromFloat(0.1),
			},
		},
		LiquiditySLAParams: &types.LiquiditySLAParams{
			PriceRange:                      num.DecimalFromFloat(0.05),
			CommitmentMinTimeFraction:       num.DecimalFromFloat(0.5),
			SlaCompetitionFactor:            num.DecimalFromFloat(0.5),
			ProvidersFeeCalculationTimeStep: time.Second,
			PerformanceHysteresisEpochs:     1,
		},
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				ID:   "Crypto/BTC/ETH",
				Code: "SPOT:BTC/ETH",
				Name: "BTC/ETH SPOT",
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:spot/crypto",
						"product:spot",
					},
				},
				Product: &types.InstrumentSpot{
					Spot: &types.Spot{
						BaseAsset:  "BTC",
						QuoteAsset: "ETH",
						Name:       "BTC/ETH",
					},
				},
			},
			RiskModel: &types.TradableInstrumentLogNormalRiskModel{
				LogNormalRiskModel: &types.LogNormalRiskModel{
					RiskAversionParameter: num.DecimalFromFloat(0.01),
					Tau:                   num.DecimalFromFloat(1.0 / 365.25 / 24),
					Params: &types.LogNormalModelParams{
						Mu:    num.DecimalZero(),
						R:     num.DecimalFromFloat(0.016),
						Sigma: num.DecimalFromFloat(0.09),
					},
				},
			},
		},
		State: types.MarketStateActive,
	}
}

func getMarketConfig() *types.Market {
	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
	}

	return &types.Market{
		ID: "MarketID", // ID will be generated
		PriceMonitoringSettings: &types.PriceMonitoringSettings{
			Parameters: &types.PriceMonitoringParameters{
				Triggers: []*types.PriceMonitoringTrigger{
					{
						Horizon:          1000,
						HorizonDec:       num.DecimalFromFloat(1000.0),
						Probability:      num.DecimalFromFloat(0.3),
						AuctionExtension: 10000,
					},
				},
			},
		},
		LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
			TargetStakeParameters: &types.TargetStakeParameters{
				TimeWindow:    101,
				ScalingFactor: num.DecimalFromFloat(1.0),
			},
			TriggeringRatio:  num.DecimalFromFloat(0.9),
			AuctionExtension: 10000,
		},
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				MakerFee:          num.DecimalFromFloat(0.1),
				InfrastructureFee: num.DecimalFromFloat(0.1),
				LiquidityFee:      num.DecimalFromFloat(0.1),
			},
		},
		TradableInstrument: &types.TradableInstrument{
			MarginCalculator: &types.MarginCalculator{
				ScalingFactors: &types.ScalingFactors{
					SearchLevel:       num.DecimalFromFloat(1.2),
					InitialMargin:     num.DecimalFromFloat(1.3),
					CollateralRelease: num.DecimalFromFloat(1.4),
				},
			},
			Instrument: &types.Instrument{
				ID:   "Crypto/ETHUSD/Futures/Dec19",
				Code: "FX:ETHUSD/DEC19",
				Name: "December 2019 ETH vs USD future",
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				Product: &types.InstrumentFuture{
					Future: &types.Future{
						SettlementAsset: "Ethereum/Ether",
						DataSourceSpecForSettlementData: &datasource.Spec{
							ID: "1",
							Data: datasource.NewDefinition(
								datasource.ContentTypeOracle,
							).SetOracleConfig(
								&signedoracle.SpecConfiguration{
									Signers: pubKeys,
									Filters: []*dstypes.SpecFilter{
										{
											Key: &dstypes.SpecPropertyKey{
												Name: "prices.ETH.value",
												Type: datapb.PropertyKey_TYPE_INTEGER,
											},
											Conditions: []*dstypes.SpecCondition{},
										},
									},
								},
							),
						},
						DataSourceSpecForTradingTermination: &datasource.Spec{
							ID: "2",
							Data: datasource.NewDefinition(
								datasource.ContentTypeOracle,
							).SetOracleConfig(
								&signedoracle.SpecConfiguration{
									Signers: pubKeys,
									Filters: []*dstypes.SpecFilter{
										{
											Key: &dstypes.SpecPropertyKey{
												Name: "trading.terminated",
												Type: datapb.PropertyKey_TYPE_BOOLEAN,
											},
											Conditions: []*dstypes.SpecCondition{},
										},
									},
								},
							),
						},
						DataSourceSpecBinding: &datasource.SpecBindingForFuture{
							SettlementDataProperty:     "prices.ETH.value",
							TradingTerminationProperty: "trading.terminated",
						},
					},
				},
			},
			RiskModel: &types.TradableInstrumentLogNormalRiskModel{
				LogNormalRiskModel: &types.LogNormalRiskModel{
					RiskAversionParameter: num.DecimalFromFloat(0.01),
					Tau:                   num.DecimalFromFloat(1.0 / 365.25 / 24),
					Params: &types.LogNormalModelParams{
						Mu:    num.DecimalZero(),
						R:     num.DecimalFromFloat(0.016),
						Sigma: num.DecimalFromFloat(0.09),
					},
				},
			},
		},
		LiquiditySLAParams: &types.LiquiditySLAParams{
			PriceRange:                      num.DecimalOne(),
			CommitmentMinTimeFraction:       num.DecimalFromFloat(0.5),
			SlaCompetitionFactor:            num.DecimalOne(),
			ProvidersFeeCalculationTimeStep: time.Second * 1,
			PerformanceHysteresisEpochs:     1,
		},
		State: types.MarketStateActive,
	}
}

func TestEmptyExecEngineSnapshot(t *testing.T) {
	engine, ctrl := createEngine(t)
	assert.NotNil(t, engine)
	defer ctrl.Finish()

	keys := engine.Keys()
	require.Equal(t, 1, len(keys))
	key := keys[0]

	bytes, providers, err := engine.GetState(key)
	require.NoError(t, err)
	require.Empty(t, providers)
	require.NotNil(t, bytes)
}

func TestValidMarketSnapshot(t *testing.T) {
	ctx := context.Background()
	engine, ctrl := createEngine(t)
	defer ctrl.Finish()
	assert.NotNil(t, engine)

	marketConfig := getMarketConfig()
	err := engine.SubmitMarket(ctx, marketConfig, "", time.Now())
	assert.NoError(t, err)

	// submit successor
	marketConfig2 := getMarketConfig()
	marketConfig2.ParentMarketID = marketConfig.ID
	marketConfig2.InsurancePoolFraction = num.DecimalOne()
	err = engine.SubmitMarket(ctx, marketConfig2, "", time.Now())
	assert.NoError(t, err)

	keys := engine.Keys()
	require.Equal(t, 1, len(keys))
	key := keys[0]

	// Take the snapshot and hash
	b, providers, err := engine.GetState(key)
	assert.NoError(t, err)
	assert.NotEmpty(t, b)
	assert.Len(t, providers, 5)

	// Turn the bytes back into a payload and restore to a new engine
	engine2, ctrl := createEngine(t)

	defer ctrl.Finish()
	assert.NotNil(t, engine2)
	snap := &snapshot.Payload{}
	err = proto.Unmarshal(b, snap)
	assert.NoError(t, err)

	// check expected successors are in there
	tt, ok := snap.Data.(*snapshot.Payload_ExecutionMarkets)
	require.True(t, ok)
	require.Equal(t, 1, len(tt.ExecutionMarkets.Successors))
	require.Equal(t, marketConfig.ID, tt.ExecutionMarkets.Successors[0].ParentMarket)
	require.Equal(t, 1, len(tt.ExecutionMarkets.Successors[0].SuccessorMarkets))
	require.Equal(t, marketConfig2.ID, tt.ExecutionMarkets.Successors[0].SuccessorMarkets[0])

	loadStateProviders, err := engine2.LoadState(ctx, types.PayloadFromProto(snap))
	assert.Len(t, loadStateProviders, 10)
	assert.NoError(t, err)

	providerMap := map[string]map[string]types.StateProvider{}
	for _, p := range loadStateProviders {
		providerMap[p.Namespace().String()] = map[string]types.StateProvider{}
		for _, k := range p.Keys() {
			providerMap[p.Namespace().String()][k] = p
		}
	}

	// Check the hashes are the same
	state2, _, err := engine2.GetState(key)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(b, state2))

	snap = &snapshot.Payload{}
	err = proto.Unmarshal(state2, snap)
	assert.NoError(t, err)
	tt, ok = snap.Data.(*snapshot.Payload_ExecutionMarkets)
	require.True(t, ok)
	require.Equal(t, 1, len(tt.ExecutionMarkets.Successors))
	require.Equal(t, marketConfig.ID, tt.ExecutionMarkets.Successors[0].ParentMarket)
	require.Equal(t, 1, len(tt.ExecutionMarkets.Successors[0].SuccessorMarkets))
	require.Equal(t, marketConfig2.ID, tt.ExecutionMarkets.Successors[0].SuccessorMarkets[0])

	// now load the providers state
	for _, p := range providers {
		for _, k := range p.Keys() {
			b, _, err := p.GetState(k)
			require.NoError(t, err)

			snap := &snapshot.Payload{}
			err = proto.Unmarshal(b, snap)
			assert.NoError(t, err)

			toRestore := providerMap[p.Namespace().String()][k]
			_, err = toRestore.LoadState(ctx, types.PayloadFromProto(snap))
			require.NoError(t, err)
			b2, _, err := toRestore.GetState(k)
			require.NoError(t, err)
			assert.True(t, bytes.Equal(b, b2))
		}
	}

	m2, ok := engine2.GetMarket(marketConfig2.ID, false)
	require.True(t, ok)
	require.NotEmpty(t, marketConfig2.ParentMarketID, m2.ParentMarketID)
}

func TestValidSpotMarketSnapshot(t *testing.T) {
	ctx := context.Background()
	engine, ctrl := createEngine(t)
	defer ctrl.Finish()
	assert.NotNil(t, engine)

	marketConfig := getSpotMarketConfig()
	err := engine.SubmitSpotMarket(ctx, marketConfig, "", time.Now())
	assert.NoError(t, err)

	err = engine.SubmitLiquidityProvision(ctx, &types.LiquidityProvisionSubmission{
		MarketID:         marketConfig.ID,
		CommitmentAmount: num.NewUint(1000),
		Fee:              num.DecimalFromFloat(0.5),
	}, "zohar", crypto.RandomHash())
	require.NoError(t, err)

	keys := engine.Keys()
	require.Equal(t, 1, len(keys))
	key := keys[0]

	// Take the snapshot and hash
	b, providers, err := engine.GetState(key)
	assert.NoError(t, err)
	assert.NotEmpty(t, b)
	assert.Len(t, providers, 4)

	// Turn the bytes back into a payload and restore to a new engine
	engine2, ctrl := createEngine(t)

	defer ctrl.Finish()
	assert.NotNil(t, engine2)
	snap := &snapshot.Payload{}
	err = proto.Unmarshal(b, snap)
	assert.NoError(t, err)

	loadStateProviders, err := engine2.LoadState(ctx, types.PayloadFromProto(snap))
	assert.Len(t, loadStateProviders, 4)
	assert.NoError(t, err)

	providerMap := map[string]map[string]types.StateProvider{}
	for _, p := range loadStateProviders {
		providerMap[p.Namespace().String()] = map[string]types.StateProvider{}
		for _, k := range p.Keys() {
			providerMap[p.Namespace().String()][k] = p
		}
	}

	// Check the hashes are the same
	state2, _, err := engine2.GetState(key)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(b, state2))

	snap = &snapshot.Payload{}
	err = proto.Unmarshal(state2, snap)
	assert.NoError(t, err)
	_, ok := snap.Data.(*snapshot.Payload_ExecutionMarkets)
	require.True(t, ok)

	// now load the providers state
	for _, p := range providers {
		for _, k := range p.Keys() {
			b, _, err := p.GetState(k)
			require.NoError(t, err)

			snap := &snapshot.Payload{}
			err = proto.Unmarshal(b, snap)
			assert.NoError(t, err)

			toRestore := providerMap[p.Namespace().String()][k]
			_, err = toRestore.LoadState(ctx, types.PayloadFromProto(snap))
			require.NoError(t, err)
			b2, _, err := toRestore.GetState(k)
			require.NoError(t, err)
			assert.True(t, bytes.Equal(b, b2))
		}
	}
}

func TestValidSettledMarketSnapshot(t *testing.T) {
	ctx := vgcontext.WithTraceID(context.Background(), hex.EncodeToString([]byte("0deadbeef")))
	engine := getMockedEngine(t)
	engine.collateral.EXPECT().AssetExists(gomock.Any()).AnyTimes().Return(true)
	engine.collateral.EXPECT().CreateMarketAccounts(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	engine.collateral.EXPECT().GetMarketLiquidityFeeAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&types.Account{Balance: num.UintZero()}, nil)
	engine.collateral.EXPECT().GetLiquidityFeesBonusDistributionAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&types.Account{Balance: num.UintZero()}, nil)
	engine.collateral.EXPECT().GetInsurancePoolBalance(gomock.Any(), gomock.Any()).AnyTimes().Return(num.UintZero(), true)
	engine.collateral.EXPECT().FinalSettlement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	engine.collateral.EXPECT().ClearMarket(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).AnyTimes().Return(nil, nil)
	engine.collateral.EXPECT().TransferFees(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	engine.timeSvc.EXPECT().GetTimeNow().AnyTimes()
	engine.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	engine.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	// engine.oracle.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	engine.statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	engine.statevar.EXPECT().UnregisterStateVariable(gomock.Any(), gomock.Any()).AnyTimes()
	engine.statevar.EXPECT().NewEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	engine.epoch.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).AnyTimes()
	engine.asset.EXPECT().Get(gomock.Any()).AnyTimes().DoAndReturn(func(a string) (*assets.Asset, error) {
		as := NewAssetStub(a, 0)
		return as, nil
	})
	// create a market
	marketConfig := getMarketConfig()
	// ensure CP state doesn't get invalidated the moment the market is settled
	engine.OnSuccessorMarketTimeWindowUpdate(ctx, time.Hour)
	// now let's set up the settlement and trading terminated callbacks
	var ttCB, sCB spec.OnMatchedData
	ttData := dstypes.Data{
		Signers: marketConfig.TradableInstrument.Instrument.GetFuture().DataSourceSpecForTradingTermination.Data.GetSigners(),
		Data: map[string]string{
			"trading.terminated": "true",
		},
	}
	sData := dstypes.Data{
		Signers: marketConfig.TradableInstrument.Instrument.GetFuture().DataSourceSpecForSettlementData.Data.GetSigners(),
		Data: map[string]string{
			"prices.ETH.value": "100000",
		},
	}
	engine.oracle.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ context.Context, s spec.Spec, cb spec.OnMatchedData) (spec.SubscriptionID, spec.Unsubscriber, error) {
		if ok, _ := s.MatchData(ttData); ok {
			ttCB = cb
		} else if ok, _ := s.MatchData(sData); ok {
			sCB = cb
		}
		return spec.SubscriptionID(0), func(_ context.Context, _ spec.SubscriptionID) {}, nil
	})
	defer engine.ctrl.Finish()
	assert.NotNil(t, engine)

	err := engine.SubmitMarket(ctx, marketConfig, "", time.Now())
	assert.NoError(t, err)
	// now let's settle the market by:
	// 1. Ensuring the market is in active state
	marketConfig.State = types.MarketStateActive
	engine.OnTick(ctx, time.Now())
	// 2. Using the oracle to set the market to trading terminated, then settling the market
	ttCB(ctx, ttData)
	sCB(ctx, sData)
	require.Equal(t, marketConfig.State, types.MarketStateSettled)
	// ensure the market data returns no trading
	md, err := engine.GetMarketData(marketConfig.ID)
	require.NoError(t, err)
	require.Equal(t, types.MarketTradingModeNoTrading, md.MarketTradingMode)
	engine.OnTick(ctx, time.Now())

	keys := engine.Keys()
	require.Equal(t, 1, len(keys))
	key := keys[0]

	// Take the snapshot and hash
	b, providers, err := engine.GetState(key)
	assert.NoError(t, err)
	assert.NotEmpty(t, b)
	// this is now empty, the market is settled, no state providers required
	assert.Len(t, providers, 0)

	// Turn the bytes back into a payload and restore to a new engine
	engine2, ctrl := createEngine(t)
	engine2.OnSuccessorMarketTimeWindowUpdate(ctx, time.Hour)

	defer ctrl.Finish()
	assert.NotNil(t, engine2)
	snap := &snapshot.Payload{}
	err = proto.Unmarshal(b, snap)
	assert.NoError(t, err)
	loadStateProviders, err := engine2.LoadState(ctx, types.PayloadFromProto(snap))
	assert.Len(t, loadStateProviders, 0)
	assert.NoError(t, err)

	providerMap := map[string]map[string]types.StateProvider{}
	for _, p := range loadStateProviders {
		providerMap[p.Namespace().String()] = map[string]types.StateProvider{}
		for _, k := range p.Keys() {
			providerMap[p.Namespace().String()][k] = p
		}
	}

	// Check the hashes are the same
	state2, _, err := engine2.GetState(key)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(b, state2))

	// now load the providers state
	for _, p := range providers {
		for _, k := range p.Keys() {
			b, _, err := p.GetState(k)
			require.NoError(t, err)

			snap := &snapshot.Payload{}
			err = proto.Unmarshal(b, snap)
			assert.NoError(t, err)

			toRestore := providerMap[p.Namespace().String()][k]
			_, err = toRestore.LoadState(ctx, types.PayloadFromProto(snap))
			require.NoError(t, err)
			b2, _, err := toRestore.GetState(k)
			require.NoError(t, err)
			assert.True(t, bytes.Equal(b, b2))
		}
	}
	// ensure the market is restored as settled
	_, ok := engine2.GetMarket(marketConfig.ID, true)
	require.True(t, ok)
}

func TestSuccessorMapSnapshot(t *testing.T) {
	ctx := vgcontext.WithTraceID(context.Background(), hex.EncodeToString([]byte("0deadbeef")))
	engine := getMockedEngine(t)
	engine.collateral.EXPECT().AssetExists(gomock.Any()).AnyTimes().Return(true)
	engine.collateral.EXPECT().CreateMarketAccounts(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	engine.collateral.EXPECT().GetMarketLiquidityFeeAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&types.Account{Balance: num.UintZero()}, nil)
	engine.collateral.EXPECT().GetInsurancePoolBalance(gomock.Any(), gomock.Any()).AnyTimes().Return(num.UintZero(), true)
	engine.collateral.EXPECT().FinalSettlement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	engine.collateral.EXPECT().ClearMarket(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	engine.timeSvc.EXPECT().GetTimeNow().AnyTimes()
	engine.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	engine.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	// engine.oracle.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	engine.statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	engine.statevar.EXPECT().UnregisterStateVariable(gomock.Any(), gomock.Any()).AnyTimes()
	engine.statevar.EXPECT().NewEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	engine.epoch.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).AnyTimes()
	engine.asset.EXPECT().Get(gomock.Any()).AnyTimes().DoAndReturn(func(a string) (*assets.Asset, error) {
		as := NewAssetStub(a, 0)
		return as, nil
	})
	// create a market
	marketConfig := getMarketConfig()
	// ensure CP state doesn't get invalidated the moment the market is settled
	engine.OnSuccessorMarketTimeWindowUpdate(ctx, time.Hour)
	// now let's set up the settlement and trading terminated callbacks
	engine.oracle.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ context.Context, s spec.Spec, cb spec.OnMatchedData) (spec.SubscriptionID, spec.Unsubscriber, error) {
		return spec.SubscriptionID(0), func(_ context.Context, _ spec.SubscriptionID) {}, nil
	})
	defer engine.ctrl.Finish()
	assert.NotNil(t, engine)

	err := engine.SubmitMarket(ctx, marketConfig, "", time.Now())
	assert.NoError(t, err)
	// now let's settle the market by:
	// 1. Create successor
	successor := marketConfig.DeepClone()
	successor.ID = "successor-id"
	successor.ParentMarketID = marketConfig.ID
	successor.InsurancePoolFraction = num.DecimalFromFloat(1)
	successor.State = types.MarketStateProposed
	successor.TradingMode = types.MarketTradingModeNoTrading
	// submit the successor market
	engine.SubmitMarket(ctx, successor, "", time.Now())
	engine.OnTick(ctx, time.Now())
	// 2. cancel the parent market (before leaving opening auction)
	engine.RejectMarket(ctx, marketConfig.ID)
	engine.OnTick(ctx, time.Now())

	// 3. Check the successor map in the snapshot

	keys := engine.Keys()
	require.Equal(t, 1, len(keys))
	key := keys[0]

	// Take the snapshot and hash
	b, _, err := engine.GetState(key)
	assert.NoError(t, err)
	assert.NotEmpty(t, b)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(b, snap)
	assert.NoError(t, err)

	// Check the hashes are the same
	execMkts := snap.GetExecutionMarkets()
	require.NotNil(t, execMkts)
	require.Empty(t, execMkts.Successors)
}
