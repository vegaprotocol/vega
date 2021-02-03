package execution_test

import (
	"context"
	"fmt"
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

func TestIssue2876(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{Duration: 30})
	ctx := context.Background()

	tm.market.OnChainTimeUpdate(ctx, now)

	addAccountWithAmount(tm, "trader-0", 100000000)
	addAccountWithAmount(tm, "trader-1", 100000000)
	addAccountWithAmount(tm, "trader-2", 100000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(5)

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, "Order01", types.Side_SIDE_BUY, "trader-0", 20, 3500)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, "Order02", types.Side_SIDE_SELL, "trader-1", 20, 4000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, "Order03", types.Side_SIDE_BUY, "trader-2", 10, 5500)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, "Order04", types.Side_SIDE_SELL, "trader-2", 10, 5000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	lporder := types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: 1000000,
		Fee:              "0.01",
		Buys: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_BEST_BID,
				Proportion: 10,
				Offset:     -1000,
			},
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_MID,
				Proportion: 13,
				Offset:     -1500,
			},
		},
		Sells: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
				Proportion: 10,
				Offset:     2000,
			},
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
				Proportion: 13,
				Offset:     1000,
			},
		},
	}

	err = tm.market.SubmitLiquidityProvision(ctx, &lporder, "trader-2", "lp-order-01")
	assert.NoError(t, err)

	bondAccount, err := tm.collateraEngine.GetOrCreatePartyBondAccount(ctx, "trader-2", tm.market.GetID(), tm.asset)
	assert.NoError(t, err)
	// we expect the whole commitment to be there
	assert.Equal(t, 1000000, int(bondAccount.Balance))

	// but also some margin to cover the orders
	marginAccount, err := tm.collateraEngine.GetPartyMarginAccount(tm.market.GetID(), "trader-2", tm.asset)
	assert.NoError(t, err)
	assert.Equal(t, 15000, int(marginAccount.Balance))

	// but also some funds left in the genearal
	generalAccount, err := tm.collateraEngine.GetPartyGeneralAccount("trader-2", tm.asset)
	assert.NoError(t, err)
	assert.Equal(t, 98985000, int(generalAccount.Balance))

	// now let's move time and see
	// this should end the opening auction
	now = now.Add(31 * time.Second)

	tm.market.OnChainTimeUpdate(ctx, now)

	bondAccount, err = tm.collateraEngine.GetOrCreatePartyBondAccount(ctx, "trader-2", tm.market.GetID(), tm.asset)
	assert.NoError(t, err)
	// we expect the whole commitment to be there
	assert.Equal(t, 1000000, int(bondAccount.Balance))

	// but also some margin to cover the orders
	marginAccount, err = tm.collateraEngine.GetPartyMarginAccount(tm.market.GetID(), "trader-2", tm.asset)
	assert.NoError(t, err)
	assert.Equal(t, 13219725, int(marginAccount.Balance))

	// but also some funds left in the genearal
	generalAccount, err = tm.collateraEngine.GetPartyGeneralAccount("trader-2", tm.asset)
	assert.NoError(t, err)
	assert.Equal(t, 85780275, int(generalAccount.Balance))

	assert.Equal(t, tm.market.GetPeggedOrderCount(), 4)

}

func TestIssue2876_NewGetMarket(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarketNoBrokerExpect(t, now, closingAt, nil, &types.AuctionDuration{Duration: 30}, true)
	defer tm.ctrl.Finish()

	tm.broker.EXPECT().Send(gomock.Any()).Times(2)
	ctx := context.Background()
	// update time
	// 2 calls to the mock
	tm.market.OnChainTimeUpdate(ctx, now)
	// make accounts
	addAccountWithAmount(tm, "trader-0", 100000000)
	addAccountWithAmount(tm, "trader-1", 100000000)
	addAccountWithAmount(tm, "trader-2", 100000000)
	// update stake factors
	tm.market.OnSuppliedStakeToObligationFactorUpdate(5)

	tm.broker.EXPECT().Send(gomock.Any()).Times(6)
	tm.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, "Order01", types.Side_SIDE_BUY, "trader-0", 20, 3500)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	tm.broker.EXPECT().Send(gomock.Any()).Times(6)
	tm.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, "Order02", types.Side_SIDE_SELL, "trader-1", 20, 4000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	tm.broker.EXPECT().Send(gomock.Any()).Times(6)
	tm.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, "Order03", types.Side_SIDE_BUY, "trader-2", 10, 5500)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	tm.broker.EXPECT().Send(gomock.Any()).Times(5)
	tm.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, "Order04", types.Side_SIDE_SELL, "trader-2", 10, 5000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		fmt.Printf("EVENT(%v): %#v\n", evt.Type(), evt)
	})
	tm.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(func(evts []events.Event) {
		for _, evt := range evts {
			fmt.Printf("ALLEVENT(%v): %#v\n", evt.Type(), evt)
		}
	})

	lporder := types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: 1000000,
		Fee:              "0.01",
		Buys: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_BEST_BID,
				Proportion: 10,
				Offset:     -1000,
			},
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_MID,
				Proportion: 13,
				Offset:     -1500,
			},
		},
		Sells: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
				Proportion: 10,
				Offset:     2000,
			},
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
				Proportion: 13,
				Offset:     1000,
			},
		},
	}

	err = tm.market.SubmitLiquidityProvision(ctx, &lporder, "trader-2", "lp-order-01")
	assert.NoError(t, err)

	bondAccount, err := tm.collateraEngine.GetOrCreatePartyBondAccount(ctx, "trader-2", tm.market.GetID(), tm.asset)
	assert.NoError(t, err)
	// we expect the whole commitment to be there
	assert.Equal(t, 1000000, int(bondAccount.Balance))

	// but also some margin to cover the orders
	marginAccount, err := tm.collateraEngine.GetPartyMarginAccount(tm.market.GetID(), "trader-2", tm.asset)
	assert.NoError(t, err)
	assert.Equal(t, 15000, int(marginAccount.Balance))

	// but also some funds left in the genearal
	generalAccount, err := tm.collateraEngine.GetPartyGeneralAccount("trader-2", tm.asset)
	assert.NoError(t, err)
	assert.Equal(t, 98985000, int(generalAccount.Balance))

	// now let's move time and see
	// this should end the opening auction
	now = now.Add(31 * time.Second)

	tm.market.OnChainTimeUpdate(ctx, now)

	bondAccount, err = tm.collateraEngine.GetOrCreatePartyBondAccount(ctx, "trader-2", tm.market.GetID(), tm.asset)
	assert.NoError(t, err)
	// we expect the whole commitment to be there
	assert.Equal(t, 1000000, int(bondAccount.Balance))

	// but also some margin to cover the orders
	marginAccount, err = tm.collateraEngine.GetPartyMarginAccount(tm.market.GetID(), "trader-2", tm.asset)
	assert.NoError(t, err)
	assert.Equal(t, 13219725, int(marginAccount.Balance))

	// but also some funds left in the genearal
	generalAccount, err = tm.collateraEngine.GetPartyGeneralAccount("trader-2", tm.asset)
	assert.NoError(t, err)
	assert.Equal(t, 85780275, int(generalAccount.Balance))

	assert.Equal(t, tm.market.GetPeggedOrderCount(), 4)
}

func getTestMarketNoBrokerExpect(t *testing.T, now time.Time, closingAt time.Time, pMonitorSettings *types.PriceMonitoringSettings, openingAuctionDuration *types.AuctionDuration, startOpeninAuction bool) *testMarket {
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	riskConfig := risk.NewDefaultConfig()
	positionConfig := positions.NewDefaultConfig()
	settlementConfig := settlement.NewDefaultConfig()
	matchingConfig := matching.NewDefaultConfig()
	feeConfig := fee.NewDefaultConfig()
	broker := mocks.NewMockBroker(ctrl)

	// add expectation of broket for this setup
	broker.EXPECT().Send(gomock.Any()).Times(10)

	tm := &testMarket{
		log:    log,
		ctrl:   ctrl,
		broker: broker,
		now:    now,
	}

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

	mkts := getMarkets(closingAt, pMonitorSettings, openingAuctionDuration)

	mktCfg := &mkts[0]

	mas := monitor.NewAuctionState(mktCfg, now)
	mktEngine, err := execution.NewMarket(context.Background(),
		log, riskConfig, positionConfig, settlementConfig, matchingConfig,
		feeConfig, collateralEngine, mktCfg, now, broker, execution.NewIDGen(), mas)
	assert.NoError(t, err)

	if startOpeninAuction {
		mktEngine.StartOpeningAuction(context.Background())
	}

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
