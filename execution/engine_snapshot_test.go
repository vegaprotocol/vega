package execution_test

import (
	"bytes"
	"context"
	"testing"

	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/assets"
	bmock "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/execution/mocks"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createEngine(t *testing.T) (*execution.Engine, *gomock.Controller) {
	t.Helper()
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	executionConfig := execution.NewDefaultConfig()
	broker := bmock.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	timeService := mocks.NewMockTimeService(ctrl)
	timeService.EXPECT().NotifyOnTick(gomock.Any()).Times(1)
	timeService.EXPECT().GetTimeNow().AnyTimes()

	collateralService := mocks.NewMockCollateral(ctrl)
	collateralService.EXPECT().AssetExists(gomock.Any()).AnyTimes().Return(true)
	collateralService.EXPECT().CreateMarketAccounts(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	oracleService := mocks.NewMockOracleEngine(ctrl)
	oracleService.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statevar.EXPECT().NewEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	epochEngine := mocks.NewMockEpochEngine(ctrl)
	epochEngine.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).Times(1)
	asset := mocks.NewMockAssets(ctrl)
	asset.EXPECT().Get(gomock.Any()).AnyTimes().DoAndReturn(func(a string) (*assets.Asset, error) {
		as := NewAssetStub(a, 0)
		return as, nil
	})
	return execution.NewEngine(log, executionConfig, timeService, collateralService, oracleService, broker, statevar, execution.NewMarketActivityTracker(log, epochEngine), asset), ctrl
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

func getMarketConfig() *types.Market {
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
						OracleSpecForSettlementPrice: &types.OracleSpec{
							ID:      "1",
							PubKeys: []string{"0xDEADBEEF"},
							Filters: []*types.OracleSpecFilter{
								{
									Key: &types.OracleSpecPropertyKey{
										Name: "prices.ETH.value",
										Type: oraclespb.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*types.OracleSpecCondition{},
								},
							},
						},
						OracleSpecForTradingTermination: &types.OracleSpec{
							ID:      "2",
							PubKeys: []string{"0xDEADBEEF"},
							Filters: []*types.OracleSpecFilter{
								{
									Key: &types.OracleSpecPropertyKey{
										Name: "trading.terminated",
										Type: oraclespb.PropertyKey_TYPE_BOOLEAN,
									},
									Conditions: []*types.OracleSpecCondition{},
								},
							},
						},
						OracleSpecBinding: &types.OracleSpecBindingForFuture{
							SettlementPriceProperty:    "prices.ETH.value",
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
	err := engine.SubmitMarket(ctx, marketConfig, "")
	assert.NoError(t, err)

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
