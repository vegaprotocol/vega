package execution_test

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/execution/mocks"
	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/monitor"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/positions"
	types "code.vegaprotocol.io/vega/proto"
	oraclesv1 "code.vegaprotocol.io/vega/proto/oracles/v1"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const MAXMOVEUP = 1000
const MINMOVEDOWN = -500

var defaultCollateralAssets = []types.Asset{
	{
		Id:     "ETH",
		Symbol: "ETH",
	},
	{
		Id:          "VOTE",
		Name:        "VOTE",
		Symbol:      "VOTE",
		Decimals:    5,
		TotalSupply: "1000",
		Source: &types.AssetSource{
			Source: &types.AssetSource_BuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					Name:        "VOTE",
					Symbol:      "VOTE",
					Decimals:    5,
					TotalSupply: "1000",
				},
			},
		},
	},
}

var defaultPriceMonitorSettings = &types.PriceMonitoringSettings{
	Parameters: &types.PriceMonitoringParameters{
		Triggers: []*types.PriceMonitoringTrigger{},
	},
	UpdateFrequency: 0,
}

type testMarket struct {
	t *testing.T

	market           *execution.Market
	log              *logging.Logger
	ctrl             *gomock.Controller
	collateralEngine *collateral.Engine
	broker           *mocks.MockBroker
	now              time.Time
	asset            string
	mas              *monitor.AuctionState
	eventCount       uint64
	orderEventCount  uint64
	events           []events.Event
	orderEvents      []events.Event
	mktCfg           *types.Market

	// Options
	Assets []types.Asset
}

func newTestMarket(t *testing.T, now time.Time) *testMarket {
	ctrl := gomock.NewController(t)
	tm := &testMarket{
		t:    t,
		ctrl: ctrl,
		log:  logging.NewTestLogger(),
		now:  now,
	}

	// Setup Mocking Expectations
	tm.broker = mocks.NewMockBroker(ctrl)

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

	return tm
}

func (tm *testMarket) Run(ctx context.Context, mktCfg types.Market) *testMarket {
	collateralEngine, err := collateral.New(tm.log, collateral.NewDefaultConfig(), tm.broker, tm.now)
	require.NoError(tm.t, err)
	var assets = tm.Assets
	if len(assets) == 0 {
		assets = defaultCollateralAssets
	}
	for _, asset := range assets {
		collateralEngine.EnableAsset(ctx, asset)
	}

	var (
		riskConfig       = risk.NewDefaultConfig()
		positionConfig   = positions.NewDefaultConfig()
		settlementConfig = settlement.NewDefaultConfig()
		matchingConfig   = matching.NewDefaultConfig()
		feeConfig        = fee.NewDefaultConfig()
	)

	oracleEngine := oracles.NewEngine(tm.log, oracles.NewDefaultConfig(), tm.now, tm.broker)

	mas := monitor.NewAuctionState(&mktCfg, tm.now)
	monitor.NewAuctionState(&mktCfg, tm.now)
	mktEngine, err := execution.NewMarket(ctx,
		tm.log, riskConfig, positionConfig, settlementConfig, matchingConfig,
		feeConfig, collateralEngine, oracleEngine, &mktCfg, tm.now, tm.broker, execution.NewIDGen(), mas,
	)
	require.NoError(tm.t, err)

	asset, err := mktCfg.GetAsset()
	require.NoError(tm.t, err)

	_, _, err = collateralEngine.CreateMarketAccounts(ctx, mktEngine.GetID(), asset, 0)
	require.NoError(tm.t, err)

	tm.market = mktEngine
	tm.collateralEngine = collateralEngine
	tm.asset = asset
	tm.mas = mas
	tm.mktCfg = &mktCfg

	// Reset event counters
	tm.eventCount = 0
	tm.orderEventCount = 0

	return tm
}

func (tm *testMarket) StartOpeningAuction() *testMarket {
	tm.market.StartOpeningAuction(context.Background())
	return tm
}

func (tm *testMarket) WithAccountAndAmount(id string, amount uint64) *testMarket {
	addAccountWithAmount(tm, id, amount)
	return tm
}

func (tm *testMarket) PartyGeneralAccount(t *testing.T, party string) *types.Account {
	acc, err := tm.collateralEngine.GetPartyGeneralAccount(party, tm.asset)
	require.NoError(t, err)
	require.NotNil(t, acc)
	return acc
}

func (tm *testMarket) PartyMarginAccount(t *testing.T, party string) *types.Account {
	acc, err := tm.collateralEngine.GetPartyMarginAccount(tm.market.GetID(), party, tm.asset)
	require.NoError(t, err)
	require.NotNil(t, acc)
	return acc
}

func getTestMarket(t *testing.T, now time.Time, closingAt time.Time, pMonitorSettings *types.PriceMonitoringSettings, openingAuctionDuration *types.AuctionDuration) *testMarket {
	return getTestMarket2(t, now, closingAt, pMonitorSettings, openingAuctionDuration, true)
}

func getTestMarket2(
	t *testing.T,
	now time.Time,
	closingAt time.Time,
	pMonitorSettings *types.PriceMonitoringSettings,
	openingAuctionDuration *types.AuctionDuration,
	startOpeningAuction bool,
) *testMarket {
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	riskConfig := risk.NewDefaultConfig()
	positionConfig := positions.NewDefaultConfig()
	settlementConfig := settlement.NewDefaultConfig()
	matchingConfig := matching.NewDefaultConfig()
	feeConfig := fee.NewDefaultConfig()
	broker := mocks.NewMockBroker(ctrl)

	tm := &testMarket{
		log:    log,
		ctrl:   ctrl,
		broker: broker,
		now:    now,
	}

	handleEvent := func(evt events.Event) {
		te := evt.Type()
		if te == events.OrderEvent {
			tm.orderEventCount++
			tm.orderEvents = append(tm.orderEvents, evt)
		}
		tm.eventCount++
		tm.events = append(tm.events, evt)
	}

	// catch all expected calls
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(
		func(evts []events.Event) {
			for _, evt := range evts {
				handleEvent(evt)
			}
		},
	)

	broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(handleEvent)

	collateralEngine, err := collateral.New(log, collateral.NewDefaultConfig(), broker, now)
	assert.Nil(t, err)
	collateralEngine.EnableAsset(context.Background(), types.Asset{
		Symbol: "ETH",
		Id:     "ETH",
	})

	oracleEngine := oracles.NewEngine(log, oracles.NewDefaultConfig(), now, broker)

	// add the token asset
	tokAsset := types.Asset{
		Id:          "VOTE",
		Name:        "VOTE",
		Symbol:      "VOTE",
		Decimals:    5,
		TotalSupply: "1000",
		Source: &types.AssetSource{
			Source: &types.AssetSource_BuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					Name:        "VOTE",
					Symbol:      "VOTE",
					Decimals:    5,
					TotalSupply: "1000",
				},
			},
		},
	}

	collateralEngine.EnableAsset(context.Background(), tokAsset)

	if pMonitorSettings == nil {
		pMonitorSettings = &types.PriceMonitoringSettings{
			Parameters: &types.PriceMonitoringParameters{
				Triggers: []*types.PriceMonitoringTrigger{},
			},
			UpdateFrequency: 0,
		}
	}

	mkt := getMarket(closingAt, pMonitorSettings, openingAuctionDuration)
	mktCfg := &mkt

	mas := monitor.NewAuctionState(mktCfg, now)
	mktEngine, err := execution.NewMarket(context.Background(),
		log, riskConfig, positionConfig, settlementConfig, matchingConfig,
		feeConfig, collateralEngine, oracleEngine, mktCfg, now, broker, execution.NewIDGen(), mas)
	assert.NoError(t, err)

	if startOpeningAuction {
		mktEngine.StartOpeningAuction(context.Background())
	}

	asset, err := mkt.GetAsset()
	assert.NoError(t, err)

	// ignore response ids here + this cannot fail
	_, _, err = collateralEngine.CreateMarketAccounts(context.Background(), mktEngine.GetID(), asset, 0)
	assert.NoError(t, err)

	tm.market = mktEngine
	tm.collateralEngine = collateralEngine
	tm.asset = asset
	tm.mas = mas
	tm.mktCfg = mktCfg

	// Reset event counters
	tm.eventCount = 0
	tm.orderEventCount = 0

	return tm
}

func getMarket(closingAt time.Time, pMonitorSettings *types.PriceMonitoringSettings, openingAuctionDuration *types.AuctionDuration) types.Market {
	mkt := types.Market{
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				LiquidityFee:      "0.3",
				InfrastructureFee: "0.001",
				MakerFee:          "0.004",
			},
		},
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				Id:   "Crypto/ETHUSD/Futures/Dec19",
				Code: "CRYPTO:ETHUSD/DEC19",
				Name: "December 2019 ETH vs USD future",
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				Product: &types.Instrument_Future{
					Future: &types.Future{
						Maturity:        closingAt.Format(time.RFC3339),
						SettlementAsset: "ETH",
						QuoteName:       "USD",
						OracleSpec: &oraclesv1.OracleSpec{
							PubKeys: []string{"0xDEADBEEF"},
							Filters: []*oraclesv1.Filter{
								{
									Key: &oraclesv1.PropertyKey{
										Name: "prices.ETH.value",
										Type: oraclesv1.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*oraclesv1.Condition{},
								},
							},
						},
						OracleSpecBinding: &types.OracleSpecToFutureBinding{
							SettlementPriceProperty: "prices.ETH.value",
						},
					},
				},
			},
			MarginCalculator: &types.MarginCalculator{
				ScalingFactors: &types.ScalingFactors{
					SearchLevel:       1.1,
					InitialMargin:     1.2,
					CollateralRelease: 1.4,
				},
			},
			RiskModel: &types.TradableInstrument_SimpleRiskModel{
				SimpleRiskModel: &types.SimpleRiskModel{
					Params: &types.SimpleModelParams{
						FactorLong:           0.15,
						FactorShort:          0.25,
						MaxMoveUp:            MAXMOVEUP,
						MinMoveDown:          MINMOVEDOWN,
						ProbabilityOfTrading: 0.1,
					},
				},
			},
		},
		OpeningAuction: openingAuctionDuration,
		TradingModeConfig: &types.Market_Continuous{
			Continuous: &types.ContinuousTrading{},
		},
		PriceMonitoringSettings: pMonitorSettings,
		TargetStakeParameters: &types.TargetStakeParameters{
			TimeWindow:    3600,
			ScalingFactor: 10,
		},
	}

	execution.SetMarketID(&mkt, 0)
	return mkt
}

func addAccount(market *testMarket, party string) {
	market.collateralEngine.Deposit(context.Background(), party, market.asset, 1000000000)
}

func addAccountWithAmount(market *testMarket, party string, amnt uint64) {
	market.collateralEngine.Deposit(context.Background(), party, market.asset, amnt)
}

// WithSubmittedLiquidityProvision Submits a Liquidity Provision and asserts that it was created without errors
func (tm *testMarket) WithSubmittedLiquidityProvision(t *testing.T, party, id string, amount uint64, fee string,
	buys, sells []*types.LiquidityOrder) *testMarket {
	ctx := context.Background()

	lps := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: amount,
		Fee:              fee,
		Buys:             buys,
		Sells:            sells,
	}

	require.NoError(t,
		tm.market.SubmitLiquidityProvision(ctx, lps, party, id),
	)

	return tm
}

// WithSubmittedOrder returns a market with Submitted orders defined in `orders`.
// If one submission fails, it will make the test fail and stop.
func (tm *testMarket) WithSubmittedOrders(t *testing.T, orders ...*types.Order) *testMarket {
	ctx := context.Background()
	for i, order := range orders {
		order.MarketId = tm.market.GetID()
		_, err := tm.market.SubmitOrder(ctx, order)
		require.NoError(t, err, "Submitting Order(@index#%d): '%s' failed", i, order.String())
	}
	return tm
}

func TestMarketClosing(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(20, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	defer tm.ctrl.Finish()
	addAccount(tm, party1)
	addAccount(tm, party2)

	// check account gets updated
	closed := tm.market.OnChainTimeUpdate(context.Background(), closingAt.Add(1*time.Second))
	assert.True(t, closed)
}

func TestMarketWithTradeClosing(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(20, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	defer tm.ctrl.Finish()
	// add 2 traders to the party engine
	// this will create 2 traders, credit their account
	// and move some monies to the market
	// this will also output the close accounts
	addAccount(tm, party1)
	addAccount(tm, party2)

	// submit orders
	// party1 buys
	// party2 sells
	orderBuy := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order",
	}
	orderSell := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order",
	}

	// submit orders
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	// tm.transferResponseStore.EXPECT().Add(gomock.Any()).AnyTimes()

	_, err := tm.market.SubmitOrder(context.Background(), orderBuy)
	assert.Nil(t, err)
	if err != nil {
		t.Fail()
	}
	_, err = tm.market.SubmitOrder(context.Background(), orderSell)
	assert.Nil(t, err)
	if err != nil {
		t.Fail()
	}

	// update collateral time first, normally done by execution engine
	futureTime := closingAt.Add(1 * time.Second)
	tm.collateralEngine.OnChainTimeUpdate(context.Background(), futureTime)
	closed := tm.market.OnChainTimeUpdate(context.Background(), futureTime)
	assert.True(t, closed)
}

func TestMarketGetMarginOnNewOrderEmptyBook(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	defer tm.ctrl.Finish()
	// add 2 traders to the party engine
	// this will create 2 traders, credit their account
	// and move some monies to the market
	addAccount(tm, party1)

	// submit orders
	// party1 buys
	// party2 sells
	orderBuy := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order",
	}

	// submit orders
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	// tm.transferResponseStore.EXPECT().Add(gomock.Any()).AnyTimes()

	_, err := tm.market.SubmitOrder(context.Background(), orderBuy)
	assert.Nil(t, err)
	if err != nil {
		t.Fail()
	}
}

func TestMarketGetMarginOnFailNoFund(t *testing.T) {
	party1, party2, party3 := "party1", "party2", "party3"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	defer tm.ctrl.Finish()
	// add 2 traders to the party engine
	// this will create 2 traders, credit their account
	// and move some monies to the market
	addAccountWithAmount(tm, party1, 0)
	addAccountWithAmount(tm, party2, 1000000)
	addAccountWithAmount(tm, party3, 1000000)

	order1 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Id:          "someid12",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-buy-order",
	}
	order2 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Id:          "someid123",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party3,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party3-buy-order",
	}
	_, err := tm.market.SubmitOrder(context.TODO(), order1)
	assert.NoError(t, err)
	confirmation, err := tm.market.SubmitOrder(context.TODO(), order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirmation.Trades))

	// submit orders
	// party1 buys
	// party2 sells
	orderBuy := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order",
	}

	// submit orders
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	// tm.transferResponseStore.EXPECT().Add(gomock.Any()).AnyTimes()

	_, err = tm.market.SubmitOrder(context.Background(), orderBuy)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "margin check failed")
}

func TestMarketGetMarginOnAmendOrderCancelReplace(t *testing.T) {
	party1 := "party1"
	now := time.Unix(100000, 0)
	closingAt := time.Unix(1000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	defer tm.ctrl.Finish()

	addAccount(tm, party1)

	// submit orders
	// party1 buys
	// party2 sells
	orderBuy := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order",
		Version:     execution.InitialOrderVersion,
	}

	// submit orders
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	// tm.transferResponseStore.EXPECT().Add(gomock.Any()).AnyTimes()

	_, err := tm.market.SubmitOrder(context.Background(), orderBuy)
	assert.Nil(t, err)
	if err != nil {
		t.Fail()
	}

	t.Log("amending order now")

	// now try to amend and make sure monies are updated
	amendedOrder := &types.OrderAmendment{
		OrderId:     orderBuy.Id,
		PartyId:     party1,
		Price:       &types.Price{Value: 200},
		SizeDelta:   -50,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		ExpiresAt:   &types.Timestamp{Value: orderBuy.ExpiresAt},
	}

	_, err = tm.market.AmendOrder(context.Background(), amendedOrder)
	if !assert.Nil(t, err) {
		t.Fatalf("Error: %v", err)
	}
}

func TestSetMarketID(t *testing.T) {
	t.Run("nil market config", func(t *testing.T) {
		marketcfg := &types.Market{}
		err := execution.SetMarketID(marketcfg, 0)
		assert.Error(t, err)
	})

	t.Run("good market config", func(t *testing.T) {
		marketcfg := &types.Market{
			Id: "", // ID will be generated
			TradableInstrument: &types.TradableInstrument{
				Instrument: &types.Instrument{
					Id:   "Crypto/ETHUSD/Futures/Dec19",
					Code: "FX:ETHUSD/DEC19",
					Name: "December 2019 ETH vs USD future",
					Metadata: &types.InstrumentMetadata{
						Tags: []string{
							"asset_class:fx/crypto",
							"product:futures",
						},
					},
					Product: &types.Instrument_Future{
						Future: &types.Future{
							Maturity:        "2019-12-31T23:59:59Z",
							SettlementAsset: "Ethereum/Ether",
							OracleSpec: &oraclesv1.OracleSpec{
								PubKeys: []string{"0xDEADBEEF"},
								Filters: []*oraclesv1.Filter{
									{
										Key: &oraclesv1.PropertyKey{
											Name: "prices.ETH.value",
											Type: oraclesv1.PropertyKey_TYPE_INTEGER,
										},
										Conditions: []*oraclesv1.Condition{},
									},
								},
							},
							OracleSpecBinding: &types.OracleSpecToFutureBinding{
								SettlementPriceProperty: "prices.ETH.value",
							},
						},
					},
				},
				RiskModel: &types.TradableInstrument_LogNormalRiskModel{
					LogNormalRiskModel: &types.LogNormalRiskModel{
						RiskAversionParameter: 0.01,
						Tau:                   1.0 / 365.25 / 24,
						Params: &types.LogNormalModelParams{
							Mu:    0,
							R:     0.016,
							Sigma: 0.09,
						},
					},
				},
			},
			TradingModeConfig: &types.Market_Continuous{
				Continuous: &types.ContinuousTrading{},
			},
		}

		err := execution.SetMarketID(marketcfg, 0)
		assert.NoError(t, err)
		fmt.Println(marketcfg.Id)
		id := marketcfg.Id

		err = execution.SetMarketID(marketcfg, 0)
		assert.NoError(t, err)
		assert.Equal(t, id, marketcfg.Id)

		err = execution.SetMarketID(marketcfg, 1)
		assert.NoError(t, err)
		fmt.Println(marketcfg.Id)
		assert.NotEqual(t, id, marketcfg.Id)
	})
}

func TestTriggerByPriceNoTradesInAuction(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	var auctionExtensionSeconds int64 = 45
	auctionEndTime := now.Add(time.Duration(auctionExtensionSeconds) * time.Second)
	afterAuciton := auctionEndTime.Add(time.Nanosecond)
	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{
				{Horizon: 60, Probability: 0.95, AuctionExtension: auctionExtensionSeconds},
			},
		},
		UpdateFrequency: 600,
	}
	var initialPrice uint64 = 100
	var auctionTriggeringPrice = initialPrice + MAXMOVEUP + 1
	tm := getTestMarket(t, now, closingAt, pMonitorSettings, nil)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       initialPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order-1",
	}
	confirmationBuy, err := tm.market.SubmitOrder(context.Background(), orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       initialPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order-1",
	}
	confirmationSell, err := tm.market.SubmitOrder(context.Background(), orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	require.Equal(t, 1, len(confirmationSell.Trades))

	auctionEnd := tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	orderBuy2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order-2",
	}
	confirmationBuy, err = tm.market.SubmitOrder(context.Background(), orderBuy2)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	orderSell2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order-2",
	}
	confirmationSell, err = tm.market.SubmitOrder(context.Background(), orderSell2)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	require.Empty(t, confirmationSell.Trades)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, auctionEndTime.UnixNano(), auctionEnd) // In auction

	closed := tm.market.OnChainTimeUpdate(context.Background(), auctionEndTime)
	assert.False(t, closed)

	closed = tm.market.OnChainTimeUpdate(context.Background(), afterAuciton)
	assert.False(t, closed)
}

func TestTriggerByPriceAuctionPriceInBounds(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	var auctionExtensionSeconds int64 = 45
	auctionEndTime := now.Add(time.Duration(auctionExtensionSeconds) * time.Second)
	afterAuction := auctionEndTime.Add(time.Nanosecond)
	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{
				{Horizon: 60, Probability: 0.95, AuctionExtension: auctionExtensionSeconds},
			},
		},
		UpdateFrequency: 600,
	}
	var initialPrice uint64 = 100
	var validPrice = initialPrice + (MAXMOVEUP+MINMOVEDOWN)/2
	var auctionTriggeringPrice = initialPrice + MAXMOVEUP + 1
	tm := getTestMarket(t, now, closingAt, pMonitorSettings, nil)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       initialPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order-1",
	}
	confirmationSell, err := tm.market.SubmitOrder(context.Background(), orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       initialPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order-1",
	}
	confirmationBuy, err := tm.market.SubmitOrder(context.Background(), orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 1, len(confirmationBuy.Trades))

	auctionEnd := tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	orderSell2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order-2",
	}
	confirmationSell, err = tm.market.SubmitOrder(context.Background(), orderSell2)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order-2",
	}
	confirmationBuy, err = tm.market.SubmitOrder(context.Background(), orderBuy2)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Empty(t, confirmationSell.Trades)

	closed := tm.market.OnChainTimeUpdate(context.Background(), auctionEndTime)
	assert.False(t, closed)

	now = auctionEndTime
	orderSell3 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid6",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       validPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order-3",
	}
	confirmationSell, err = tm.market.SubmitOrder(context.Background(), orderSell3)
	assert.NotNil(t, confirmationSell)
	assert.NoError(t, err)

	orderBuy3 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid5",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       validPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order-3",
	}
	confirmationBuy, err = tm.market.SubmitOrder(context.Background(), orderBuy3)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)
	require.Empty(t, confirmationBuy.Trades)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, auctionEndTime.UnixNano(), auctionEnd) // In auction

	closed = tm.market.OnChainTimeUpdate(context.Background(), afterAuction)
	assert.False(t, closed)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	//TODO: Check that `party2-sell-order-3` & `party1-buy-order-3` get matched in auction and a trade is generated

	// Test that orders get matched as expected upon returning to continuous trading
	now = afterAuction.Add(time.Second)
	orderSell4 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid8",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       validPrice,
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order-4",
	}
	confirmationSell, err = tm.market.SubmitOrder(context.Background(), orderSell4)
	assert.NotNil(t, confirmationSell)
	assert.NoError(t, err)

	orderBuy4 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid7",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       validPrice,
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order-4",
	}
	confirmationBuy, err = tm.market.SubmitOrder(context.Background(), orderBuy4)
	require.NotNil(t, confirmationBuy)
	require.NoError(t, err)
	require.Equal(t, 1, len(confirmationBuy.Trades))

}

func TestTriggerByPriceAuctionPriceOutsideBounds(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	var auctionExtensionSeconds int64 = 45
	auctionEndTime := now.Add(time.Duration(auctionExtensionSeconds) * time.Second)
	initialAuctionEnd := auctionEndTime.Add(time.Second)
	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{
				{Horizon: 60, Probability: 0.95, AuctionExtension: auctionExtensionSeconds},
			},
		},
		UpdateFrequency: 600,
	}
	var initialPrice uint64 = 100
	var auctionTriggeringPrice = initialPrice + MAXMOVEUP + 1
	tm := getTestMarket(t, now, closingAt, pMonitorSettings, nil)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       initialPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order-1",
	}
	confirmationSell, err := tm.market.SubmitOrder(context.Background(), orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       initialPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order-1",
	}
	confirmationBuy, err := tm.market.SubmitOrder(context.Background(), orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 1, len(confirmationBuy.Trades))

	auctionEnd := tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	orderSell2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order-2",
	}
	confirmationSell, err = tm.market.SubmitOrder(context.Background(), orderSell2)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice - 1,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order-2",
	}
	confirmationBuy, err = tm.market.SubmitOrder(context.Background(), orderBuy2)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Empty(t, confirmationBuy.Trades)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	amendedOrder := &types.OrderAmendment{
		OrderId:     orderBuy2.Id,
		PartyId:     party1,
		Price:       &types.Price{Value: auctionTriggeringPrice},
		SizeDelta:   0,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	}

	conf, err := tm.market.AmendOrder(context.Background(), amendedOrder)
	require.NoError(t, err)
	require.NotNil(t, conf)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, auctionEndTime.UnixNano(), auctionEnd) // In auction

	closed := tm.market.OnChainTimeUpdate(context.Background(), auctionEndTime)
	assert.False(t, closed)

	now = auctionEndTime
	orderSell3 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid6",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order-3",
	}
	confirmationSell, err = tm.market.SubmitOrder(context.Background(), orderSell3)
	assert.NotNil(t, confirmationSell)
	assert.NoError(t, err)

	orderBuy3 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid5",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order-3",
	}
	confirmationBuy, err = tm.market.SubmitOrder(context.Background(), orderBuy3)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)
	require.Empty(t, confirmationBuy.Trades)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, auctionEndTime.UnixNano(), auctionEnd) // In auction

	// Expecting market to still be in auction as auction resulted in invalid price
	closed = tm.market.OnChainTimeUpdate(context.Background(), initialAuctionEnd)
	assert.False(t, closed)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction (trigger can only start auction, but can't stop it from concluding at a higher price)
}

func TestTriggerByMarketOrder(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	var auctionExtensionSeconds int64 = 45
	auctionEndTime := now.Add(time.Duration(auctionExtensionSeconds) * time.Second)
	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{
				{Horizon: 60, Probability: 0.95, AuctionExtension: auctionExtensionSeconds},
			},
		},
		UpdateFrequency: 600,
	}
	var initialPrice uint64 = 100
	var auctionTriggeringPriceHigh = initialPrice + MAXMOVEUP + 1
	tm := getTestMarket(t, now, closingAt, pMonitorSettings, nil)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       initialPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order-1",
	}
	confirmationSell, err := tm.market.SubmitOrder(context.Background(), orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       initialPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order-1",
	}
	confirmationBuy, err := tm.market.SubmitOrder(context.Background(), orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 1, len(confirmationBuy.Trades))

	auctionEnd := tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	orderSell2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        3,
		Price:       auctionTriggeringPriceHigh - 1,
		Remaining:   3,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order-2",
	}
	confirmationSell, err = tm.market.SubmitOrder(context.Background(), orderSell2)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	require.Empty(t, confirmationSell.Trades)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	orderSell3 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       auctionTriggeringPriceHigh,
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order-3",
	}
	confirmationSell, err = tm.market.SubmitOrder(context.Background(), orderSell3)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	require.Empty(t, confirmationSell.Trades)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	orderBuy2 := &types.Order{
		Type:      types.Order_TYPE_MARKET,
		Status:    types.Order_STATUS_ACTIVE,
		Id:        "someid5",
		Side:      types.Side_SIDE_BUY,
		PartyId:   party1,
		MarketId:  tm.market.GetID(),
		Size:      4,
		Remaining: 4,
		CreatedAt: now.UnixNano(),
		Reference: "party1-buy-order-2",
	}
	confirmationBuy, err = tm.market.SubmitOrder(context.Background(), orderBuy2)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Empty(t, confirmationSell.Trades)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, auctionEndTime.UnixNano(), auctionEnd) // In auction

	closed := tm.market.OnChainTimeUpdate(context.Background(), auctionEndTime)
	assert.False(t, closed)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, auctionEndTime.UnixNano(), auctionEnd) // Still in auction

	closed = tm.market.OnChainTimeUpdate(context.Background(), auctionEndTime.Add(time.Nanosecond))
	assert.False(t, closed)

	md := tm.market.GetMarketData()
	auctionEnd = md.AuctionEnd
	require.Equal(t, int64(0), auctionEnd) //Not in auction

	require.Equal(t, initialPrice, md.MarkPrice)
}

func TestPriceMonitoringBoundsInGetMarketData(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	t1 := &types.PriceMonitoringTrigger{Horizon: 60, Probability: 0.95, AuctionExtension: 45}
	t2 := &types.PriceMonitoringTrigger{Horizon: 120, Probability: 0.99, AuctionExtension: 90}
	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{
				t1,
				t2,
			},
		},
		UpdateFrequency: 600,
	}
	auctionEndTime := now.Add(time.Duration(t1.AuctionExtension+t2.AuctionExtension) * time.Second)
	var initialPrice uint64 = 100
	var auctionTriggeringPrice = initialPrice + MAXMOVEUP + 1
	tm := getTestMarket(t, now, closingAt, pMonitorSettings, nil)

	expectedPmRange1 := types.PriceMonitoringBounds{
		MinValidPrice:  uint64(int64(initialPrice) + MINMOVEDOWN),
		MaxValidPrice:  initialPrice + MAXMOVEUP,
		Trigger:        t1,
		ReferencePrice: float64(initialPrice),
	}
	expectedPmRange2 := types.PriceMonitoringBounds{
		MinValidPrice:  uint64(int64(initialPrice) + MINMOVEDOWN),
		MaxValidPrice:  initialPrice + MAXMOVEUP,
		Trigger:        t2,
		ReferencePrice: float64(initialPrice),
	}

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       initialPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order-1",
	}
	confirmationBuy, err := tm.market.SubmitOrder(context.Background(), orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       initialPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order-1",
	}
	confirmationSell, err := tm.market.SubmitOrder(context.Background(), orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)
	require.Equal(t, 1, len(confirmationSell.Trades))

	md := tm.market.GetMarketData()
	require.NotNil(t, md)

	auctionEnd := md.AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	pmBounds := md.PriceMonitoringBounds
	require.Equal(t, 2, len(pmBounds))
	require.Equal(t, expectedPmRange1, *pmBounds[0])
	require.Equal(t, expectedPmRange2, *pmBounds[1])

	orderBuy2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order-2",
	}
	confirmationBuy, err = tm.market.SubmitOrder(context.Background(), orderBuy2)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	orderSell2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order-2",
	}
	confirmationSell, err = tm.market.SubmitOrder(context.Background(), orderSell2)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	require.Empty(t, confirmationSell.Trades)

	md = tm.market.GetMarketData()
	require.NotNil(t, md)
	auctionEnd = md.AuctionEnd
	require.Equal(t, auctionEndTime.UnixNano(), auctionEnd) // In auction

	require.Empty(t, md.PriceMonitoringBounds)

	closed := tm.market.OnChainTimeUpdate(context.Background(), auctionEndTime)
	assert.False(t, closed)

	md = tm.market.GetMarketData()
	require.NotNil(t, md)
	auctionEnd = md.AuctionEnd
	require.Equal(t, auctionEndTime.UnixNano(), auctionEnd) // In auction

	require.Empty(t, md.PriceMonitoringBounds)

	closed = tm.market.OnChainTimeUpdate(context.Background(), auctionEndTime.Add(time.Nanosecond))
	assert.False(t, closed)

	md = tm.market.GetMarketData()
	require.NotNil(t, md)
	auctionEnd = md.AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	require.Equal(t, 2, len(md.PriceMonitoringBounds))
	require.Equal(t, expectedPmRange1, *pmBounds[0])
	require.Equal(t, expectedPmRange2, *pmBounds[1])
}

func TestTargetStakeReturnedAndCorrect(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	var oi uint64 = 123
	var matchingPrice uint64 = 111
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	rmParams := tm.mktCfg.TradableInstrument.GetSimpleRiskModel().Params
	expectedTargetStake := float64(matchingPrice*oi) * math.Max(rmParams.FactorLong, rmParams.FactorShort) * tm.mktCfg.TargetStakeParameters.ScalingFactor

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        oi,
		Price:       matchingPrice,
		Remaining:   oi,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order-1",
	}
	confirmationSell, err := tm.market.SubmitOrder(context.Background(), orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        oi,
		Price:       matchingPrice,
		Remaining:   oi,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order-1",
	}
	confirmationBuy, err := tm.market.SubmitOrder(context.Background(), orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 1, len(confirmationBuy.Trades))

	mktData := tm.market.GetMarketData()
	require.NotNil(t, mktData)
	require.Equal(t, strconv.FormatFloat(expectedTargetStake, 'f', -1, 64), mktData.TargetStake)
}

func TestHandleLPCommitmentChange(t *testing.T) {
	ctx := context.Background()
	party1 := "party1"
	party2 := "party2"
	party3 := "party3"
	party4 := "party4"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	var matchingPrice uint64 = 111

	addAccount(tm, party1)
	addAccount(tm, party2)
	addAccount(tm, party3)
	addAccount(tm, party4)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	price := uint64(99)

	order1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party3,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       price,
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		Reference:   "party3-sell-order-1",
	}
	order2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party4,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       price,
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		Reference:   "party4-sell-order-1",
	}
	_, err := tm.market.SubmitOrder(context.TODO(), order1)
	assert.NoError(t, err)
	confirmationSell, err := tm.market.SubmitOrder(ctx, order2)
	assert.NoError(t, err)
	require.Equal(t, 1, len(confirmationSell.Trades))
	order1 = &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid5",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party4,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       price,
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		Reference:   "party5-sell-order-1",
	}
	order2 = &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid6",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party3,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       price,
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		Reference:   "party6-sell-order-1",
	}
	_, err = tm.market.SubmitOrder(context.TODO(), order1)
	assert.NoError(t, err)
	confirmationSell, err = tm.market.SubmitOrder(ctx, order2)
	assert.NoError(t, err)
	require.Equal(t, 1, len(confirmationSell.Trades))

	//TODO (WG 07/01/21): Currently limit orders need to be present on order book for liquidity provision submission to work, remove once fixed.
	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       matchingPrice + 1,
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order-1",
	}
	confirmationSell, err = tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       matchingPrice - 1,
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order-1",
	}
	_, err = tm.market.SubmitOrder(ctx, orderBuy1)
	require.NoError(t, err)

	lp := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 2000,
		Fee:              "0.05",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	require.NoError(t,
		tm.market.SubmitLiquidityProvision(ctx, lp, party1, "id-lp"),
	)

	// this will make current target stake returns 2475
	tm.market.TSCalc().RecordOpenInterest(10, now)

	// by set a very low commitment we should fail
	lp.CommitmentAmount = 1
	require.Equal(t, execution.ErrNotEnoughStake,
		tm.market.SubmitLiquidityProvision(ctx, lp, party1, "id-lp"),
	)

	// 2000 + 600 should be enough to get us on top of the
	// target stake
	lp.CommitmentAmount = 2000 + 600
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(ctx, lp, party1, "id-lp"),
	)

	// 2600 - 125 should be enough to get just at the required stake
	lp.CommitmentAmount = 2600 - 125
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(ctx, lp, party1, "id-lp"),
	)
}

func TestSuppliedStakeReturnedAndCorrect(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	var matchingPrice uint64 = 111

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	//TODO (WG 07/01/21): Currently limit orders need to be present on order book for liquidity provision submission to work, remove once fixed.
	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       matchingPrice + 1,
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order-1",
	}
	confirmationSell, err := tm.market.SubmitOrder(context.Background(), orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       matchingPrice - 1,
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order-1",
	}
	confirmationBuy, err := tm.market.SubmitOrder(context.Background(), orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp1 := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200,
		Fee:              "0.05",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	err = tm.market.SubmitLiquidityProvision(context.Background(), lp1, party1, "id-lp1")
	require.NoError(t, err)

	lp2 := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 100,
		Fee:              "0.06",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	err = tm.market.SubmitLiquidityProvision(context.Background(), lp2, party2, "id-lp2")
	require.NoError(t, err)

	mktData := tm.market.GetMarketData()
	require.NotNil(t, mktData)
	expectedSuppliedStake := lp1.CommitmentAmount + lp2.CommitmentAmount

	require.Equal(t, strconv.FormatUint(expectedSuppliedStake, 10), mktData.SuppliedStake)
}

func TestSubmitLiquidityProvisionWithNoOrdersOnBook(t *testing.T) {
	ctx := context.Background()
	mainParty := "mainParty"
	auxParty := "auxParty"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	var midPrice uint64 = 100

	addAccount(tm, mainParty)
	addAccount(tm, auxParty)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	lp1 := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200,
		Fee:              "0.05",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	err := tm.market.SubmitLiquidityProvision(ctx, lp1, mainParty, "id-lp1")
	require.NoError(t, err)

	orderSell1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "auxParty-sell-order-1", types.Side_SIDE_SELL, auxParty, 1, midPrice+2)

	confirmationSell, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "auxParty-buy-order-1", types.Side_SIDE_BUY, auxParty, 1, midPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	// Check that liquidity orders appear on the book once reference prices exist
	mktData := tm.market.GetMarketData()
	lpOrderVolumeBid := mktData.BestBidVolume - mktData.BestStaticBidVolume
	lpOrderVolumeOffer := mktData.BestOfferVolume - mktData.BestStaticOfferVolume

	var zero uint64 = 0
	require.Greater(t, lpOrderVolumeBid, zero)
	require.Greater(t, lpOrderVolumeOffer, zero)
}

func TestSubmitLiquidityProvisionInOpeningAuction(t *testing.T) {
	ctx := context.Background()
	mainParty := "mainParty"
	auxParty := "auxParty"
	p1, p2 := "party1", "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	var auctionDuration int64 = 5
	tm := getTestMarket2(t, now, closingAt, nil, &types.AuctionDuration{Duration: auctionDuration}, true)
	var midPrice uint64 = 100

	addAccount(tm, mainParty)
	addAccount(tm, auxParty)
	addAccount(tm, p1)
	addAccount(tm, p2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	lp1 := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200,
		Fee:              "0.05",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	require.Equal(t, types.Market_TRADING_MODE_OPENING_AUCTION, tm.market.GetMarketData().MarketTradingMode)

	err := tm.market.SubmitLiquidityProvision(ctx, lp1, mainParty, "id-lp1")
	require.NoError(t, err)

	tradingOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "p1-sell-order", types.Side_SIDE_SELL, p1, 1, midPrice),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "p2-buy-order", types.Side_SIDE_BUY, p2, 1, midPrice),
	}
	for _, o := range tradingOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		assert.NoError(t, err)
		assert.NotNil(t, conf)
	}
	orderSell1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "auxParty-sell-order-1", types.Side_SIDE_SELL, auxParty, 1, midPrice+2)

	confirmationSell, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "auxParty-buy-order-1", types.Side_SIDE_BUY, auxParty, 1, midPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	tm.market.OnChainTimeUpdate(ctx, now.Add(time.Duration((auctionDuration+1)*time.Second.Nanoseconds())))

	// Check that liquidity orders appear on the book once reference prices exist
	mktData := tm.market.GetMarketData()
	lpOrderVolumeBid := mktData.BestBidVolume - mktData.BestStaticBidVolume
	lpOrderVolumeOffer := mktData.BestOfferVolume - mktData.BestStaticOfferVolume

	require.Equal(t, types.Market_TRADING_MODE_CONTINUOUS, mktData.MarketTradingMode)
	var zero uint64 = 0
	require.Greater(t, lpOrderVolumeBid, zero)
	require.Greater(t, lpOrderVolumeOffer, zero)

}

func TestLimitOrderChangesAffectLiquidityOrders(t *testing.T) {
	mainParty := "mainParty"
	auxParty := "auxParty"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	var matchingPrice uint64 = 111
	ctx := context.Background()

	addAccount(tm, mainParty)
	addAccount(tm, auxParty)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-1", types.Side_SIDE_SELL, mainParty, 5, matchingPrice+2)

	confirmationSell, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-buy-order-1", types.Side_SIDE_BUY, mainParty, 4, matchingPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationBuy.Trades))

	mktData := tm.market.GetMarketData()
	require.Equal(t, mktData.BestBidPrice, mktData.BestStaticBidPrice)
	require.Equal(t, mktData.BestBidVolume, mktData.BestStaticBidVolume)
	require.Equal(t, mktData.BestOfferPrice, mktData.BestStaticOfferPrice)
	require.Equal(t, mktData.BestOfferVolume, mktData.BestStaticOfferVolume)

	lp1 := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200,
		Fee:              "0.05",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lp1, mainParty, "id-lp1")
	require.NoError(t, err)

	mktDataPrev := mktData
	mktData = tm.market.GetMarketData()

	require.Greater(t, mktData.BestBidVolume, mktDataPrev.BestStaticBidVolume)
	require.Greater(t, mktData.BestOfferVolume, mktDataPrev.BestStaticOfferVolume)

	mktDataPrev = mktData
	lpOrderVolumeBidPrev := mktData.BestBidVolume - mktData.BestStaticBidVolume
	lpOrderVolumeOfferPrev := mktData.BestOfferVolume - mktData.BestStaticOfferVolume
	// Amend limit order
	amendment := &types.OrderAmendment{
		OrderId:   confirmationBuy.Order.Id,
		PartyId:   confirmationBuy.Order.PartyId,
		SizeDelta: 9,
	}
	_, err = tm.market.AmendOrder(ctx, amendment)
	require.NoError(t, err)

	mktData = tm.market.GetMarketData()
	lpOrderVolumeBid := mktData.BestBidVolume - mktData.BestStaticBidVolume
	lpOrderVolumeOffer := mktData.BestOfferVolume - mktData.BestStaticOfferVolume

	require.Equal(t, mktData.BestStaticOfferVolume, mktDataPrev.BestStaticOfferVolume)
	require.Equal(t, lpOrderVolumeOffer, lpOrderVolumeOfferPrev)
	require.Greater(t, mktData.BestStaticBidVolume, mktDataPrev.BestStaticBidVolume)
	require.Less(t, lpOrderVolumeBid, lpOrderVolumeBidPrev)
	require.Equal(t, uint64(amendment.SizeDelta), lpOrderVolumeBidPrev-lpOrderVolumeBid)

	lpOrderVolumeBidPrev = lpOrderVolumeBid
	lpOrderVolumeOfferPrev = lpOrderVolumeOffer
	mktDataPrev = mktData
	// Submit another non-lp order
	orderSell2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-2", types.Side_SIDE_SELL, mainParty, 3, matchingPrice+3)
	confirmationSell2, err := tm.market.SubmitOrder(ctx, orderSell2)
	require.NotNil(t, confirmationSell2)
	require.NoError(t, err)

	mktData = tm.market.GetMarketData()
	lpOrderVolumeBid = mktData.BestBidVolume - mktData.BestStaticBidVolume
	lpOrderVolumeOffer = mktData.BestOfferVolume - mktData.BestStaticOfferVolume

	require.Equal(t, mktData.BestStaticBidVolume, mktDataPrev.BestStaticBidVolume)
	require.Equal(t, lpOrderVolumeBid, lpOrderVolumeBidPrev)
	require.Equal(t, mktData.BestStaticBidVolume, mktDataPrev.BestStaticBidVolume)
	require.Less(t, lpOrderVolumeOffer, lpOrderVolumeOfferPrev)

	lpOrderVolumeBidPrev = lpOrderVolumeBid
	lpOrderVolumeOfferPrev = lpOrderVolumeOffer
	mktDataPrev = mktData
	// Partial fill of the limit order
	auxOrder1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux-order-1", types.Side_SIDE_BUY, auxParty, orderSell1.Size-1, orderSell1.Price)
	confirmationAux, err := tm.market.SubmitOrder(ctx, auxOrder1)
	assert.NoError(t, err)
	require.Equal(t, 1, len(confirmationAux.Trades))

	mktData = tm.market.GetMarketData()
	lpOrderVolumeBid = mktData.BestBidVolume - mktData.BestStaticBidVolume
	lpOrderVolumeOffer = mktData.BestOfferVolume - mktData.BestStaticOfferVolume

	require.Equal(t, mktData.BestStaticBidVolume, mktDataPrev.BestStaticBidVolume)
	require.Equal(t, lpOrderVolumeBid, lpOrderVolumeBidPrev)
	require.Equal(t, mktData.BestStaticOfferVolume, mktDataPrev.BestStaticOfferVolume-confirmationAux.Trades[0].Size)
	require.Equal(t, lpOrderVolumeOffer, lpOrderVolumeOfferPrev+confirmationAux.Trades[0].Size)

	lpOrderVolumeBidPrev = lpOrderVolumeBid
	lpOrderVolumeOfferPrev = lpOrderVolumeOffer
	mktDataPrev = mktData
	// Cancel limit order
	conf, err := tm.market.CancelOrder(ctx, orderSell1.PartyId, orderSell1.Id)
	require.NoError(t, err)
	require.NotNil(t, conf)

	mktData = tm.market.GetMarketData()
	lpOrderVolumeBid = mktData.BestBidVolume - mktData.BestStaticBidVolume
	lpOrderVolumeOffer = mktData.BestOfferVolume - mktData.BestStaticOfferVolume

	require.Equal(t, mktData.BestStaticBidVolume, mktDataPrev.BestStaticBidVolume)
	require.Equal(t, lpOrderVolumeBid, lpOrderVolumeBidPrev)
	require.Equal(t, mktData.BestStaticOfferVolume, orderSell2.Size)
	require.Greater(t, lpOrderVolumeOffer, lpOrderVolumeOfferPrev)

	lpOrderVolumeBidPrev = lpOrderVolumeBid
	lpOrderVolumeOfferPrev = lpOrderVolumeOffer
	mktDataPrev = mktData
	// Submit another limit order that fills partially on submission
	// Modify LP order so it's not on the best offer
	lp1.Sells[0].Offset = +1
	err = tm.market.SubmitLiquidityProvision(ctx, lp1, mainParty, "id-lp1")
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux-order-2", types.Side_SIDE_SELL, auxParty, 7, matchingPrice+1)
	confirmationAux, err = tm.market.SubmitOrder(ctx, auxOrder2)
	assert.NoError(t, err)
	require.Equal(t, 0, len(confirmationAux.Trades))

	var sizeDiff uint64 = 3
	orderBuy2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-buy-order-2", types.Side_SIDE_BUY, mainParty, auxOrder2.Size+sizeDiff, auxOrder2.Price)
	confirmationBuy2, err := tm.market.SubmitOrder(ctx, orderBuy2)
	require.NoError(t, err)
	require.Equal(t, 1, len(confirmationBuy2.Trades))
	require.Equal(t, auxOrder2.Size, confirmationBuy2.Trades[0].Size)
	require.Equal(t, sizeDiff, orderBuy2.Remaining)

	mktData = tm.market.GetMarketData()
	lpOrderVolumeBid = mktData.BestBidVolume - mktData.BestStaticBidVolume

	require.Equal(t, lpOrderVolumeBid, lpOrderVolumeBidPrev-sizeDiff)

	// Liquidity  order fills entirely
	// First add another limit not to loose the peg reference later on
	orderBuy3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-buy-order-3", types.Side_SIDE_BUY, mainParty, 1, matchingPrice)
	confirmationBuy3, err := tm.market.SubmitOrder(ctx, orderBuy3)
	require.NoError(t, err)
	require.Equal(t, 0, len(confirmationBuy3.Trades))

	mktData = tm.market.GetMarketData()
	lpOrderVolumeBidPrev = mktData.BestBidVolume - mktData.BestStaticBidVolume

	now = now.Add(time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)

	orderBuy2SizeBeforeTrade := orderBuy2.Remaining
	auxOrder3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux-order-3", types.Side_SIDE_SELL, auxParty, 5, matchingPrice+1)
	confirmationAux, err = tm.market.SubmitOrder(ctx, auxOrder3)
	assert.NoError(t, err)
	require.Equal(t, 2, len(confirmationAux.Trades))

	mktData = tm.market.GetMarketData()
	lpOrderVolumeBid = mktData.BestBidVolume - mktData.BestStaticBidVolume

	require.Equal(t, lpOrderVolumeBidPrev+orderBuy2SizeBeforeTrade, lpOrderVolumeBid)

	// Liquidity  order fills partially
	// First add another limit not to loose the peg reference later on
	orderBuy4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-buy-order-4", types.Side_SIDE_BUY, mainParty, 1, matchingPrice-1)
	confirmationBuy4, err := tm.market.SubmitOrder(ctx, orderBuy4)
	require.NoError(t, err)
	require.Equal(t, 0, len(confirmationBuy4.Trades))

	mktData = tm.market.GetMarketData()
	lpOrderVolumeBidPrev = mktData.BestBidVolume - mktData.BestStaticBidVolume

	orderBuy3SizeBeforeTrade := orderBuy3.Remaining
	auxOrder4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux-order-4", types.Side_SIDE_SELL, auxParty, orderBuy3.Size+1, orderBuy3.Price)
	confirmationAux, err = tm.market.SubmitOrder(ctx, auxOrder4)
	assert.NoError(t, err)
	require.Equal(t, 2, len(confirmationAux.Trades))

	mktData = tm.market.GetMarketData()
	lpOrderVolumeBid = mktData.BestBidVolume - mktData.BestStaticBidVolume

	require.Equal(t, lpOrderVolumeBidPrev+orderBuy3SizeBeforeTrade, lpOrderVolumeBid)
}

func getMarketOrder(tm *testMarket,
	now time.Time,
	orderType types.Order_Type,
	orderTIF types.Order_TimeInForce,
	id string,
	side types.Side,
	partyID string,
	size uint64,
	price uint64) *types.Order {
	order := &types.Order{
		Type:        orderType,
		TimeInForce: orderTIF,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          id,
		Side:        side,
		PartyId:     partyID,
		MarketId:    tm.market.GetID(),
		Size:        size,
		Price:       price,
		Remaining:   size,
		CreatedAt:   now.UnixNano(),
		Reference:   "marketorder",
	}
	return order
}

func TestOrderBook_Crash2651(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "613f")
	addAccount(tm, "f9e7")
	addAccount(tm, "98e1")
	addAccount(tm, "qqqq")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Switch to auction mode
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "Order01", types.Side_SIDE_BUY, "613f", 5, 9000)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "Order02", types.Side_SIDE_SELL, "f9e7", 5, 9000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "Order03", types.Side_SIDE_BUY, "613f", 4, 8000)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "Order04", types.Side_SIDE_SELL, "f9e7", 4, 8000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "Order05", types.Side_SIDE_BUY, "613f", 4, 3000)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	o6 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "Order06", types.Side_SIDE_SELL, "f9e7", 3, 3000)
	o6conf, err := tm.market.SubmitOrder(ctx, o6)
	require.NotNil(t, o6conf)
	require.NoError(t, err)

	o7 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order07", types.Side_SIDE_SELL, "f9e7", 20, 0)
	o7.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1000}
	o7conf, err := tm.market.SubmitOrder(ctx, o7)
	require.NotNil(t, o7conf)
	require.NoError(t, err)

	o8 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "Order08", types.Side_SIDE_SELL, "613f", 5, 10001)
	o8conf, err := tm.market.SubmitOrder(ctx, o8)
	require.NotNil(t, o8conf)
	require.NoError(t, err)

	o9 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "Order09", types.Side_SIDE_BUY, "613f", 5, 15001)
	o9conf, err := tm.market.SubmitOrder(ctx, o9)
	require.NotNil(t, o9conf)
	require.NoError(t, err)

	o10 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order10", types.Side_SIDE_BUY, "f9e7", 12, 0)
	o10.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1000}
	o10conf, err := tm.market.SubmitOrder(ctx, o10)
	require.NotNil(t, o10conf)
	require.NoError(t, err)

	o11 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order11", types.Side_SIDE_BUY, "613f", 21, 0)
	o11.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -2000}
	o11conf, err := tm.market.SubmitOrder(ctx, o11)
	require.NotNil(t, o11conf)
	require.NoError(t, err)

	// Leave auction and uncross the book
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))
	require.Equal(t, 3, tm.market.GetPeggedOrderCount())
	require.Equal(t, 3, tm.market.GetParkedOrderCount())

	o12 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order12", types.Side_SIDE_SELL, "613f", 22, 9023)
	o12conf, err := tm.market.SubmitOrder(ctx, o12)
	require.NotNil(t, o12conf)
	require.NoError(t, err)

	o13 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order13", types.Side_SIDE_BUY, "98e1", 23, 11119)
	o13conf, err := tm.market.SubmitOrder(ctx, o13)
	require.NotNil(t, o13conf)
	require.NoError(t, err)

	// This order should cause a crash
	o14 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order14", types.Side_SIDE_BUY, "qqqq", 34, 11513)
	o14conf, err := tm.market.SubmitOrder(ctx, o14)
	require.NotNil(t, o14conf)
	require.NoError(t, err)
}

func TestOrderBook_Crash2599(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "A")
	addAccount(tm, "B")
	addAccount(tm, "C")
	addAccount(tm, "D")
	addAccount(tm, "E")
	addAccount(tm, "F")
	addAccount(tm, "G")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFN, "Order01", types.Side_SIDE_BUY, "A", 5, 11500)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFN, "Order02", types.Side_SIDE_SELL, "B", 25, 11000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFN, "Order03", types.Side_SIDE_BUY, "A", 10, 10500)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	o4 := getMarketOrder(tm, now, types.Order_TYPE_MARKET, types.Order_TIME_IN_FORCE_IOC, "Order04", types.Side_SIDE_SELL, "C", 5, 0)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order05", types.Side_SIDE_BUY, "C", 35, 0)
	o5.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -500}
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	o6 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order06", types.Side_SIDE_BUY, "D", 16, 0)
	o6.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -2000}
	o6conf, err := tm.market.SubmitOrder(ctx, o6)
	require.NotNil(t, o6conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	o7 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order07", types.Side_SIDE_SELL, "E", 19, 0)
	o7.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: +3000}
	o7.ExpiresAt = now.Add(time.Second * 60).UnixNano()
	o7conf, err := tm.market.SubmitOrder(ctx, o7)
	require.NotNil(t, o7conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	o8 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order08", types.Side_SIDE_BUY, "F", 25, 10000)
	o8conf, err := tm.market.SubmitOrder(ctx, o8)
	require.NotNil(t, o8conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	// This one should crash
	o9 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order09", types.Side_SIDE_SELL, "F", 25, 10250)
	o9conf, err := tm.market.SubmitOrder(ctx, o9)
	require.NotNil(t, o9conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	o10 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order10", types.Side_SIDE_BUY, "G", 45, 14000)
	o10conf, err := tm.market.SubmitOrder(ctx, o10)
	require.NotNil(t, o10conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	o11 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order11", types.Side_SIDE_SELL, "G", 45, 8500)
	o11conf, err := tm.market.SubmitOrder(ctx, o11)
	require.NotNil(t, o11conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)
}

func TestTriggerAfterOpeningAuction(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	party3 := "party3"
	party4 := "party4"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	var auctionExtensionSeconds int64 = 45
	openingAuctionDuration := &types.AuctionDuration{Duration: 10}
	openingAuctionEndTime := now.Add(time.Duration(openingAuctionDuration.Duration) * time.Second)
	afterOpeningAuction := openingAuctionEndTime.Add(time.Nanosecond)
	pMonitorAuctionEndTime := afterOpeningAuction.Add(time.Duration(auctionExtensionSeconds) * time.Second)
	afterPMonitorAuction := pMonitorAuctionEndTime.Add(time.Nanosecond)
	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{
				{Horizon: 60, Probability: 0.95, AuctionExtension: auctionExtensionSeconds},
			},
		},
		UpdateFrequency: 600,
	}
	var initialPrice uint64 = 100
	var auctionTriggeringPrice = initialPrice + MAXMOVEUP + 1

	tm := getTestMarket(t, now, closingAt, pMonitorSettings, openingAuctionDuration)

	addAccount(tm, party1)
	addAccount(tm, party2)
	addAccount(tm, party3)
	addAccount(tm, party4)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	gtcOrders := []*types.Order{
		{
			Type:        types.Order_TYPE_LIMIT,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Status:      types.Order_STATUS_ACTIVE,
			Id:          "someid3",
			Side:        types.Side_SIDE_BUY,
			PartyId:     party3,
			MarketId:    tm.market.GetID(),
			Size:        1,
			Price:       initialPrice - 5,
			Remaining:   1,
			CreatedAt:   now.UnixNano(),
			ExpiresAt:   closingAt.UnixNano(),
			Reference:   "party3-buy-order-1",
		},
		{
			Type:        types.Order_TYPE_LIMIT,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Status:      types.Order_STATUS_ACTIVE,
			Id:          "someid4",
			Side:        types.Side_SIDE_SELL,
			PartyId:     party4,
			MarketId:    tm.market.GetID(),
			Size:        1,
			Price:       initialPrice + 10,
			Remaining:   1,
			CreatedAt:   now.UnixNano(),
			Reference:   "party4-sell-order-1",
		}}
	for _, o := range gtcOrders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		assert.NotNil(t, conf)
		assert.NoError(t, err)
	}
	orderBuy1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       initialPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order-1",
	}
	confirmationBuy, err := tm.market.SubmitOrder(context.Background(), orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       initialPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order-1",
	}
	confirmationSell, err := tm.market.SubmitOrder(context.Background(), orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	require.Empty(t, confirmationSell.Trades)

	auctionEnd := tm.market.GetMarketData().AuctionEnd
	require.Equal(t, openingAuctionEndTime.UnixNano(), auctionEnd) // In opening auction

	closed := tm.market.OnChainTimeUpdate(context.Background(), afterOpeningAuction)
	assert.False(t, closed)
	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	// let's cancel the orders we had to place to end opening auction
	for _, o := range gtcOrders {
		_, err := tm.market.CancelOrder(context.Background(), o.PartyId, o.Id)
		assert.NoError(t, err)
	}
	orderBuy2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order-2",
	}
	confirmationBuy, err = tm.market.SubmitOrder(context.Background(), orderBuy2)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	orderSell2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order-2",
	}
	confirmationSell, err = tm.market.SubmitOrder(context.Background(), orderSell2)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	require.Empty(t, confirmationSell.Trades)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, pMonitorAuctionEndTime.UnixNano(), auctionEnd) // In auction

	closed = tm.market.OnChainTimeUpdate(context.Background(), pMonitorAuctionEndTime)
	assert.False(t, closed)

	closed = tm.market.OnChainTimeUpdate(context.Background(), afterPMonitorAuction)
	assert.False(t, closed)
}

func TestOrderBook_Crash2718(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "aaa")
	addAccount(tm, "bbb")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// We start in continuous trading, create order to set best bid
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "aaa", 1, 100)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	// Now the pegged order which will be live
	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "bbb", 1, 0)
	o2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10}
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)
	assert.Equal(t, types.Order_STATUS_ACTIVE, o2.Status)
	assert.Equal(t, uint64(90), o2.Price)

	// Force the pegged order to reprice
	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_BUY, "aaa", 1, 110)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)
	assert.Equal(t, types.Order_STATUS_ACTIVE, o2.Status)
	assert.Equal(t, uint64(100), o2.Price)

	// Flip to auction so the pegged order will be parked
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)
	assert.Equal(t, types.Order_STATUS_PARKED, o2.Status)
	assert.Equal(t, uint64(0), o2.Price)

	// Flip out of auction to un-park it
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))
	assert.Equal(t, types.Order_STATUS_ACTIVE, o2.Status)
	assert.Equal(t, uint64(100), o2.Price)
}

func TestOrderBook_AmendPriceInParkedOrder(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Create a parked pegged order
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "aaa", 1, 0)
	o1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10}
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)
	now = now.Add(time.Second * 1)
	tm.market.OnChainTimeUpdate(context.Background(), now)
	assert.Equal(t, types.Order_STATUS_PARKED, o1.Status)
	assert.Equal(t, uint64(0), o1.Price)

	// Try to amend the price
	amendment := &types.OrderAmendment{
		OrderId: o1.Id,
		PartyId: "aaa",
		Price:   &types.Price{Value: 200},
	}

	// This should fail as we cannot amend a pegged order price
	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	require.Nil(t, amendConf)
	require.Error(t, types.OrderError_ORDER_ERROR_UNABLE_TO_AMEND_PRICE_ON_PEGGED_ORDER, err)
}

func TestOrderBook_ExpiredOrderTriggersReprice(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Create an expiring order
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order01", types.Side_SIDE_BUY, "aaa", 1, 10)
	o1.ExpiresAt = now.Add(5 * time.Second).UnixNano()
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Create a pegged order that references it's price
	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "aaa", 1, 0)
	o2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -2}
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	// Move the clock forward to expire the first order
	now = now.Add(time.Second * 10)
	tm.market.OnChainTimeUpdate(context.Background(), now)
	orders, err := tm.market.RemoveExpiredOrders(context.Background(), now.UnixNano())
	require.NoError(t, err)

	// we have one order
	require.Len(t, orders, 1)
	// id == o1.Id
	assert.Equal(t, o1.Id, orders[0].Id)
	// status is expired
	assert.Equal(t, types.Order_STATUS_EXPIRED, orders[0].Status)
	assert.Equal(t, types.Order_STATUS_PARKED, o2.Status)
}

// This is a scenario to test issue: 2734
// Trader A - 100000000
//  A - Buy 5@15000 GTC
// Trader B - 100000000
//  B - Sell 10 IOC Market
// Trader C - Deposit 100000
//  C - Buy GTT 6@1001 (60s)
// Trader D- Fund 578
//  D - Pegged 3@BA +1
// Trader E - Deposit 100000
//  E - Sell GTC 3@1002
// C amends order price=1002
func TestOrderBook_CrashWithDistressedTraderPeggedOrderNotRemovedFromPeggedList2734(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccountWithAmount(tm, "trader-A", 100000000)
	addAccountWithAmount(tm, "trader-B", 100000000)
	addAccountWithAmount(tm, "trader-C", 100000)
	addAccountWithAmount(tm, "trader-D", 578)
	addAccountWithAmount(tm, "trader-E", 100000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-A", 5, 15000)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_MARKET, types.Order_TIME_IN_FORCE_IOC, "Order02", types.Side_SIDE_SELL, "trader-B", 10, 0)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order03", types.Side_SIDE_BUY, "trader-C", 6, 1001)
	o3.ExpiresAt = now.Add(60 * time.Second).UnixNano()
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_SELL, "trader-D", 3, 0)
	o4.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: +1}
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order05", types.Side_SIDE_SELL, "trader-E", 3, 1002)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	// Try to amend the price
	amendment := &types.OrderAmendment{
		OrderId: o3.Id,
		PartyId: "trader-C",
		Price:   &types.Price{Value: 1002},
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	require.NotNil(t, amendConf)
	require.NoError(t, err)

	// nothing to do we just expect no crash.
}

func TestOrderBook_Crash2733(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := now.Add(120 * time.Second)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{Duration: 30})
	ctx := context.Background()

	addAccountWithAmount(tm, "trader-A", 1000000)
	addAccountWithAmount(tm, "trader-B", 1000000)
	addAccountWithAmount(tm, "trader-C", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	for i := 1; i <= 10; i += 1 {
		o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, fmt.Sprintf("Order1%v", i), types.Side_SIDE_BUY, "trader-A", uint64(i), 0)
		o1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -int64(i * 15)}
		o1conf, err := tm.market.SubmitOrder(ctx, o1)
		require.NotNil(t, o1conf)
		require.NoError(t, err)

		o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, fmt.Sprintf("Order2%v", i), types.Side_SIDE_SELL, "trader-A", uint64(i), 0)
		o2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: int64(i * 10)}
		o2conf, err := tm.market.SubmitOrder(ctx, o2)
		require.NotNil(t, o2conf)
		require.NoError(t, err)

		o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, fmt.Sprintf("Order3%v", i), types.Side_SIDE_BUY, "trader-A", uint64(i), 0)
		o3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -int64(i * 5)}
		o3conf, err := tm.market.SubmitOrder(ctx, o3)
		require.NotNil(t, o3conf)
		require.NoError(t, err)

	}

	// now move time to after auction
	now = now.Add(31 * time.Second)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	for i := 1; i <= 10; i += 1 {
		o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, fmt.Sprintf("Order4%v", i), types.Side_SIDE_SELL, "trader-B", uint64(i), uint64(i*150))
		o1conf, err := tm.market.SubmitOrder(ctx, o1)
		require.NotNil(t, o1conf)
		require.NoError(t, err)

	}

	for i := 1; i <= 20; i += 1 {
		o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, fmt.Sprintf("Order5%v", i), types.Side_SIDE_BUY, "trader-C", uint64(i), uint64(i*100))
		o1conf, err := tm.market.SubmitOrder(ctx, o1)
		require.NotNil(t, o1conf)
		require.NoError(t, err)

	}
}

func TestOrderBook_Bug2747(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccountWithAmount(tm, "trader-A", 100000000)
	addAccountWithAmount(tm, "trader-B", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-A", 100, 0)
	o1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -15}
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Try to amend the price
	amendment := &types.OrderAmendment{
		OrderId:         o1.Id,
		PartyId:         "trader-A",
		PeggedOffset:    &wrapperspb.Int64Value{Value: 20},
		PeggedReference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
	}
	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	assert.Nil(t, amendConf)
	assert.EqualError(t, err, "OrderError: buy cannot reference best ask price")
}

func TestOrderBook_AmendTIME_IN_FORCEForPeggedOrder(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Create a normal order to set a BB price
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "aaa", 1, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Create a pegged order that references the BB price with an expiry time
	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order02", types.Side_SIDE_BUY, "aaa", 1, 0)
	o2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -2}
	o2.ExpiresAt = now.Add(5 * time.Second).UnixNano()
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	// Amend the pegged order from GTT to GTC
	amendment := &types.OrderAmendment{
		OrderId:     o2.Id,
		PartyId:     "aaa",
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	require.NotNil(t, amendConf)
	require.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_ACTIVE, o2.Status)

	// Move the clock forward to expire any old orders
	now = now.Add(time.Second * 10)
	tm.market.OnChainTimeUpdate(context.Background(), now)
	orders, err := tm.market.RemoveExpiredOrders(context.Background(), now.UnixNano())
	require.Equal(t, 0, len(orders))
	require.NoError(t, err)

	// The pegged order should not be expired
	assert.Equal(t, types.Order_STATUS_ACTIVE.String(), o2.Status.String())
	assert.Equal(t, 0, tm.market.GetPeggedExpiryOrderCount())
}

func TestOrderBook_AmendTIME_IN_FORCEForPeggedOrder2(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Create a normal order to set a BB price
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "aaa", 1, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Create a pegged order that references the BB price
	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "aaa", 1, 0)
	o2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -2}
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	// Amend the pegged order so that is has an expiry
	amendment := &types.OrderAmendment{
		OrderId:     o2.Id,
		PartyId:     "aaa",
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		ExpiresAt:   &types.Timestamp{Value: now.Add(5 * time.Second).UnixNano()},
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	require.NotNil(t, amendConf)
	require.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_ACTIVE, o2.Status)
	assert.Equal(t, 1, tm.market.GetPeggedExpiryOrderCount())

	// Move the clock forward to expire any old orders
	now = now.Add(time.Second * 10)
	tm.market.OnChainTimeUpdate(context.Background(), now)
	orders, err := tm.market.RemoveExpiredOrders(context.Background(), now.UnixNano())
	require.NoError(t, err)

	// 1 expired order
	require.Len(t, orders, 1)
	//
	assert.Equal(t, orders[0].Id, o2.Id)
	// The pegged order should be expired
	assert.Equal(t, types.Order_STATUS_EXPIRED, orders[0].Status)
	assert.Equal(t, 0, tm.market.GetPeggedExpiryOrderCount())
}

func TestOrderBook_AmendFilledWithActiveStatus2736(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "trader-A")
	addAccount(tm, "trader-B")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_SELL, "trader-A", 5, 5000)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	assert.NotNil(t, o1conf)
	assert.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 5, 4500)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	assert.NotNil(t, o2conf)
	assert.NoError(t, err)

	// Amend the pegged order so that is has an expiry
	amendment := &types.OrderAmendment{
		OrderId: o2.Id,
		PartyId: "trader-B",
		Price:   &types.Price{Value: 5000},
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_FILLED, o2.Status)
}

func TestOrderBook_PeggedOrderReprice2748(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccountWithAmount(tm, "trader-A", 100000000)
	addAccountWithAmount(tm, "trader-B", 100000000)
	addAccountWithAmount(tm, "trader-C", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// set the mid price first to 6.5k
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-A", 5, 6000)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-B", 5, 7000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	// then place pegged order
	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_BUY, "trader-C", 100, 0)
	o3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -15}
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	assert.Equal(t, o3conf.Order.Status, types.Order_STATUS_ACTIVE)
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())

	// then amend
	// Amend the pegged order so that is has an expiry
	amendment := &types.OrderAmendment{
		OrderId:      o3.Id,
		PartyId:      "trader-C",
		PeggedOffset: &wrapperspb.Int64Value{Value: -6500},
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	require.NotNil(t, amendConf)
	require.NoError(t, err)

	assert.Equal(t, amendConf.Order.Status, types.Order_STATUS_PARKED)
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
}

func TestOrderBook_AmendGFNToGTCOrGTTNotAllowed2486(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccountWithAmount(tm, "trader-A", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// set the mid price first to 6.5k
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFN, "Order01", types.Side_SIDE_BUY, "trader-A", 5, 6000)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// then amend
	// Amend the pegged order so that is has an expiry
	amendment := &types.OrderAmendment{
		OrderId:     o1.Id,
		PartyId:     "trader-A",
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	assert.Nil(t, amendConf)
	assert.EqualError(t, err, "OrderError: Cannot amend TIF from GFA or GFN")
}

func TestOrderBook_CancelAll2771(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccountWithAmount(tm, "trader-A", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_SELL, "trader-A", 1, 0)
	o1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10}
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)
	assert.Equal(t, o1conf.Order.Status, types.Order_STATUS_PARKED)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-A", 1, 0)
	o2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10}
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)
	assert.Equal(t, o2conf.Order.Status, types.Order_STATUS_PARKED)

	confs, err := tm.market.CancelAllOrders(ctx, "trader-A")
	assert.NoError(t, err)
	assert.Len(t, confs, 2)
}

func TestOrderBook_RejectAmendPriceOnPeggedOrder2658(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "trader-A")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-A", 5, 5000)
	o1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -10}
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	assert.NotNil(t, o1conf)
	assert.NoError(t, err)

	// Try to amend the price
	amendment := &types.OrderAmendment{
		OrderId:   o1.Id,
		PartyId:   "trader-A",
		Price:     &types.Price{Value: 4000},
		SizeDelta: +10,
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	assert.Nil(t, amendConf)
	assert.Error(t, types.OrderError_ORDER_ERROR_UNABLE_TO_AMEND_PRICE_ON_PEGGED_ORDER, err)
	assert.Equal(t, types.Order_STATUS_PARKED, o1.Status)
	assert.Equal(t, uint64(1), o1.Version)
}

func TestOrderBook_AmendToCancelForceReprice(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "trader-A")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_SELL, "trader-A", 1, 5000)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	assert.NotNil(t, o1conf)
	assert.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-A", 1, 0)
	o2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10}
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	assert.NotNil(t, o2conf)
	assert.NoError(t, err)

	// Try to amend the price
	amendment := &types.OrderAmendment{
		OrderId:   o1.Id,
		PartyId:   "trader-A",
		SizeDelta: -1,
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_PARKED, o2.Status)
	assert.Equal(t, types.Order_STATUS_CANCELLED, o1.Status)
}

func TestOrderBook_AmendExpPersistParkPeggedOrder(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "trader-A")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_SELL, "trader-A", 10, 4550)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	assert.NotNil(t, o1conf)
	assert.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-A", 105, 0)
	o2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 100}
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	assert.NotNil(t, o2conf)
	assert.NoError(t, err)

	// Try to amend the price
	amendment := &types.OrderAmendment{
		OrderId:   o1.Id,
		PartyId:   "trader-A",
		SizeDelta: -10,
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_PARKED, o2.Status)
	assert.Equal(t, int(o2.Price), 0)
	assert.Equal(t, types.Order_STATUS_CANCELLED, o1.Status)
}

// This test is to make sure when we move into a price monitoring auction that we
// do not allow the parked orders to be repriced.
func TestOrderBook_ParkPeggedOrderWhenMovingToAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "trader-A")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFN, "Order01", types.Side_SIDE_SELL, "trader-A", 10, 1010)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFN, "Order02", types.Side_SIDE_BUY, "trader-A", 10, 990)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "PeggyWeggy", types.Side_SIDE_SELL, "trader-A", 10, 0)
	o3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 100}
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)
	assert.Equal(t, int64(3), tm.market.GetOrdersOnBookCount())

	// Move into a price monitoring auction so that the pegged orders are parked and the other orders are cancelled
	tm.market.StartPriceAuction(now)
	tm.market.EnterAuction(ctx)

	require.Equal(t, 1, tm.market.GetPeggedOrderCount())
	require.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, int64(0), tm.market.GetOrdersOnBookCount())
}

func TestMarket_LeaveAuctionRepricePeggedOrdersShouldFailIfNoMargin(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-C", 1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Start the opening auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -20, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 20, Proportion: 50},
	}

	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000000000,
		Buys:             buys,
		Sells:            sells}

	// Because we do not have enough funds to support our commitment level, we should reject this call
	err := tm.market.SubmitLiquidityProvision(ctx, lps, "trader-C", "LPOrder01")
	require.Error(t, err)
}

func TestMarket_LeaveAuctionAndRepricePeggedOrders(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "trader-A")
	addAccount(tm, "trader-B")
	addAccount(tm, "trader-C")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Start the opening auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Add orders that will outlive the auction to set the reference prices
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_SELL, "trader-A", 10, 1010)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-A", 10, 990)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	require.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -20, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 20, Proportion: 50},
	}

	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000000000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-C", "LPOrder01")
	require.NoError(t, err)

	// Leave the auction so pegged orders are unparked
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))

	// 6 live orders, 2 normal and 4 pegged
	require.Equal(t, int64(6), tm.market.GetOrdersOnBookCount())
	require.Equal(t, 4, tm.market.GetPeggedOrderCount())
	require.Equal(t, 0, tm.market.GetParkedOrderCount())

	// Remove an order to invalidate reference prices and force pegged orders to park
	tm.market.CancelOrder(ctx, o1.PartyId, o1.Id)

	//
	// 1 live orders, 1 normal
	// all LP have been removed as cannot be repriced.
	assert.Equal(t, int64(1), tm.market.GetOrdersOnBookCount())
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

// TODO(): this test is wrong.
// it expects 4 orders to be parked straight away, but we cannot
// initially price the orders as there's no orders in the book.
// this will need to be revisited.
func TestOrderBook_ParkLiquidityProvisionOrders(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1000,
	})
	ctx := context.Background()

	addAccount(tm, "trader-A")

	lp := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200000,
		Fee:              "0.01",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 2000},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 1000},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -1000},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -1500},
		},
	}

	require.NoError(t,
		tm.market.SubmitLiquidityProvision(ctx, lp, "trader-A", "id-lp"),
	)

	// assert.Equal(t,
	// 	len(lp.Sells)+len(lp.Buys),
	// 	tm.market.GetParkedOrderCount(),
	// 	"Market should Park shapes when can't reprice",
	// )
}

func TestOrderBook_RemovingLiquidityProvisionOrders(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1000,
	})
	ctx := context.Background()

	addAccount(tm, "trader-A")

	// Add a LPSubmission
	lp := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200000,
		Fee:              "0.01",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 2000},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 1000},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -1000},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -1500},
		},
	}

	require.NoError(t, tm.market.SubmitLiquidityProvision(ctx, lp, "trader-A", "id-lp"))
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// Remove the LPSubmission by setting the commitment to 0
	lp2 := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 0,
		Fee:              "0.01",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 2000},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 1000},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -1000},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -1500},
		},
	}

	require.NoError(t, tm.market.SubmitLiquidityProvision(ctx, lp2, "trader-A", "id-lp"))
	assert.Equal(t, 0, tm.market.GetLPSCount())
}

func TestOrderBook_ClosingOutLPProviderShouldRemoveCommitment(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1000,
	})
	ctx := context.Background()

	addAccountWithAmount(tm, "trader-A", 2000)
	addAccountWithAmount(tm, "trader-B", 10000000)
	addAccountWithAmount(tm, "trader-C", 10000000)

	// Leave auction right away
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-A", 1, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 1, 50)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_SELL, "trader-C", 10, 50000000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// Create a LP order for trader-A
	lp := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 500,
		Fee:              "0.01",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 25, Offset: 2},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 25, Offset: 3},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 25, Offset: -2},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 25, Offset: -3},
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(ctx, lp, "trader-A", "id-lp"))
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, int64(7), tm.market.GetOrdersOnBookCount())

	// Now move the mark price
	o10 := getMarketOrder(tm, now, types.Order_TYPE_MARKET, types.Order_TIME_IN_FORCE_IOC, "Order05", types.Side_SIDE_BUY, "trader-B", 2, 0)
	o10conf, err := tm.market.SubmitOrder(ctx, o10)
	require.NotNil(t, o10conf)
	require.NoError(t, err)
	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())
}

func TestOrderBook_PartiallyFilledMarketOrderThatWouldWashIOC(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1000,
	})
	ctx := context.Background()

	addAccountWithAmount(tm, "trader-A", 10000000)
	addAccountWithAmount(tm, "trader-B", 10000000)

	// Leave auction right away
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))

	// Create 2 buy orders that we will try to match against
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-B", 10, 100)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-A", 10, 90)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	// Send the sell order with enough volume to match both existing trades
	o3 := getMarketOrder(tm, now, types.Order_TYPE_MARKET, types.Order_TIME_IN_FORCE_IOC, "Order03", types.Side_SIDE_SELL, "trader-A", 20, 0)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_PARTIALLY_FILLED, o3.Status)
	assert.Equal(t, uint64(10), o3.Remaining)
}

func TestOrderBook_PartiallyFilledMarketOrderThatWouldWashFOK(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1000,
	})
	ctx := context.Background()

	addAccountWithAmount(tm, "trader-A", 10000000)
	addAccountWithAmount(tm, "trader-B", 10000000)

	// Leave auction right away
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))

	// Create 2 buy orders that we will try to match against
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-B", 10, 100)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-A", 10, 90)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	// Send the sell order with enough volume to match both existing trades
	o3 := getMarketOrder(tm, now, types.Order_TYPE_MARKET, types.Order_TIME_IN_FORCE_FOK, "Order03", types.Side_SIDE_SELL, "trader-A", 20, 0)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Even though this is FOK we can still partially match if a wash trade is found
	require.Equal(t, types.Order_STATUS_PARTIALLY_FILLED, o3.Status)
	assert.Equal(t, uint64(10), o3.Remaining)
}

func TestOrderBook_PartiallyFilledLimitOrderThatWouldWashFOK(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1000,
	})
	ctx := context.Background()

	addAccountWithAmount(tm, "trader-A", 10000000)
	addAccountWithAmount(tm, "trader-B", 10000000)

	// Leave auction right away
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))

	// Create 2 buy orders that we will try to match against
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-B", 10, 100)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-A", 10, 90)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	// Send the sell order with enough volume to match both existing trades
	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_FOK, "Order03", types.Side_SIDE_SELL, "trader-A", 20, 90)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Even though this is FOK we can still partially match if a wash trade is found
	require.Equal(t, types.Order_STATUS_PARTIALLY_FILLED, o3.Status)
	assert.Equal(t, uint64(10), o3.Remaining)
}

// Tests that during a list of LiquidityProvision order creation (tiggered by
// SubmitLiquidityProvision) fails, the created orders are rolled back.
func TestLPOrdersRollback(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)

	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount("trader-0", 1000000).
		WithAccountAndAmount("trader-1", 1000000).
		WithAccountAndAmount("trader-2", 1000000).
		WithAccountAndAmount("trader-3", 1000000).
		WithAccountAndAmount("trader-4", 1000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	orderParams := []struct {
		id        string
		size      uint64
		side      types.Side
		tif       types.Order_TimeInForce
		pegRef    types.PeggedReference
		pegOffset int64
	}{
		{"trader-4", 1, types.Side_SIDE_BUY, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_BID, -2000},
		{"trader-3", 1, types.Side_SIDE_SELL, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, 1000},
	}
	traderA, traderB := orderParams[0], orderParams[1]

	tpl := OrderTemplate{
		Type: types.Order_TYPE_LIMIT,
	}
	var orders = []*types.Order{
		// Limit Orders
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5500 + traderA.pegOffset), // 3500
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5000 - traderB.pegOffset), // 4000
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-1",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        10,
			Remaining:   10,
			Price:       5500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       5000,
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       3500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       8500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),

		// Pegged Orders
		tpl.New(types.Order{
			PartyId:     traderA.id,
			Side:        traderA.side,
			Size:        traderA.size,
			Remaining:   traderA.size,
			TimeInForce: traderA.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderA.pegRef,
				Offset:    traderA.pegOffset,
			},
		}),
		tpl.New(types.Order{
			PartyId:     traderB.id,
			Side:        traderB.side,
			Size:        traderB.size,
			Remaining:   traderB.size,
			TimeInForce: traderB.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderB.pegRef,
				Offset:    traderB.pegOffset,
			},
		}),
	}

	tm.WithSubmittedOrders(t, orders...)

	lp := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 195000,
		Fee:              "0.01",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 22, Offset: -800},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 64, Offset: -900},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 45, Offset: 1200},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 66, Offset: 1300},
		},
	}

	tm.events = nil
	// Leave the auction
	tm.market.OnChainTimeUpdate(ctx, now.Add(10001*time.Second))
	// reset the registered events
	tm.events = nil

	balanceBeforeLP := tm.PartyGeneralAccount(t, "trader-2").Balance +
		tm.PartyMarginAccount(t, "trader-2").Balance

	err := tm.market.SubmitLiquidityProvision(ctx, lp, "trader-2", "id-lp")
	// require.Error(t, err)
	assert.EqualError(t, err, "margin check failed")

	t.Run("GeneralAccountBalance", func(t *testing.T) {
		newBalance := tm.PartyGeneralAccount(t, "trader-2").Balance +
			tm.PartyMarginAccount(t, "trader-2").Balance

		assert.Equal(t, int(balanceBeforeLP), int(newBalance),
			"Balance should == value before LiquidityProvision",
		)

	})

	t.Run("BondAccountShouldBeZero", func(t *testing.T) {
		bacc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, "trader-2", tm.market.GetID(), tm.asset)
		require.NoError(t, err)
		require.Zero(t, bacc.Balance)
	})

	t.Run("LiquidityProvision_REJECTED", func(t *testing.T) {
		// Filter events until LP is found
		var found types.LiquidityProvision
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				found = evt.LiquidityProvision()
			}
		}

		assert.Equal(t, types.LiquidityProvision_STATUS_REJECTED.String(), found.Status.String())
	})

	t.Run("ExpectedEventStatus", func(t *testing.T) {
		// First collect all the orders events
		found := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found = append(found, evt.Order())
			}
		}

		// one event is sent, this is a rejected event from
		// the first order we try to place, the party does
		// not have enough funds
		expectedStatus := []types.Order_Status{
			types.Order_STATUS_REJECTED, // second gets rejected
		}

		require.Len(t, found, len(expectedStatus))

		for i, status := range expectedStatus {
			got := found[i].Status
			assert.Equal(t, status, got, "Status:", got.String())
		}
	})
}

func Test3008CancelLiquidityProvisionWhenTargetStakeNotReached(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount("trader-0", 1000000).
		WithAccountAndAmount("trader-1", 1000000).
		WithAccountAndAmount("trader-2", 10000000000).
		WithAccountAndAmount("trader-3", 1000000).
		WithAccountAndAmount("trader-4", 1000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	orderParams := []struct {
		id        string
		size      uint64
		side      types.Side
		tif       types.Order_TimeInForce
		pegRef    types.PeggedReference
		pegOffset int64
	}{
		{"trader-4", 1, types.Side_SIDE_BUY, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_BID, -2000},
		{"trader-3", 1, types.Side_SIDE_SELL, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, 1000},
	}
	traderA, traderB := orderParams[0], orderParams[1]

	tpl := OrderTemplate{
		Type: types.Order_TYPE_LIMIT,
	}
	var orders = []*types.Order{
		// Limit Orders
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5500 + traderA.pegOffset), // 3500
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5000 - traderB.pegOffset), // 4000
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-1",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        10,
			Remaining:   10,
			Price:       5500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       5000,
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       3500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       8500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),

		// Pegged Orders
		tpl.New(types.Order{
			PartyId:     traderA.id,
			Side:        traderA.side,
			Size:        traderA.size,
			Remaining:   traderA.size,
			TimeInForce: traderA.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderA.pegRef,
				Offset:    traderA.pegOffset,
			},
		}),
		tpl.New(types.Order{
			PartyId:     traderB.id,
			Side:        traderB.side,
			Size:        traderB.size,
			Remaining:   traderB.size,
			TimeInForce: traderB.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderB.pegRef,
				Offset:    traderB.pegOffset,
			},
		}),
	}

	tm.WithSubmittedOrders(t, orders...)

	// Add a LPSubmission
	lp := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200000,
		Fee:              "0.01",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 2},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 1},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -15},
		},
	}

	// Leave the auction
	tm.market.OnChainTimeUpdate(ctx, now.Add(10001*time.Second))

	require.NoError(t, tm.market.SubmitLiquidityProvision(ctx, lp, "trader-2", "id-lp"))
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// now we do a cancellation
	lpCancel := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 0,
	}

	require.EqualError(t,
		tm.market.SubmitLiquidityProvision(ctx, lpCancel, "trader-2", "id-lp2"),
		"commitment submission rejected, not enouth stake",
	)
}

func Test3008And3007CancelLiquidityProvision(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount("trader-0", 1000000).
		WithAccountAndAmount("trader-1", 1000000).
		WithAccountAndAmount("trader-2", 10000000000).
		// provide stake as well but will cancel
		WithAccountAndAmount("trader-2-bis", 10000000000).
		WithAccountAndAmount("trader-3", 1000000).
		WithAccountAndAmount("trader-4", 1000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	orderParams := []struct {
		id        string
		size      uint64
		side      types.Side
		tif       types.Order_TimeInForce
		pegRef    types.PeggedReference
		pegOffset int64
	}{
		{"trader-4", 1, types.Side_SIDE_BUY, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_BID, -2000},
		{"trader-3", 1, types.Side_SIDE_SELL, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, 1000},
	}
	traderA, traderB := orderParams[0], orderParams[1]

	tpl := OrderTemplate{
		Type: types.Order_TYPE_LIMIT,
	}
	var orders = []*types.Order{
		// Limit Orders
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5500 + traderA.pegOffset), // 3500
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5000 - traderB.pegOffset), // 4000
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-1",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        10,
			Remaining:   10,
			Price:       5500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       5000,
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       3500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       8500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),

		// Pegged Orders
		tpl.New(types.Order{
			PartyId:     traderA.id,
			Side:        traderA.side,
			Size:        traderA.size,
			Remaining:   traderA.size,
			TimeInForce: traderA.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderA.pegRef,
				Offset:    traderA.pegOffset,
			},
		}),
		tpl.New(types.Order{
			PartyId:     traderB.id,
			Side:        traderB.side,
			Size:        traderB.size,
			Remaining:   traderB.size,
			TimeInForce: traderB.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderB.pegRef,
				Offset:    traderB.pegOffset,
			},
		}),
	}

	tm.WithSubmittedOrders(t, orders...)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lp := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 2000000,
		Fee:              "0.01",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 2},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 1},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -15},
		},
	}

	// Leave the auction
	tm.market.OnChainTimeUpdate(ctx, now.Add(10001*time.Second))

	require.NoError(t, tm.market.SubmitLiquidityProvision(ctx, lp, "trader-2", "id-lp"))
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// this is our second stake provider
	// small player
	lp2 := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Fee:              "0.01",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 2},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 1},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -15},
		},
	}

	// cleanup the events, we want to make sure our orders are created
	tm.events = nil

	require.NoError(t, tm.market.SubmitLiquidityProvision(
		ctx, lp2, "trader-2-bis", "id-lp-2"))
	assert.Equal(t, 2, tm.market.GetLPSCount())

	t.Run("ExpectedOrderStatus", func(t *testing.T) {
		// First collect all the orders events
		found := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found = append(found, evt.Order())
			}
		}

		// one event is sent, this is a rejected event from
		// the first order we try to place, the party does
		// not have enough funds
		expectedStatus := []types.Order_Status{
			types.Order_STATUS_ACTIVE,
			types.Order_STATUS_ACTIVE,
			types.Order_STATUS_ACTIVE,
			types.Order_STATUS_ACTIVE,
		}

		require.Len(t, found, len(expectedStatus))

		for i, status := range expectedStatus {
			got := found[i].Status
			assert.Equal(t, status, got, "Status:", got.String())
		}
	})

	tm.market.OnChainTimeUpdate(ctx, now.Add(10011*time.Second))

	// now we do a cancellation
	lpCancel := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 0,
	}

	// cleanup the events before we continue
	tm.events = nil

	require.NoError(t, tm.market.SubmitLiquidityProvision(
		ctx, lpCancel, "trader-2-bis", "id-lp-id3"))
	assert.Equal(t, 1, tm.market.GetLPSCount())

	t.Run("LiquidityProvision_CANCELLED", func(t *testing.T) {
		// Filter events until LP is found
		var found types.LiquidityProvision
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				if evt.LiquidityProvision().PartyId == "trader-2-bis" {
					found = evt.LiquidityProvision()
				}
			}
		}
		assert.Equal(t, types.LiquidityProvision_STATUS_CANCELLED.String(), found.Status.String())
	})

	// now all our orders have been cancelled
	t.Run("ExpectedOrderStatus", func(t *testing.T) {
		// First collect all the orders events
		found := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found = append(found, evt.Order())
			}
		}

		// one event is sent, this is a rejected event from
		// the first order we try to place, the party does
		// not have enough funds
		expectedStatus := []types.Order_Status{
			types.Order_STATUS_CANCELLED,
			types.Order_STATUS_CANCELLED,
			types.Order_STATUS_CANCELLED,
			types.Order_STATUS_CANCELLED,
		}

		require.Len(t, found, len(expectedStatus))

		for i, status := range expectedStatus {
			got := found[i].Status
			assert.Equal(t, status, got, "Status:", got.String())
		}
	})

	// testing #3007 fee transfer are not distributed to cancelled
	// liquidity provisions parties

	newOrder := tpl.New(types.Order{
		MarketId:    tm.market.GetID(),
		Size:        20,
		Remaining:   20,
		Price:       5250,
		Side:        types.Side_SIDE_BUY,
		PartyId:     "trader-0",
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	})

	tm.events = nil
	cnf, err := tm.market.SubmitOrder(ctx, newOrder)
	assert.NoError(t, err)
	assert.True(t, len(cnf.Trades) > 0)

	// clean the events
	// then check for transfer of liquidity fees
	// trader-2-bis should receive none
	tm.events = nil
	tm.market.OnChainTimeUpdate(ctx, now.Add(10021*time.Second))

	t.Run("Fee are distribute to trader-2 only", func(t *testing.T) {
		var found []*types.TransferResponse
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.TransferResponse:
				found = append(found, evt.TransferResponses()...)
			}
		}
		// a single transfer response is required
		require.Len(t, found, 1)
		require.Len(t, found[0].Transfers, 1)
		require.Equal(t, found[0].Transfers[0].Reference, types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_DISTRIBUTE.String())
		require.Len(t, found[0].Balances, 1)
		require.Equal(t, found[0].Balances[0].Account.Owner, "trader-2")
	})

}

func Test2963EnsureMarketValueProxyAndEquitityShareAreInMarketData(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount("trader-0", 1000000).
		WithAccountAndAmount("trader-1", 1000000).
		WithAccountAndAmount("trader-2", 10000000000).
		// provide stake as well but will cancel
		WithAccountAndAmount("trader-2-bis", 10000000000).
		WithAccountAndAmount("trader-3", 1000000).
		WithAccountAndAmount("trader-4", 1000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	orderParams := []struct {
		id        string
		size      uint64
		side      types.Side
		tif       types.Order_TimeInForce
		pegRef    types.PeggedReference
		pegOffset int64
	}{
		{"trader-4", 1, types.Side_SIDE_BUY, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_BID, -2000},
		{"trader-3", 1, types.Side_SIDE_SELL, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, 1000},
	}
	traderA, traderB := orderParams[0], orderParams[1]

	tpl := OrderTemplate{
		Type: types.Order_TYPE_LIMIT,
	}
	var orders = []*types.Order{
		// Limit Orders
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5500 + traderA.pegOffset), // 3500
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5000 - traderB.pegOffset), // 4000
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-1",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        10,
			Remaining:   10,
			Price:       5500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       5000,
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       3500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       8500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),

		// Pegged Orders
		tpl.New(types.Order{
			PartyId:     traderA.id,
			Side:        traderA.side,
			Size:        traderA.size,
			Remaining:   traderA.size,
			TimeInForce: traderA.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderA.pegRef,
				Offset:    traderA.pegOffset,
			},
		}),
		tpl.New(types.Order{
			PartyId:     traderB.id,
			Side:        traderB.side,
			Size:        traderB.size,
			Remaining:   traderB.size,
			TimeInForce: traderB.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderB.pegRef,
				Offset:    traderB.pegOffset,
			},
		}),
	}

	tm.WithSubmittedOrders(t, orders...)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lp := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 2000000,
		Fee:              "0.01",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 2},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 1},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -15},
		},
	}

	// Leave the auction
	tm.market.OnChainTimeUpdate(ctx, now.Add(10001*time.Second))

	require.NoError(t, tm.market.SubmitLiquidityProvision(ctx, lp, "trader-2", "id-lp"))
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// this is our second stake provider
	// small player
	lp2 := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Fee:              "0.01",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 2},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 1},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -15},
		},
	}

	// cleanup the events, we want to make sure our orders are created
	tm.events = nil

	require.NoError(t, tm.market.SubmitLiquidityProvision(
		ctx, lp2, "trader-2-bis", "id-lp-2"))
	assert.Equal(t, 2, tm.market.GetLPSCount())

	tm.market.OnChainTimeUpdate(ctx, now.Add(10011*time.Second))

	mktData := tm.market.GetMarketData()
	assert.Equal(t, mktData.MarketValueProxy, "2001000")
	assert.Len(t, mktData.LiquidityProviderFeeShare, 2)
}

func Test3045DistributeFeesToManyProviders(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount("trader-0", 1000000).
		WithAccountAndAmount("trader-1", 1000000).
		WithAccountAndAmount("trader-2", 10000000000).
		// provide stake as well but will cancel
		WithAccountAndAmount("trader-2-bis", 10000000000).
		WithAccountAndAmount("trader-3", 1000000).
		WithAccountAndAmount("trader-4", 1000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	orderParams := []struct {
		id        string
		size      uint64
		side      types.Side
		tif       types.Order_TimeInForce
		pegRef    types.PeggedReference
		pegOffset int64
	}{
		{"trader-4", 1, types.Side_SIDE_BUY, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_BID, -2000},
		{"trader-3", 1, types.Side_SIDE_SELL, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, 1000},
	}
	traderA, traderB := orderParams[0], orderParams[1]

	tpl := OrderTemplate{
		Type: types.Order_TYPE_LIMIT,
	}
	var orders = []*types.Order{
		// Limit Orders
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5500 + traderA.pegOffset), // 3500
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5000 - traderB.pegOffset), // 4000
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-1",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        10,
			Remaining:   10,
			Price:       5500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       5000,
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       3500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       8500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),

		// Pegged Orders
		tpl.New(types.Order{
			PartyId:     traderA.id,
			Side:        traderA.side,
			Size:        traderA.size,
			Remaining:   traderA.size,
			TimeInForce: traderA.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderA.pegRef,
				Offset:    traderA.pegOffset,
			},
		}),
		tpl.New(types.Order{
			PartyId:     traderB.id,
			Side:        traderB.side,
			Size:        traderB.size,
			Remaining:   traderB.size,
			TimeInForce: traderB.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderB.pegRef,
				Offset:    traderB.pegOffset,
			},
		}),
	}

	tm.WithSubmittedOrders(t, orders...)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lp := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 2000000,
		Fee:              "0.01",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 2},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 1},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -15},
		},
	}

	// Leave the auction
	tm.market.OnChainTimeUpdate(ctx, now.Add(10001*time.Second))

	require.NoError(t, tm.market.SubmitLiquidityProvision(ctx, lp, "trader-2", "id-lp"))
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// this is our second stake provider
	// small player
	lp2 := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Fee:              "0.01",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 2},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 1},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -15},
		},
	}

	// cleanup the events, we want to make sure our orders are created
	tm.events = nil

	require.NoError(t, tm.market.SubmitLiquidityProvision(
		ctx, lp2, "trader-2-bis", "id-lp-2"))
	assert.Equal(t, 2, tm.market.GetLPSCount())

	t.Run("ExpectedOrderStatus", func(t *testing.T) {
		// First collect all the orders events
		found := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found = append(found, evt.Order())
			}
		}

		// one event is sent, this is a rejected event from
		// the first order we try to place, the party does
		// not have enough funds
		expectedStatus := []types.Order_Status{
			types.Order_STATUS_ACTIVE,
			types.Order_STATUS_ACTIVE,
			types.Order_STATUS_ACTIVE,
			types.Order_STATUS_ACTIVE,
		}

		require.Len(t, found, len(expectedStatus))

		for i, status := range expectedStatus {
			got := found[i].Status
			assert.Equal(t, status, got, "Status:", got.String())
		}
	})

	tm.market.OnChainTimeUpdate(ctx, now.Add(10011*time.Second))

	newOrder := tpl.New(types.Order{
		MarketId:    tm.market.GetID(),
		Size:        20,
		Remaining:   20,
		Price:       5250,
		Side:        types.Side_SIDE_BUY,
		PartyId:     "trader-0",
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	})

	tm.events = nil
	cnf, err := tm.market.SubmitOrder(ctx, newOrder)
	assert.NoError(t, err)
	assert.True(t, len(cnf.Trades) > 0)

	// clean the events
	// then check for transfer of liquidity fees
	// trader-2-bis should receive none
	tm.events = nil
	tm.market.OnChainTimeUpdate(ctx, now.Add(10021*time.Second))

	t.Run("Fee are distributed", func(t *testing.T) {
		var found []*types.TransferResponse
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.TransferResponse:
				found = append(found, evt.TransferResponses()...)
			}
		}
		// a single transfer response is required
		require.Len(t, found, 2)
		// require.Len(t, found[0].Transfers, 1)
		// require.Equal(t, found[0].Transfers[0].Reference, types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_DISTRIBUTE.String())
		// require.Len(t, found[0].Balances, 1)
		// require.Equal(t, found[0].Balances[0].Account.Owner, "trader-2")
	})

}

func TestAverageEntryValuation(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	// auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	lpparty := "lp-party-1"
	lpparty2 := "lp-party-2"
	lpparty3 := "lp-party-3"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(lpparty, 500000000000).
		WithAccountAndAmount(lpparty2, 500000000000).
		WithAccountAndAmount(lpparty3, 500000000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(.2)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 8000,
		Fee:              "0.01",
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, &lpSubmission, lpparty, "liquidity-submission-1"),
	)

	lpSubmission2 := lpSubmission
	lpSubmission2.Reference = "lp-submission-2"
	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, &lpSubmission2, lpparty2, "liquidity-submission-2"),
	)

	lpSubmission3 := lpSubmission
	lpSubmission3.Reference = "lp-submission-3"
	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, &lpSubmission3, lpparty3, "liquidity-submission-3"),
	)

	marketData := tm.market.GetMarketData()
	expects := map[string]struct {
		found bool
		value string
	}{
		lpparty:  {value: "0.5454545454545454"},
		lpparty2: {value: "0.2727272727272727"},
		lpparty3: {value: "0.18181818181818182"},
	}

	for _, v := range marketData.LiquidityProviderFeeShare {
		expv, ok := expects[v.Party]
		assert.True(t, ok, "unexpected lp provider in market data", v.Party)
		assert.Equal(t, expv.value, v.EquityLikeShare)
		expv.found = true
		expects[v.Party] = expv
	}

	// now ensure all are found
	for k, v := range expects {
		assert.True(t, v.found, "was not in the list of lp providers", k)
	}
}

func TestBondAccountIsReleasedItMarketRejected(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.WithAccountAndAmount(lpparty, 500000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(0.20)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 150000,
		Fee:              "0.01",
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	t.Run("bond account is updated with the new commitment", func(t *testing.T) {
		bacc, err := tm.collateralEngine.GetPartyBondAccount(
			tm.market.GetID(), lpparty, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, 150000, int(bacc.Balance))
		gacc, err := tm.collateralEngine.GetPartyGeneralAccount(
			lpparty, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, 350000, int(gacc.Balance))
	})

	// now we reject the network and our party bond account should be released to general
	assert.NoError(t,
		tm.market.Reject(context.Background()),
	)

	t.Run("bond is released to general account", func(t *testing.T) {
		// an error as the bon account is being deleted
		_, err := tm.collateralEngine.GetPartyBondAccount(
			tm.market.GetID(), lpparty, tm.asset)
		assert.EqualError(t, err, collateral.ErrAccountDoesNotExist.Error())
		gacc, err := tm.collateralEngine.GetPartyGeneralAccount(
			lpparty, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, 500000, int(gacc.Balance))
	})
}
