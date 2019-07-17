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

	"github.com/golang/mock/gomock"
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

func getTestMarket(t *testing.T) *testMarket {
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	riskConfig := risk.NewDefaultConfig()
	positionConfig := positions.NewDefaultConfig()
	settlementConfig := settlement.NewDefaultConfig()
	matchingConfig := matching.NewDefaultConfig()

	candleStore := mocks.NewMockCandleStore(ctrl)
	orderStore := mocks.NewMockOrderStore(ctrl)
	partyStore := mocks.NewMockPartyStore(ctrl)
	tradeStore := mocks.NewMockTradeStore(ctrl)

	accountBuf := collateralmocks.NewMockAccountBuffer(ctrl)
	now := time.Now()
	collateralEngine, err := collateral.New(log, collateral.NewDefaultConfig(), accountBuf, now)
	assert.Nil(t, err)
	mkts := getMarkets()
	partyEngine := execution.NewParty(log, collateralEngine, mkts, partyStore)

	mktEngine, err := execution.NewMarket(
		log, riskConfig, positionConfig, settlementConfig, matchingConfig,
		collateralEngine, partyEngine, &mkts[0], candleStore, orderStore,
		partyStore, tradeStore, now, 0,
	)

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

func getMarkets() []proto.Market {
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
						Maturity: "2019-07-16T10:17:00Z",
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

func addTrader(t *testing.T, partyEngine *execution.Party, party string) {

}

func TestMarketClosing(t *testing.T) {
	_ = getTestMarket(t)

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
