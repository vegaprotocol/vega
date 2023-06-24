package spot_test

import (
	"context"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/execution/spot"
	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/monitor"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
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
			Quantum: num.DecimalZero(),
		},
	},
	{
		ID: "BTC",
		Details: &types.AssetDetails{
			Symbol:  "BTC",
			Quantum: num.DecimalZero(),
		},
	},
	{
		ID: "VOTE",
		Details: &types.AssetDetails{
			Name:     "VOTE",
			Symbol:   "VOTE",
			Decimals: 5,
			Quantum:  num.DecimalZero(),
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
			TriggeringRatio: num.DecimalZero(),
		},
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

	as := monitor.NewAuctionState(&mkt, now)
	epoch := mocks.NewMockEpochEngine(ctrl)
	epoch.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).Times(1)

	mat := common.NewMarketActivityTracker(log, epoch)

	baseAsset := NewAssetStub(base, baseDP)
	quoteAsset := NewAssetStub(quote, quoteDP)
	market, _ := spot.NewMarket(
		context.Background(),
		log,
		matching.NewDefaultConfig(),
		fee.NewDefaultConfig(),
		collateral,
		&mkt,
		ts,
		broker,
		as,
		statevarEngine,
		mat,
		baseAsset,
		quoteAsset,
		peggedOrderCounterForTest,
	)

	tm := &testMarket{
		market:           market,
		log:              log,
		ctrl:             ctrl,
		broker:           broker,
		timeService:      ts,
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
