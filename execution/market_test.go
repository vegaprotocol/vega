package execution_test

import (
	"context"
	"fmt"
	"math"
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
	"code.vegaprotocol.io/vega/positions"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const MAXMOVEUP = 10
const MINMOVEDOWN = -5

type testMarket struct {
	market          *execution.Market
	log             *logging.Logger
	ctrl            *gomock.Controller
	collateraEngine *collateral.Engine
	broker          *mocks.MockBroker
	now             time.Time
	asset           string
	mas             *monitor.AuctionState
	eventCount      uint64
	orderEventCount uint64
	mktCfg          *types.Market
}

func getTestMarket(t *testing.T, now time.Time, closingAt time.Time, pMonitorSettings *types.PriceMonitoringSettings) *testMarket {
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

	// catch all expected calls
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(
		func(evt events.Event) {
			te := evt.Type()
			if te == events.OrderEvent {
				tm.orderEventCount++
			}
			tm.eventCount++
		},
	)

	collateralEngine, err := collateral.New(log, collateral.NewDefaultConfig(), broker, now)
	assert.Nil(t, err)
	collateralEngine.EnableAsset(context.Background(), types.Asset{
		Symbol: "ETH",
		ID:     "ETH",
	})

	// add the token asset
	tokAsset := types.Asset{
		ID:          "VOTE",
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

	mkts := getMarkets(closingAt, pMonitorSettings)

	mktCfg := &mkts[0]

	mas := monitor.NewAuctionState(mktCfg, now)
	mktEngine, err := execution.NewMarket(context.Background(),
		log, riskConfig, positionConfig, settlementConfig, matchingConfig,
		feeConfig, collateralEngine, mktCfg, now, broker, execution.NewIDGen(), mas)
	assert.NoError(t, err)

	asset, err := mkts[0].GetAsset()
	assert.NoError(t, err)

	// ignore response ids here + this cannot fail
	_, _, err = collateralEngine.CreateMarketAccounts(context.Background(), mktEngine.GetID(), asset, 0)
	assert.NoError(t, err)

	tm.market = mktEngine
	tm.collateraEngine = collateralEngine
	tm.asset = asset
	tm.mas = mas
	tm.mktCfg = mktCfg

	// Reset event counters
	tm.eventCount = 0
	tm.orderEventCount = 0

	return tm
}

func getMarkets(closingAt time.Time, pMonitorSettings *types.PriceMonitoringSettings) []types.Market {
	mkt := types.Market{
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				LiquidityFee:      "0.001",
				InfrastructureFee: "0.0005",
				MakerFee:          "0.00025",
			},
		},
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				Id:        "Crypto/ETHUSD/Futures/Dec19",
				Code:      "CRYPTO:ETHUSD/DEC19",
				Name:      "December 2019 ETH vs USD future",
				QuoteName: "USD",
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				InitialMarkPrice: 99,
				Product: &types.Instrument_Future{
					Future: &types.Future{
						Maturity: closingAt.Format(time.RFC3339),
						Oracle: &types.Future_EthereumEvent{
							EthereumEvent: &types.EthereumEvent{
								ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
								Event:      "price_changed",
							},
						},
						Asset: "ETH",
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
						FactorLong:  0.15,
						FactorShort: 0.25,
						MaxMoveUp:   MAXMOVEUP,
						MinMoveDown: MINMOVEDOWN,
					},
				},
			},
		},
		// For now all tests are done without an opening auction
		//		OpeningAuction: &types.AuctionDuration{},
		TradingMode: &types.Market_Continuous{
			Continuous: &types.ContinuousTrading{},
		},
		PriceMonitoringSettings: pMonitorSettings,
		TargetStakeParameters: &types.TargetStakeParameters{
			TimeWindow:    3600,
			ScalingFactor: 10,
		},
	}

	execution.SetMarketID(&mkt, 0)
	return []types.Market{mkt}
}

func addAccount(market *testMarket, party string) {
	market.collateraEngine.Deposit(context.Background(), party, market.asset, 1000000000)
	market.broker.EXPECT().Send(gomock.Any()).AnyTimes()
}
func addAccountWithAmount(market *testMarket, party string, amnt uint64) {
	market.collateraEngine.Deposit(context.Background(), party, market.asset, amnt)
	market.broker.EXPECT().Send(gomock.Any()).AnyTimes()
}

func TestMarketClosing(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(20, 0)
	tm := getTestMarket(t, now, closingAt, nil)
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
	tm := getTestMarket(t, now, closingAt, nil)
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
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order",
	}
	orderSell := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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

	// update collateral time first, normally done by execution engin
	futureTime := closingAt.Add(1 * time.Second)
	tm.collateraEngine.OnChainTimeUpdate(context.Background(), futureTime)
	closed := tm.market.OnChainTimeUpdate(context.Background(), futureTime)
	assert.True(t, closed)
}

func TestMarketGetMarginOnNewOrderEmptyBook(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil)
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
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil)
	defer tm.ctrl.Finish()
	// add 2 traders to the party engine
	// this will create 2 traders, credit their account
	// and move some monies to the market
	addAccountWithAmount(tm, party1, 0)

	// submit orders
	// party1 buys
	// party2 sells
	orderBuy := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
	assert.NotNil(t, err)
	assert.EqualError(t, err, "margin check failed")
}

func TestMarketGetMarginOnAmendOrderCancelReplace(t *testing.T) {
	party1 := "party1"
	now := time.Unix(100000, 0)
	closingAt := time.Unix(1000000, 0)
	tm := getTestMarket(t, now, closingAt, nil)
	defer tm.ctrl.Finish()

	addAccount(tm, party1)

	// submit orders
	// party1 buys
	// party2 sells
	orderBuy := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
		OrderID:     orderBuy.Id,
		PartyID:     party1,
		Price:       &types.Price{Value: 200},
		SizeDelta:   -50,
		TimeInForce: types.Order_TIF_GTT,
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
							Maturity: "2019-12-31T23:59:59Z",
							Oracle: &types.Future_EthereumEvent{
								EthereumEvent: &types.EthereumEvent{
									ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
									Event:      "price_changed",
								},
							},
							Asset: "Ethereum/Ether",
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
			TradingMode: &types.Market_Continuous{
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
	var auctionTriggeringPrice uint64 = initialPrice + MAXMOVEUP + 1
	tm := getTestMarket(t, now, closingAt, pMonitorSettings)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order-2",
	}
	confirmationSell, err = tm.market.SubmitOrder(context.Background(), orderSell2)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	require.Equal(t, 0, len(confirmationSell.Trades))

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
	var validPrice uint64 = initialPrice + (MAXMOVEUP+MINMOVEDOWN)/2
	var auctionTriggeringPrice uint64 = initialPrice + MAXMOVEUP + 1
	tm := getTestMarket(t, now, closingAt, pMonitorSettings)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order-2",
	}
	confirmationBuy, err = tm.market.SubmitOrder(context.Background(), orderBuy2)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationSell.Trades))

	closed := tm.market.OnChainTimeUpdate(context.Background(), auctionEndTime)
	assert.False(t, closed)

	now = auctionEndTime
	orderSell3 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GFA,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid6",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_GFA,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid5",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
	require.Equal(t, 0, len(confirmationBuy.Trades))

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, auctionEndTime.UnixNano(), auctionEnd) // In auction

	closed = tm.market.OnChainTimeUpdate(context.Background(), afterAuciton)
	assert.False(t, closed)

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	//TODO: Check that `party2-sell-order-3` & `party1-buy-order-3` get matched in auction and a trade is generated

	// Test that orders get matched as expected upon returning to continous trading
	now = afterAuciton.Add(time.Second)
	orderSell4 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid8",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid7",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
	var auctionTriggeringPrice uint64 = initialPrice + MAXMOVEUP + 1
	tm := getTestMarket(t, now, closingAt, pMonitorSettings)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order-2",
	}
	confirmationBuy, err = tm.market.SubmitOrder(context.Background(), orderBuy2)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationSell.Trades))

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, auctionEndTime.UnixNano(), auctionEnd) // In auction

	closed := tm.market.OnChainTimeUpdate(context.Background(), auctionEndTime)
	assert.False(t, closed)

	now = auctionEndTime
	orderSell3 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GFA,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid6",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_GFA,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid5",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
	require.Equal(t, 0, len(confirmationBuy.Trades))

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
	var auctionTriggeringPriceHigh uint64 = initialPrice + MAXMOVEUP + 1
	tm := getTestMarket(t, now, closingAt, pMonitorSettings)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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

	require.Equal(t, 0, len(confirmationSell.Trades))

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	orderSell3 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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

	require.Equal(t, 0, len(confirmationSell.Trades))

	auctionEnd = tm.market.GetMarketData().AuctionEnd
	require.Equal(t, int64(0), auctionEnd) // Not in auction

	orderBuy2 := &types.Order{
		Type:      types.Order_TYPE_MARKET,
		Status:    types.Order_STATUS_ACTIVE,
		Id:        "someid5",
		Side:      types.Side_SIDE_BUY,
		PartyID:   party1,
		MarketID:  tm.market.GetID(),
		Size:      4,
		Remaining: 4,
		CreatedAt: now.UnixNano(),
		Reference: "party1-buy-order-2",
	}
	confirmationBuy, err = tm.market.SubmitOrder(context.Background(), orderBuy2)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationSell.Trades))

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

func TestTargetStakeReturnedAndCorrect(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	var oi uint64 = 123
	var matchingPrice uint64 = 111
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil)

	rmParams := tm.mktCfg.TradableInstrument.GetSimpleRiskModel().Params
	expectedTargetStake := float64(oi) * math.Max(rmParams.FactorLong, rmParams.FactorShort) * tm.mktCfg.TargetStakeParameters.ScalingFactor

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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

	require.Equal(t, fmt.Sprintf("%.f", expectedTargetStake), mktData.TargetStake)

}
