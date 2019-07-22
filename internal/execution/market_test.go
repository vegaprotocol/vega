package execution_test

import (
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/collateral"
	collateralmocks "code.vegaprotocol.io/vega/internal/collateral/mocks"
	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/execution/mocks"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/matching"
	"code.vegaprotocol.io/vega/internal/positions"
	"code.vegaprotocol.io/vega/internal/risk"
	"code.vegaprotocol.io/vega/internal/settlement"
	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type testMarket struct {
	market     *execution.Market
	log        *logging.Logger
	ctrl       *gomock.Controller
	accountBuf *collateralmocks.MockAccountBuffer

	collateraEngine *collateral.Engine
	partyEngine     *execution.Party
	candleStore     *mocks.MockCandleStore
	orderStore      *mocks.MockOrderStore
	partyStore      *mocks.MockPartyStore
	tradeStore      *mocks.MockTradeStore

	now time.Time
}

func getTestMarket(t *testing.T, now time.Time, closingAt time.Time) *testMarket {
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	riskConfig := risk.NewDefaultConfig()
	positionConfig := positions.NewDefaultConfig()
	settlementConfig := settlement.NewDefaultConfig()
	matchingConfig := matching.NewDefaultConfig()

	candleStore := mocks.NewMockCandleStore(ctrl)
	candleStore.EXPECT().FetchLastCandle(gomock.Any(), gomock.Any()).Return(nil, errors.New("some error")).AnyTimes()
	orderStore := mocks.NewMockOrderStore(ctrl)
	partyStore := mocks.NewMockPartyStore(ctrl)
	tradeStore := mocks.NewMockTradeStore(ctrl)

	accountBuf := collateralmocks.NewMockAccountBuffer(ctrl)
	collateralEngine, err := collateral.New(log, collateral.NewDefaultConfig(), accountBuf, now)
	assert.Nil(t, err)
	mkts := getMarkets(closingAt)
	partyEngine := execution.NewParty(log, collateralEngine, mkts, partyStore)

	mktEngine, err := execution.NewMarket(
		log, riskConfig, positionConfig, settlementConfig, matchingConfig,
		collateralEngine, partyEngine, &mkts[0], candleStore, orderStore,
		partyStore, tradeStore, now, 0,
	)

	asset, err := mkts[0].GetAsset()
	assert.Nil(t, err)

	accountBuf.EXPECT().Add(gomock.Any()).Times(4)
	// ignore response ids here + this cannot fail
	_, _ = collateralEngine.CreateMarketAccounts(mktEngine.GetID(), asset, 0)

	return &testMarket{
		market:          mktEngine,
		log:             log,
		ctrl:            ctrl,
		accountBuf:      accountBuf,
		collateraEngine: collateralEngine,
		partyEngine:     partyEngine,
		candleStore:     candleStore,
		orderStore:      orderStore,
		partyStore:      partyStore,
		tradeStore:      tradeStore,

		now: now,
	}
}

func getMarkets(closingAt time.Time) []proto.Market {
	mkt := proto.Market{
		Name: "ETHUSD/DEC19",
		TradableInstrument: &proto.TradableInstrument{
			Instrument: &proto.Instrument{
				Id:        "Crypto/ETHUSD/Futures/Dec19",
				Code:      "CRYPTO:ETHUSD/DEC19",
				Name:      "December 2019 ETH vs USD future",
				BaseName:  "ETH",
				QuoteName: "USD",
				Metadata: &proto.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				Product: &proto.Instrument_Future{
					Future: &proto.Future{
						Maturity: closingAt.Format(time.RFC3339),
						Oracle: &proto.Future_EthereumEvent{
							EthereumEvent: &proto.EthereumEvent{
								ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
								Event:      "price_changed",
							},
						},
						Asset: "ETH",
					},
				},
			},
			RiskModel: &proto.TradableInstrument_Forward{
				Forward: &proto.Forward{
					Lambd: 0.01,
					Tau:   1.0 / 365.25 / 24,
					Params: &proto.ModelParamsBS{
						Mu:    0,
						R:     0.016,
						Sigma: 0.09,
					},
				},
			},
		},
		TradingMode: &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{},
		},
	}

	execution.SetMarketID(&mkt, 0)
	return []proto.Market{mkt}
}

func TestMarketClosing(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(20, 0)
	tm := getTestMarket(t, now, closingAt)
	defer tm.ctrl.Finish()
	// add 2 traders to the party engine
	// this will create 2 traders, credit their account
	// and move some monies to the market
	tm.accountBuf.EXPECT().Add(gomock.Any()).Times(10)
	tm.partyStore.EXPECT().Post(gomock.Any()).Times(2).Return(nil)
	tm.partyEngine.NotifyTraderAccount(&types.NotifyTraderAccount{TraderID: party1})
	tm.partyEngine.NotifyTraderAccount(&types.NotifyTraderAccount{TraderID: party2})

	tm.candleStore.EXPECT().GenerateCandlesFromBuffer(gomock.Any(), gomock.Any()).Times(1).Return(nil)
	// check account gets updated
	tm.accountBuf.EXPECT().Add(gomock.Any()).Times(2).DoAndReturn(func(acc types.Account) {
		// if Margin -> 0
		if acc.Type == types.AccountType_MARGIN {
			assert.Equal(t, acc.Balance, int64(0))
		}
		// if general, is should be back to the original topup as no
		// trade happend
		if acc.Type == types.AccountType_GENERAL {
			assert.Equal(t, acc.Balance, int64(1000000000000))
		}
	})
	closed := tm.market.OnChainTimeUpdate(closingAt.Add(1 * time.Second))
	assert.True(t, closed)
}

func TestMarketWithTradeClosing(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(20, 0)
	tm := getTestMarket(t, now, closingAt)
	defer tm.ctrl.Finish()
	// add 2 traders to the party engine
	// this will create 2 traders, credit their account
	// and move some monies to the market
	tm.accountBuf.EXPECT().Add(gomock.Any()).Times(8)
	tm.partyStore.EXPECT().Post(gomock.Any()).Times(2).Return(nil)
	tm.partyEngine.NotifyTraderAccount(&types.NotifyTraderAccount{TraderID: party1})
	tm.partyEngine.NotifyTraderAccount(&types.NotifyTraderAccount{TraderID: party2})

	// submit orders
	// party1 buys
	// party2 sells
	orderBuy := &types.Order{
		Type:      types.Order_GTT,
		Status:    types.Order_Active,
		Id:        "",
		Side:      types.Side_Buy,
		PartyID:   party1,
		MarketID:  tm.market.GetID(),
		Size:      100,
		Price:     100,
		Remaining: 100,
		CreatedAt: now.UnixNano(),
		ExpiresAt: closingAt.UnixNano(),
		Reference: "party1-buy-order",
	}
	orderSell := &types.Order{
		Type:      types.Order_GTT,
		Status:    types.Order_Active,
		Id:        "",
		Side:      types.Side_Sell,
		PartyID:   party2,
		MarketID:  tm.market.GetID(),
		Size:      100,
		Price:     100,
		Remaining: 100,
		CreatedAt: now.UnixNano(),
		ExpiresAt: closingAt.UnixNano(),
		Reference: "party2-sell-order",
	}

	// submit orders
	tm.partyStore.EXPECT().GetByID(gomock.Any()).AnyTimes().DoAndReturn(func(id string) (*types.Party, error) {
		return &types.Party{Id: id}, nil
	})
	tm.partyStore.EXPECT().Post(gomock.Any()).AnyTimes().Return(nil)
	tm.orderStore.EXPECT().Post(gomock.Any()).AnyTimes().Return(nil)
	tm.orderStore.EXPECT().Put(gomock.Any()).AnyTimes().Return(nil)
	tm.tradeStore.EXPECT().Post(gomock.Any()).AnyTimes().Return(nil)

	// close the market now
	// check account gets updated
	// tm.accountBuf.EXPECT().Add(gomock.Any()).Times(2)

	_, err := tm.market.SubmitOrder(orderSell)
	assert.Nil(t, err)
	_, err = tm.market.SubmitOrder(orderBuy)
	assert.Nil(t, err)

	tm.candleStore.EXPECT().GenerateCandlesFromBuffer(gomock.Any(), gomock.Any()).Times(1).Return(nil)

	tm.accountBuf.EXPECT().Add(gomock.Any()).Times(9).DoAndReturn(func(acc types.Account) {

		fmt.Printf("ACCOUNT: %v\n", acc)
		// if Margin -> 0
		if acc.Type == types.AccountType_MARGIN {
			assert.Equal(t, acc.Balance, int64(0))
		}
		// if general, is should be back to the original topup as no
		// trade happend
		if acc.Type == types.AccountType_GENERAL {
			assert.Equal(t, acc.Balance, int64(1000000000000))
		}
	})

	// update collateral time first, normally done by execution engin
	futureTime := closingAt.Add(1 * time.Second)
	tm.collateraEngine.OnChainTimeUpdate(futureTime)
	closed := tm.market.OnChainTimeUpdate(futureTime)
	assert.True(t, closed)
}

func TestSetMarketID(t *testing.T) {
	t.Run("nil market config", func(t *testing.T) {
		marketcfg := &proto.Market{}
		err := execution.SetMarketID(marketcfg, 0)
		assert.Error(t, err)
	})

	t.Run("good market config", func(t *testing.T) {
		marketcfg := &proto.Market{
			Id:   "", // ID will be generated
			Name: "ETH/DEC19",
			TradableInstrument: &proto.TradableInstrument{
				Instrument: &proto.Instrument{
					Id:   "Crypto/ETHUSD/Futures/Dec19",
					Code: "FX:ETHUSD/DEC19",
					Name: "December 2019 ETH vs USD future",
					Metadata: &proto.InstrumentMetadata{
						Tags: []string{
							"asset_class:fx/crypto",
							"product:futures",
						},
					},
					Product: &proto.Instrument_Future{
						Future: &proto.Future{
							Maturity: "2019-12-31T23:59:59Z",
							Oracle: &proto.Future_EthereumEvent{
								EthereumEvent: &proto.EthereumEvent{
									ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
									Event:      "price_changed",
								},
							},
							Asset: "Ethereum/Ether",
						},
					},
				},
				RiskModel: &proto.TradableInstrument_Forward{
					Forward: &proto.Forward{
						Lambd: 0.01,
						Tau:   1.0 / 365.25 / 24,
						Params: &proto.ModelParamsBS{
							Mu:    0,
							R:     0.016,
							Sigma: 0.09,
						},
					},
				},
			},
			TradingMode: &proto.Market_Continuous{
				Continuous: &proto.ContinuousTrading{},
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
