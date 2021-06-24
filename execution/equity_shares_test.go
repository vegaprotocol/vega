package execution_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func TestEquityShares(t *testing.T) {
	t.Run("AverageEntryValuation", testAverageEntryValuation)
	t.Run("Shares", testShares)
	t.Run("WithinMarket", testWithinMarket)
}

// TestEquitySharesAverageEntryValuation is based on the spec example:
// https://github.com/vegaprotocol/product/blob/02af55e048a92a204e9ee7b7ae6b4475a198c7ff/specs/0042-setting-fees-and-rewarding-lps.md#calculating-liquidity-provider-equity-like-share
func testAverageEntryValuation(t *testing.T) {
	es := execution.NewEquityShares(num.DecimalFromFloat(100))

	es.SetPartyStake("LP1", *num.NewUint(100))
	require.EqualValues(t, num.DecimalFromFloat(100), es.AvgEntryValuation("LP1"))
	es.OpeningAuctionEnded()

	es.SetPartyStake("LP1", num.NewUint(200))
	require.True(t, num.DecimalFromFloat(100).Equal(es.AvgEntryValuation("LP1")))

	es.WithMVP(num.DecimalFromFloat(200)).SetPartyStake("LP2", num.NewUint(200))
	require.True(t, num.DecimalFromFloat(200).Equal(es.AvgEntryValuation("LP2")))
	require.True(t, num.DecimalFromFloat(100).Equal(es.AvgEntryValuation("LP1")))

	es.WithMVP(num.DecimalFromFloat(400)).SetPartyStake("LP1", num.NewUint(300))
	require.True(t, num.DecimalFromFloat(120).Equal(es.AvgEntryValuation("LP1")))

	es.SetPartyStake("LP1", num.NewUint(1))
	require.True(t, num.DecimalFromFloat(120).Equal(es.AvgEntryValuation("LP1")))
	require.True(t, num.DecimalFromFloat(200).Equal(es.AvgEntryValuation("LP2")))
}

func testShares(t *testing.T) {
	one, two, three := num.DecimalFromFloat(1), num.DecimalFromFloat(2), num.DecimalFromFloat(3)
	four, six := two.Mul(two), three.Mul(two)
	var (
		oneSixth    = one.Div(six)
		oneThird    = one.Div(three)
		oneFourth   = one.Div(four)
		threeFourth = three.Div(four)
		twoThirds   = two.Div(three)
		half        = one.Div(two)
	)

	es := execution.NewEquityShares(num.DecimalFromFloat(100))

	// Set LP1
	es.SetPartyStake("LP1", num.NewUint(100))
	t.Run("LP1", func(t *testing.T) {
		s := es.Shares(map[string]struct{}{})
		assert.True(t, one.Equal(s["LP1"]))
	})

	// Set LP2
	es.SetPartyStake("LP2", num.NewUint(200))
	t.Run("LP2", func(t *testing.T) {
		s := es.Shares(map[string]struct{}{})
		lp1, lp2 := s["LP1"], s["LP2"]

		assert.Equal(t, oneThird, lp1)
		assert.Equal(t, twoThirds, lp2)
		assert.True(t, one.Equal(lp1.Add(lp2)))
	})

	// Set LP3
	es.SetPartyStake("LP3", num.NewUint(300))
	t.Run("LP3", func(t *testing.T) {
		s := es.Shares(map[string]struct{}{})

		lp1, lp2, lp3 := s["LP1"], s["LP2"], s["LP3"]

		assert.Equal(t, oneSixth, lp1)
		assert.Equal(t, oneThird, lp2)
		assert.Equal(t, half, lp3)
		assert.True(t, one.Equal(lp1.Add(lp2).Add(lp3)))
	})

	// LP2 is undeployed
	t.Run("LP3", func(t *testing.T) {
		// pass LP as undeployed
		s := es.Shares(map[string]struct{}{"LP2": {}})

		lp1, lp3 := s["LP1"], s["LP3"]
		_, ok := s["LP2"]
		assert.False(t, ok)

		assert.Equal(t, oneFourth, lp1)
		// assert.Equal(t, oneThird, lp2)
		assert.Equal(t, threeFourth, lp3)
		assert.True(t, one.Equal(lp1.Add(lp3)))
	})
}

type equityShareMarket struct {
	t       *testing.T
	tm      *testMarket
	parties map[string]struct{}

	Now       time.Time
	ClosingAt time.Time
}

func newEquityShareMarket(t *testing.T) *equityShareMarket {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)

	return &equityShareMarket{
		t:         t,
		tm:        getTestMarket(t, now, closingAt, nil, nil),
		parties:   map[string]struct{}{},
		Now:       now,
		ClosingAt: closingAt,
	}
}

func (esm *equityShareMarket) TestMarket() *testMarket { return esm.tm }

func (esm *equityShareMarket) BuildOrder(id, party string, side types.Side, price uint64) *types.Order {
	return &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          id,
		Side:        side,
		PartyId:     party,
		MarketId:    esm.tm.market.GetID(),
		Size:        1,
		Price:       num.NewUint(price),
		Remaining:   1,
		CreatedAt:   esm.Now.UnixNano(),
		ExpiresAt:   esm.ClosingAt.UnixNano(),
	}
}

func (esm *equityShareMarket) createPartyIfMissing(party string) {
	if _, ok := esm.parties[party]; !ok {
		esm.parties[party] = struct{}{}
		addAccount(esm.tm, party)
	}
}

func (esm *equityShareMarket) SubmitOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, error) {
	esm.createPartyIfMissing(order.PartyId)
	return esm.tm.market.SubmitOrder(ctx, order)
}

func (esm *equityShareMarket) WithSubmittedOrder(id, party string, side types.Side, price uint64) *equityShareMarket {
	ctx := context.Background()
	order := esm.BuildOrder(id, party, side, price)

	_, err := esm.SubmitOrder(ctx, order)
	require.NoError(esm.t, err)
	return esm
}

func (esm *equityShareMarket) WithSubmittedLiquidityProvision(party, id string, amount uint64, fee string,
	buys, sells []*types.LiquidityOrder) *equityShareMarket {
	esm.createPartyIfMissing(party)
	esm.tm.WithSubmittedLiquidityProvision(esm.t, party, id, amount, fee, buys, sells)
	return esm
}

func (esm *equityShareMarket) LiquidityFeeAccount() *types.Account {
	acc, err := esm.tm.collateralEngine.GetMarketLiquidityFeeAccount(
		esm.tm.market.GetID(), esm.tm.asset,
	)
	require.NoError(esm.t, err)
	return acc
}

func (esm *equityShareMarket) PartyGeneralAccount(party string) *types.Account {
	acc, err := esm.tm.collateralEngine.GetPartyGeneralAccount(
		party, esm.tm.asset,
	)
	require.NoError(esm.t, err)
	return acc
}

func (esm *equityShareMarket) PartyMarginAccount(party string) *types.Account {
	acc, err := esm.tm.collateralEngine.GetPartyMarginAccount(
		esm.tm.market.GetID(), party, esm.tm.asset,
	)
	require.NoError(esm.t, err)
	return acc
}

func testWithinMarket(t *testing.T) {
	var (
		ctx = context.Background()
		// as we will split fees in 1/3 and 2/3
		// we use 900000 cause we need this number be divisible by 3
		matchingPrice = num.NewUint(900000)
		one           = num.NewUint(1)
	)

	// Setup a market with a set of non-matching orders and Liquidity Provision
	// Submissions from 2 parties.
	esm := newEquityShareMarket(t).
		WithSubmittedOrder("some-id-1", "party1", types.Side_SIDE_SELL, num.Sum(matchingPrice, one)).
		WithSubmittedOrder("some-id-2", "party2", types.Side_SIDE_BUY, num.NewUint(0).Sub(matchingPrice, one)).
		// party1 (commitment: 2000) should get 2/3 of the fee
		WithSubmittedLiquidityProvision("party1", "lp-id-1", 2000, "0.5",
			[]*types.LiquidityOrder{
				{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: -11},
			},
			[]*types.LiquidityOrder{
				{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 10},
			},
		).
		// party2 (commitment: 1000) should get 1/3 of the fee
		WithSubmittedLiquidityProvision("party2", "lp-id-2", 1000, "0.5",
			[]*types.LiquidityOrder{
				{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: -10},
			},
			[]*types.LiquidityOrder{
				{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 11},
			},
		)

	// tm is the testMarket instance
	var (
		tm      = esm.TestMarket()
		curTime = esm.Now
	)

	t.Run("WhenNoTrades", func(t *testing.T) {
		// clean up previous events
		tm.events = []events.Event{}

		// Trigger Fee distribution
		curTime = curTime.Add(1 * time.Second)
		tm.market.OnChainTimeUpdate(ctx, curTime)

		// Assert the event
		var evt *events.TransferResponse
		for _, e := range tm.events {
			if e.Type() == events.TransferResponses {
				evt = e.(*events.TransferResponse)
			}
		}
		require.Nil(t, evt, "should receive no TransferEvent")
	})

	// Match a pair of orders (same price) to trigger a fee distribution.
	conf, err := esm.
		WithSubmittedOrder("some-id-3", "party1", types.Side_SIDE_SELL, matchingPrice).
		SubmitOrder(
			context.Background(),
			esm.BuildOrder("some-id-4", "party2", types.Side_SIDE_BUY, matchingPrice),
		)
	require.NoError(t, err)
	require.Len(t, conf.Trades, 1)

	// Retrieve both MarketLiquidityFee account balance and Party Balance
	// before the fee distribution.
	var (
		originalBalance = esm.LiquidityFeeAccount().Balance
		party1Balance   = esm.PartyMarginAccount("party1").Balance
		party2Balance   = esm.PartyMarginAccount("party2").Balance
	)

	curTime = curTime.Add(1 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, curTime)

	assert.Zero(t, esm.LiquidityFeeAccount().Balance,
		"LiquidityFeeAccount should be empty after a fee distribution")

	assert.EqualValues(t,
		int(float64(originalBalance)*(2./3)),
		int(esm.PartyMarginAccount("party1").Balance-party1Balance),
		"party1 should get 2/3 of the fees",
	)

	assert.EqualValues(t,
		int(float64(originalBalance)*(1./3)),
		int(esm.PartyMarginAccount("party2").Balance-party2Balance),
		"party2 should get 2/3 of the fees",
	)
}
