package execution_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMarketStates(t *testing.T) {
	t.Run("test initial state is PROPOSED", testInitialStateIsProposed)
	t.Run("cannot do order stuff in PROPOSED state", testCannotDoOrderStuffInProposedState)
	t.Run("can move from PROPOSED to REJECTED state", testCanMoveFromProposedToRejectedState)
	t.Run("can move from PROPOSED to PENDING state", testCanMoveFromProposedToPendingState)
	t.Run("can move from PENDING to ACTIVE state", testCanMoveFromPendingToActiveState)
	t.Run("can place order in PENDING state", testCanPlaceOrderInActiveState)
}

func testInitialStateIsProposed(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(2000, 0)
	auctionDuration := &types.AuctionDuration{
		Duration: 30, // seconds
	}
	tm := getTestMarket2(t, now, closingAt, nil, auctionDuration, false)
	defer tm.ctrl.Finish()

	assert.Equal(t, types.MarketStateProposed, tm.market.State())
}

func testCannotDoOrderStuffInProposedState(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(2000, 0)
	auctionDuration := &types.AuctionDuration{
		Duration: 30, // seconds
	}
	ctx := context.Background()

	tm := getTestMarket2(t, now, closingAt, nil, auctionDuration, false)
	defer tm.ctrl.Finish()
	assert.Equal(t, types.MarketStateProposed, tm.market.State())

	addAccountWithAmount(tm, "someparty", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// expect error
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-A", 5, 5000)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	assert.Nil(t, o1conf)
	assert.EqualError(t, err, execution.ErrTradingNotAllowed.Error())

	o2conf, err := tm.market.CancelAllOrders(ctx, "someparty")
	assert.Nil(t, o2conf)
	assert.EqualError(t, err, execution.ErrTradingNotAllowed.Error())

	o3conf, err := tm.market.CancelOrder(ctx, "someparty", "someorder")
	assert.Nil(t, o3conf)
	assert.EqualError(t, err, execution.ErrTradingNotAllowed.Error())

	amendment := &types.OrderAmendment{
		OrderID:   o1.ID,
		Price:     num.NewUint(4000),
		SizeDelta: 10,
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, "party-A")
	assert.Nil(t, amendConf)
	assert.EqualError(t, err, execution.ErrTradingNotAllowed.Error())

	// but can place liquidity submission
	lpsub := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1),
		Fee:              num.DecimalFromFloat(0.1),
		Sells: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceBestAsk,
				Proportion: 1,
				Offset:     1,
			},
		},
		Buys: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceMid,
				Proportion: 1,
				Offset:     -1,
			},
		},
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lpsub, "someparty", "lpid1")

	// we expect an error as this lp may be stupid
	// but not equal to the trading not allowed one
	assert.NoError(t, err)
}

func testCanMoveFromProposedToRejectedState(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(2000, 0)
	auctionDuration := &types.AuctionDuration{
		Duration: 30, // seconds
	}
	tm := getTestMarket2(t, now, closingAt, nil, auctionDuration, false)
	defer tm.ctrl.Finish()

	assert.Equal(t, types.MarketStateProposed, tm.market.State())

	err := tm.market.Reject(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, types.MarketStateRejected, tm.market.State())
}

func testCanMoveFromProposedToPendingState(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(2000, 0)
	auctionDuration := &types.AuctionDuration{
		Duration: 30, // seconds
	}
	tm := getTestMarket2(t, now, closingAt, nil, auctionDuration, false)
	defer tm.ctrl.Finish()

	assert.Equal(t, types.MarketStateProposed, tm.market.State())

	err := tm.market.StartOpeningAuction(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, types.MarketStatePending, tm.market.State())
}

func testCanMoveFromPendingToActiveState(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(2000, 0)
	auctionDuration := &types.AuctionDuration{
		Duration: 30, // seconds
	}
	tm := getTestMarket2(t, now, closingAt, nil, auctionDuration, false)
	defer tm.ctrl.Finish()

	assert.Equal(t, types.MarketStateProposed, tm.market.State())

	err := tm.market.StartOpeningAuction(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, types.MarketStatePending, tm.market.State())

	addAccountWithAmount(tm, "party1", 100000000)
	addAccountWithAmount(tm, "party2", 100000000)
	addAccountWithAmount(tm, "party3", 100000000)
	addAccountWithAmount(tm, "party4", 100000000)
	orders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "order1", types.SideBuy, "party1", 1, 5000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "order2", types.SideSell, "party2", 1, 5000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "order3", types.SideBuy, "party3", 1, 4500),  // buy too low
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "order4", types.SideSell, "party4", 1, 5500), // sell too expensive
	}
	for _, o := range orders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		assert.NotNil(t, conf)
		assert.NoError(t, err)
	}
	// now move to after the opening auction time
	tm.market.OnChainTimeUpdate(context.Background(), now.Add(40*time.Second))
	assert.Equal(t, types.MarketStateActive, tm.market.State())
}

func testCanPlaceOrderInActiveState(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(2000, 0)
	auctionDuration := &types.AuctionDuration{
		Duration: 30, // seconds
	}
	tm := getTestMarket2(t, now, closingAt, nil, auctionDuration, false)
	defer tm.ctrl.Finish()

	assert.Equal(t, types.MarketStateProposed, tm.market.State())

	err := tm.market.StartOpeningAuction(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, types.MarketStatePending, tm.market.State())

	addAccountWithAmount(tm, "party1", 100000000)
	addAccountWithAmount(tm, "party2", 100000000)
	addAccountWithAmount(tm, "party3", 100000000)
	addAccountWithAmount(tm, "party4", 100000000)
	orders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "order1", types.SideBuy, "party1", 1, 5000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "order2", types.SideSell, "party2", 1, 5000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "order3", types.SideBuy, "party3", 1, 4500),  // buy too low
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "order4", types.SideSell, "party4", 1, 5500), // sell too expensive
	}
	for _, o := range orders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		assert.NotNil(t, conf)
		assert.NoError(t, err)
	}
	// now move to after the opening auction time
	tm.market.OnChainTimeUpdate(context.Background(), now.Add(40*time.Second))
	assert.Equal(t, types.MarketStateActive, tm.market.State())

	addAccountWithAmount(tm, "someparty", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// expect error
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "someparty", 5, 5000)
	o1conf, err := tm.market.SubmitOrder(context.Background(), o1)
	assert.NotNil(t, o1conf)
	assert.NoError(t, err)
}
