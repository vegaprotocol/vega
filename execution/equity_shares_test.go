package execution_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
)

func TestEquityShares(t *testing.T) {
	t.Run("AverageEntryValuation", testAverageEntryValuation)
	t.Run("Shares", testShares)
	t.Run("WithinMarket", testWithinMarket)
}

// TestEquitySharesAverageEntryValuation is based on the spec example:
// https://github.com/vegaprotocol/product/blob/02af55e048a92a204e9ee7b7ae6b4475a198c7ff/specs/0042-setting-fees-and-rewarding-lps.md#calculating-liquidity-provider-equity-like-share
func testAverageEntryValuation(t *testing.T) {
	es := execution.NewEquityShares(100)

	es.SetPartyStake("LP1", 100)
	require.EqualValues(t, 100, es.AvgEntryValuation("LP1"))

	es.SetPartyStake("LP1", 200)
	require.EqualValues(t, 100, es.AvgEntryValuation("LP1"))

	es.WithMVP(200).SetPartyStake("LP2", 200)
	require.EqualValues(t, 200, es.AvgEntryValuation("LP2"))
	require.EqualValues(t, 100, es.AvgEntryValuation("LP1"))

	es.WithMVP(400).SetPartyStake("LP1", 300)
	require.EqualValues(t, 120, es.AvgEntryValuation("LP1"))

	es.SetPartyStake("LP1", 1)
	require.EqualValues(t, 120, es.AvgEntryValuation("LP1"))
	require.EqualValues(t, 200, es.AvgEntryValuation("LP2"))
}

func testShares(t *testing.T) {
	var (
		oneSixth  = 1.0 / 6
		oneThird  = 1.0 / 3
		twoThirds = 2.0 / 3
		half      = 1.0 / 2
	)

	es := execution.NewEquityShares(100)

	// Set LP1
	es.SetPartyStake("LP1", 100)
	t.Run("LP1", func(t *testing.T) {
		s := es.Shares()
		assert.Equal(t, 1.0, s["LP1"])
	})

	// Set LP2
	es.SetPartyStake("LP2", 200)
	t.Run("LP2", func(t *testing.T) {
		s := es.Shares()
		lp1, lp2 := s["LP1"], s["LP2"]

		assert.Equal(t, oneThird, lp1)
		assert.Equal(t, twoThirds, lp2)
		assert.Equal(t, 1.0, lp1+lp2)
	})

	// Set LP3
	es.SetPartyStake("LP3", 300)
	t.Run("LP3", func(t *testing.T) {
		s := es.Shares()

		lp1, lp2, lp3 := s["LP1"], s["LP2"], s["LP3"]

		assert.Equal(t, oneSixth, lp1)
		assert.Equal(t, oneThird, lp2)
		assert.Equal(t, half, lp3)
		assert.Equal(t, 1.0, lp1+lp2+lp3)
	})
}

func testWithinMarket(t *testing.T) {
	ctx := context.Background()
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	var matchingPrice uint64 = 111

	addAccount(tm, party1)
	addAccount(tm, party2)

	//TODO (WG 07/01/21): Currently limit orders need to be present on order book for liquidity provision submission to work, remove once fixed.
	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid2",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
		Size:        1,
		Price:       matchingPrice + 1,
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   closingAt.UnixNano(),
		Reference:   "party2-sell-order-1",
	}
	confirmationSell, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid1",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
		MarketID:         tm.market.GetID(),
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

	// clean up previous events
	tm.events = []events.Event{}
	tick := now.Add(1 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, tick)

	// Assert the event
	var evt *events.TransferResponse
	for _, e := range tm.events {
		if e.Type() == events.TransferResponses {
			evt = e.(*events.TransferResponse)
		}
	}
	require.NotNil(t, evt, "a TransferResponse event should have been emitted")
	require.Len(t, evt.TransferResponses(), 1, "the event should contain 1 TransferResponse")
}
