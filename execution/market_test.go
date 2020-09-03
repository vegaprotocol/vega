package execution_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/execution/mocks"
	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/positions"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testMarket struct {
	market          *execution.Market
	log             *logging.Logger
	ctrl            *gomock.Controller
	collateraEngine *collateral.Engine
	broker          *mocks.MockBroker
	now             time.Time
	asset           string
	auctionTriggers []*mocks.MockAuctionTrigger
}

func getTestMarket(t *testing.T, now time.Time, closingAt time.Time, numberOfTriggers int) *testMarket {
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	riskConfig := risk.NewDefaultConfig()
	positionConfig := positions.NewDefaultConfig()
	settlementConfig := settlement.NewDefaultConfig()
	matchingConfig := matching.NewDefaultConfig()
	feeConfig := fee.NewDefaultConfig()

	broker := mocks.NewMockBroker(ctrl)

	// catch all expected calls
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	collateralEngine, err := collateral.New(log, collateral.NewDefaultConfig(), broker, now)
	assert.Nil(t, err)
	collateralEngine.EnableAsset(context.Background(), types.Asset{
		Symbol: "ETH",
		ID:     "ETH",
	})

	// add the token asset
	tokAsset := types.Asset{
		ID:          collateral.TokenAssetSource.GetBuiltinAsset().Symbol,
		Name:        collateral.TokenAssetSource.GetBuiltinAsset().Name,
		Symbol:      collateral.TokenAssetSource.GetBuiltinAsset().Symbol,
		Decimals:    collateral.TokenAssetSource.GetBuiltinAsset().Decimals,
		TotalSupply: collateral.TokenAssetSource.GetBuiltinAsset().TotalSupply,
		Source:      collateral.TokenAssetSource,
	}
	collateralEngine.EnableAsset(context.Background(), tokAsset)

	mkts := getMarkets(closingAt)
	mockTriggers := getMockTriggers(numberOfTriggers, ctrl)

	triggers := make([]execution.AuctionTrigger, len(mockTriggers))
	for i, mt := range mockTriggers {
		triggers[i] = mt
	}

	mktEngine, err := execution.NewMarket(
		log, riskConfig, positionConfig, settlementConfig, matchingConfig,
		feeConfig, collateralEngine, &mkts[0], now, broker, execution.NewIDGen(), triggers)
	assert.NoError(t, err)

	asset, err := mkts[0].GetAsset()
	assert.NoError(t, err)

	// ignore response ids here + this cannot fail
	_, _, err = collateralEngine.CreateMarketAccounts(context.Background(), mktEngine.GetID(), asset, 0)
	assert.NoError(t, err)

	return &testMarket{
		market:          mktEngine,
		log:             log,
		ctrl:            ctrl,
		collateraEngine: collateralEngine,
		broker:          broker,
		now:             now,
		asset:           asset,
		auctionTriggers: mockTriggers,
	}
}

func getMarkets(closingAt time.Time) []types.Market {
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
				BaseName:  "ETH",
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
					},
				},
			},
		},
		OpeningAuction: &types.AuctionDuration{},
		TradingMode: &types.Market_Continuous{
			Continuous: &types.ContinuousTrading{},
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

func getMockTriggers(numberOfTriggers int, ctrl *gomock.Controller) []*mocks.MockAuctionTrigger {
	triggers := make([]*mocks.MockAuctionTrigger, numberOfTriggers)
	for i := 0; i < numberOfTriggers; i++ {
		triggers[i] = mocks.NewMockAuctionTrigger(ctrl)
	}
	return triggers
}

func TestMarketClosing(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(20, 0)
	tm := getTestMarket(t, now, closingAt, 0)
	defer tm.ctrl.Finish()
	addAccount(tm, party1)
	addAccount(tm, party2)

	// check account gets updated
	closed := tm.market.OnChainTimeUpdate(closingAt.Add(1 * time.Second))
	assert.True(t, closed)
}

func TestMarketWithTradeClosing(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(20, 0)
	tm := getTestMarket(t, now, closingAt, 0)
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

	_, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.Nil(t, err)
	if err != nil {
		t.Fail()
	}
	_, err = tm.market.SubmitOrder(context.TODO(), orderSell)
	assert.Nil(t, err)
	if err != nil {
		t.Fail()
	}

	// update collateral time first, normally done by execution engin
	futureTime := closingAt.Add(1 * time.Second)
	tm.collateraEngine.OnChainTimeUpdate(futureTime)
	closed := tm.market.OnChainTimeUpdate(futureTime)
	assert.True(t, closed)
}

func TestMarketGetMarginOnNewOrderEmptyBook(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, 0)
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

	_, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.Nil(t, err)
	if err != nil {
		t.Fail()
	}
}

func TestMarketGetMarginOnFailNoFund(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, 0)
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
	tm := getTestMarket(t, now, closingAt, 0)
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

	_, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
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

	_, err = tm.market.AmendOrder(context.TODO(), amendedOrder)
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

func TestMarketCancelOrder(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, 0)

	addAccount(tm, party1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

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
	}
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	cancelled, err := tm.market.CancelOrderByID(confirmation.Order.Id)
	assert.NotNil(t, cancelled, "cancelled freshly submitted order")
	assert.NoError(t, err)
	assert.EqualValues(t, confirmation.Order.Id, cancelled.Order.Id)

	cancelled, err = tm.market.CancelOrderByID(confirmation.Order.Id)
	assert.Nil(t, cancelled, "cancelling same order twice should not work")
	assert.Error(t, err, "it should be an error to cancel same order twice")

	cancelled, err = tm.market.CancelOrderByID("an id that does not exist")
	assert.Nil(t, cancelled, "cancelling non-exitant order should not work")
	assert.Error(t, err, "it should be an error to cancel an order that does not exist")
}

func TestTriggerByTime(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	auctionStartTime := now.Add(1 * time.Second)
	stillAuction := auctionStartTime.Add(1 * time.Second)
	auctionEndTime := stillAuction.Add(1 * time.Second)
	tm := getTestMarket(t, now, closingAt, 1)

	tm.auctionTriggers[0].EXPECT().EnterPerTime(now).Return(false).Times(1)

	closed := tm.market.OnChainTimeUpdate(now)
	assert.False(t, closed)
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_CONTINUOUS), int32(tm.market.GetTradingMode()))

	tm.auctionTriggers[0].EXPECT().EnterPerTime(auctionStartTime).Return(true).Times(1)

	closed = tm.market.OnChainTimeUpdate(auctionStartTime)
	assert.False(t, closed)
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_AUCTION), int32(tm.market.GetTradingMode()))

	tm.auctionTriggers[0].EXPECT().LeavePerTime(stillAuction).Return(false).Times(1)

	closed = tm.market.OnChainTimeUpdate(stillAuction)
	assert.False(t, closed)
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_AUCTION), int32(tm.market.GetTradingMode()))

	tm.auctionTriggers[0].EXPECT().LeavePerTime(auctionEndTime).Return(true).Times(1)
	tm.auctionTriggers[0].EXPECT().EnterPerTime(auctionEndTime).Return(false).Times(1)
	tm.auctionTriggers[0].EXPECT().EnterPerPrice(gomock.Any()).Return(false).Times(1)

	closed = tm.market.OnChainTimeUpdate(auctionEndTime)
	assert.False(t, closed)
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_CONTINUOUS), int32(tm.market.GetTradingMode()))

}

func TestTriggerByPriceNoTradesInAuction(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	var auctionTriggeringPrice uint64 = 110
	stillAuction := now.Add(10 * time.Second)
	auctionEndTime := stillAuction.Add(1 * time.Minute)

	tm := getTestMarket(t, now, closingAt, 1)
	tm.auctionTriggers[0].EXPECT().EnterPerPrice(auctionTriggeringPrice).Return(true).Times(1)
	tm.auctionTriggers[0].EXPECT().LeavePerTime(stillAuction).Return(false).Times(1)
	tm.auctionTriggers[0].EXPECT().LeavePerTime(auctionEndTime).Return(true).Times(1)
	tm.auctionTriggers[0].EXPECT().EnterPerTime(auctionEndTime).Return(false).Times(1)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order",
	}
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	orderSell := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order",
	}
	confirmation, err = tm.market.SubmitOrder(context.TODO(), orderSell)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	tradingMode := tm.market.GetTradingMode()
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_AUCTION), int32(tradingMode))
	closed := tm.market.OnChainTimeUpdate(stillAuction)
	assert.False(t, closed)
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_AUCTION), int32(tm.market.GetTradingMode()))

	closed = tm.market.OnChainTimeUpdate(auctionEndTime)
	assert.False(t, closed)
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_CONTINUOUS), int32(tm.market.GetTradingMode()))

}

func TestTriggerByPriceValidPriceInAuction(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	var auctionTriggeringPrice uint64 = 110
	var nonTriggeringUncrossingPrice uint64 = auctionTriggeringPrice - 1
	stillAuction := now.Add(10 * time.Second)
	auctionEndTime := stillAuction.Add(1 * time.Minute)

	tm := getTestMarket(t, now, closingAt, 1)
	tm.auctionTriggers[0].EXPECT().EnterPerPrice(auctionTriggeringPrice).Return(true).Times(1)
	tm.auctionTriggers[0].EXPECT().LeavePerTime(stillAuction).Return(false).Times(1)
	tm.auctionTriggers[0].EXPECT().LeavePerTime(auctionEndTime).Return(true).Times(1)
	tm.auctionTriggers[0].EXPECT().EnterPerTime(auctionEndTime).Return(false).Times(1)
	tm.auctionTriggers[0].EXPECT().EnterPerPrice(nonTriggeringUncrossingPrice).Return(false).Times(1)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order",
	}
	confirmationBuy, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirmationBuy.Trades))

	orderSell := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order",
	}
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderSell)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_AUCTION), int32(tm.market.GetTradingMode()))

	cancelled, err := tm.market.CancelOrderByID(confirmationBuy.Order.Id)
	assert.NotNil(t, cancelled, "cancelled freshly submitted order")
	assert.NoError(t, err)
	assert.EqualValues(t, confirmationBuy.Order.Id, cancelled.Order.Id)

	closed := tm.market.OnChainTimeUpdate(stillAuction)
	assert.False(t, closed)
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_AUCTION), int32(tm.market.GetTradingMode()))

	orderBuy2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       nonTriggeringUncrossingPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order",
	}
	confirmation, err = tm.market.SubmitOrder(context.TODO(), orderBuy2)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirmation.Trades))

	orderSell2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       nonTriggeringUncrossingPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order",
	}

	confirmation, err = tm.market.SubmitOrder(context.TODO(), orderSell2)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirmation.Trades))

	closed = tm.market.OnChainTimeUpdate(auctionEndTime)
	assert.False(t, closed)
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_CONTINUOUS), int32(tm.market.GetTradingMode()))
}

func TestTriggerByPriceExitStoppedByOtherTirggerPrice(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	var trigger1Price uint64 = 110
	var trigger2Price uint64 = trigger1Price - 1
	stillAuction := now.Add(10 * time.Second)
	auctionEndTime := stillAuction.Add(1 * time.Minute)

	tm := getTestMarket(t, now, closingAt, 2)
	tm.auctionTriggers[0].EXPECT().EnterPerPrice(trigger1Price).Return(true).Times(1)
	tm.auctionTriggers[1].EXPECT().EnterPerPrice(trigger1Price).Return(false).Times(1) //Trigger 1 doesn't mind the price at this stage
	tm.auctionTriggers[0].EXPECT().LeavePerTime(stillAuction).Return(false).Times(1)
	tm.auctionTriggers[1].EXPECT().LeavePerTime(stillAuction).Return(true).Times(1)
	tm.auctionTriggers[0].EXPECT().LeavePerTime(auctionEndTime).Return(true).Times(1)
	tm.auctionTriggers[1].EXPECT().LeavePerTime(auctionEndTime).Return(true).Times(1)
	tm.auctionTriggers[0].EXPECT().EnterPerTime(auctionEndTime).Return(false).Times(1)
	tm.auctionTriggers[1].EXPECT().EnterPerTime(auctionEndTime).Return(false).Times(1)
	tm.auctionTriggers[0].EXPECT().EnterPerPrice(trigger2Price).Return(false).Times(1)
	tm.auctionTriggers[1].EXPECT().EnterPerPrice(trigger2Price).Return(true).Times(1)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       trigger1Price,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order",
	}
	confirmationBuy, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirmationBuy.Trades))

	orderSell := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       trigger1Price,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order",
	}
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderSell)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_AUCTION), int32(tm.market.GetTradingMode()))

	cancelled, err := tm.market.CancelOrderByID(confirmationBuy.Order.Id)
	assert.NotNil(t, cancelled, "cancelled freshly submitted order")
	assert.NoError(t, err)
	assert.EqualValues(t, confirmationBuy.Order.Id, cancelled.Order.Id)

	closed := tm.market.OnChainTimeUpdate(stillAuction)
	assert.False(t, closed)
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_AUCTION), int32(tm.market.GetTradingMode()))

	orderBuy2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       trigger2Price,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order",
	}
	confirmation, err = tm.market.SubmitOrder(context.TODO(), orderBuy2)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirmation.Trades))

	orderSell2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       trigger2Price,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order",
	}

	confirmation, err = tm.market.SubmitOrder(context.TODO(), orderSell2)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirmation.Trades))

	closed = tm.market.OnChainTimeUpdate(auctionEndTime)
	assert.False(t, closed)
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_AUCTION), int32(tm.market.GetTradingMode()))
}

func TestTriggerByPriceExitStoppedByOtherTirggerTime(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	var auctionTriggeringPrice uint64 = 110
	var nonTriggeringPrice uint64 = auctionTriggeringPrice - 1
	stillAuction := now.Add(10 * time.Second)
	auctionEndTime := stillAuction.Add(1 * time.Minute)

	tm := getTestMarket(t, now, closingAt, 2)
	tm.auctionTriggers[0].EXPECT().EnterPerPrice(auctionTriggeringPrice).Return(true).Times(1)
	tm.auctionTriggers[1].EXPECT().EnterPerPrice(auctionTriggeringPrice).Return(false).Times(1) //Trigger 1 doesn't mind the price at this stage
	tm.auctionTriggers[0].EXPECT().LeavePerTime(stillAuction).Return(false).Times(1)
	tm.auctionTriggers[1].EXPECT().LeavePerTime(stillAuction).Return(true).Times(1)
	tm.auctionTriggers[0].EXPECT().LeavePerTime(auctionEndTime).Return(true).Times(1)
	tm.auctionTriggers[1].EXPECT().LeavePerTime(auctionEndTime).Return(true).Times(1)
	tm.auctionTriggers[0].EXPECT().EnterPerTime(auctionEndTime).Return(false).Times(1)
	tm.auctionTriggers[1].EXPECT().EnterPerTime(auctionEndTime).Return(true).Times(1)
	tm.auctionTriggers[0].EXPECT().EnterPerPrice(nonTriggeringPrice).Return(false).Times(1)
	tm.auctionTriggers[1].EXPECT().EnterPerPrice(nonTriggeringPrice).Return(false).Times(1)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order",
	}
	confirmationBuy, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirmationBuy.Trades))

	orderSell := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_FOK,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       auctionTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order",
	}
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderSell)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_AUCTION), int32(tm.market.GetTradingMode()))

	cancelled, err := tm.market.CancelOrderByID(confirmationBuy.Order.Id)
	assert.NotNil(t, cancelled, "cancelled freshly submitted order")
	assert.NoError(t, err)
	assert.EqualValues(t, confirmationBuy.Order.Id, cancelled.Order.Id)

	closed := tm.market.OnChainTimeUpdate(stillAuction)
	assert.False(t, closed)
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_AUCTION), int32(tm.market.GetTradingMode()))

	orderBuy2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid3",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       nonTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party1-buy-order",
	}
	confirmation, err = tm.market.SubmitOrder(context.TODO(), orderBuy2)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirmation.Trades))

	orderSell2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid4",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       nonTriggeringPrice,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order",
	}

	confirmation, err = tm.market.SubmitOrder(context.TODO(), orderSell2)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirmation.Trades))

	closed = tm.market.OnChainTimeUpdate(auctionEndTime)
	assert.False(t, closed)
	assert.Equal(t, int32(types.MarketState_MARKET_STATE_AUCTION), int32(tm.market.GetTradingMode()))
}
