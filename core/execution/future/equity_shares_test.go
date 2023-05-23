// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package future_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	vegacontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/future"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
)

// just a convenience type used in some tests.
type lpdata struct {
	id  string
	amt *num.Uint
	avg num.Decimal
}

func TestEquityShares(t *testing.T) {
	t.Run("AvgEntryValuation with trade value", testAvgEntryValuationGrowth)
	t.Run("SharesExcept", testShares)
	t.Run("WithinMarket", testWithinMarket)
	t.Run("Average entry valuation after 6063 spec change", testAvgEntryUpdate)
}

// replicate the example given in spec file (protocol/0042-LIQF-setting_fees_and_rewarding_lps.md).
func testAvgEntryUpdate(t *testing.T) {
	es := future.NewEquityShares(num.DecimalZero())
	es.OpeningAuctionEnded()
	initial := lpdata{
		id:  "initial",
		amt: num.NewUint(900),
		avg: num.DecimalFromFloat(900),
	}
	es.SetPartyStake(initial.id, initial.amt)
	require.True(t, initial.avg.Equals(es.AvgEntryValuation(initial.id)), es.AvgEntryValuation(initial.id).String())
	// step 1 from the example: LP commitment of 100 with an existing commitment of 1k:
	step1 := lpdata{
		id:  "step1",
		amt: num.NewUint(100),
		avg: num.NewDecimalFromFloat(1000),
	}
	es.SetPartyStake(step1.id, step1.amt)
	require.True(t, step1.avg.Equals(es.AvgEntryValuation(step1.id)), es.AvgEntryValuation(step1.id).String())
	// get sum of all vStake to 2K as per example in the spec
	// total vStake == 1.1k => avg is now 1.1k
	inc := lpdata{
		id:  "topup",
		amt: num.NewUint(990),
		avg: num.DecimalFromFloat(1990),
	}
	es.SetPartyStake(inc.id, inc.amt)
	require.True(t, inc.avg.Equals(es.AvgEntryValuation(inc.id)), es.AvgEntryValuation(inc.id).String())
	// Example 2: We have a total vStake of 2k -> step1 party increases the commitment amount to 110 (so +10)
	step1.amt = num.NewUint(110)
	step1.avg, _ = num.DecimalFromString("1090.9090909090909091")
	es.SetPartyStake(step1.id, step1.amt)
	require.True(t, step1.avg.Equals(es.AvgEntryValuation(step1.id)), es.AvgEntryValuation(step1.id).String())
	// increase total vStake to be 3k using a new LP party
	testAvgEntryUpdateStep3New(t, es)
	// example 3 when total vStake is 3k -> decrease commitment by 20
	step1.amt = num.NewUint(90)
	es.SetPartyStake(step1.id, step1.amt)
	require.True(t, step1.avg.Equals(es.AvgEntryValuation(step1.id)), es.AvgEntryValuation(step1.id).String())
	// set up example 3 again, this time by increasing the commitment of an existing party, their AEV should be updated accordingly
	testAvgEntryUpdateStep3Add(t, es, &initial)
	step1.amt = num.NewUint(70) // decrease by another 20
	es.SetPartyStake(step1.id, step1.amt)
	require.True(t, step1.avg.Equals(es.AvgEntryValuation(step1.id)), es.AvgEntryValuation(step1.id).String())
	// set up for example 3, this time by increasing the total vStake to 3k using growth
	testAvgEntryUpdateStep3Growth(t, es)
	step1.amt = num.NewUint(50) // decrease by another 20
	es.SetPartyStake(step1.id, step1.amt)
	require.True(t, step1.avg.Equals(es.AvgEntryValuation(step1.id)), es.AvgEntryValuation(step1.id).String())
}

// continue based on testAvgEntryUpdate setup, just add new LP to get the total up to 3k.
func testAvgEntryUpdateStep3New(t *testing.T, es *future.EquityShares) {
	t.Helper()
	// we have 1000 + 110 + 990 (2000)
	// AEV == 2000
	inc := lpdata{
		id:  "another",
		amt: num.NewUint(1000),
		avg: num.DecimalFromFloat(3000),
	}
	es.SetPartyStake(inc.id, inc.amt)
	require.True(t, inc.avg.Equals(es.AvgEntryValuation(inc.id)), es.AvgEntryValuation(inc.id).String())
}

func testAvgEntryUpdateStep3Add(t *testing.T, es *future.EquityShares, inc *lpdata) {
	t.Helper()
	// at this point, the total vStake is 2980, get it back up to 3k
	// calc for delta 10: (average entry valuation) x S / (S + Delta S) + (entry valuation) x (Delta S) / (S + Delta S)
	// using LP0 => 900 * 900 / 920 + 2980 * 20 / 920 == 945.6521739130434783
	inc.amt.Add(inc.amt, num.NewUint(20))
	inc.avg, _ = num.DecimalFromString("945.6521739130434783")
	es.SetPartyStake(inc.id, inc.amt)
	require.True(t, inc.avg.Equals(es.AvgEntryValuation(inc.id)), es.AvgEntryValuation(inc.id).String())
}

func testAvgEntryUpdateStep3Growth(t *testing.T, es *future.EquityShares) {
	t.Helper()
	// first, set the initial avg trade value
	val := num.DecimalFromFloat(1000000) // 1 million
	es.AvgTradeValue(val)
	vStake := num.DecimalFromFloat(2980)
	delta := num.DecimalFromFloat(20)
	factor := delta.Div(vStake)
	val = val.Add(factor.Mul(val)) // increase the value by 20/total_v_stake * previous value => growth rate should increase vStake back up to 3k
	// this actually is going to set total vStake to 3000.000000000000136. Not perfect, but it's pretty close
	es.AvgTradeValue(val)
}

func testAvgEntryValuationGrowth(t *testing.T) {
	es := future.NewEquityShares(num.DecimalZero())
	tradeVal := num.DecimalFromFloat(1000)
	lps := []lpdata{
		{
			id:  "LP1",
			amt: num.NewUint(100),
			avg: num.DecimalFromFloat(100),
		},
		{
			id:  "LP2",
			amt: num.NewUint(200),
			avg: num.DecimalFromFloat(300),
		},
	}

	for _, l := range lps {
		es.SetPartyStake(l.id, l.amt)
		require.True(t, l.avg.Equals(es.AvgEntryValuation(l.id)), es.AvgEntryValuation(l.id).String())
	}
	es.OpeningAuctionEnded()

	// lps[1].avg = num.DecimalFromFloat(100)
	// set trade value at auction end
	es.AvgTradeValue(tradeVal)
	for _, l := range lps {
		aev := es.AvgEntryValuation(l.id)
		require.True(t, l.avg.Equals(es.AvgEntryValuation(l.id)), fmt.Sprintf("FAIL ==> expected %s, got %s", l.avg, aev))
	}

	// growth
	tradeVal = num.DecimalFromFloat(1100)
	// aev1, _ := num.DecimalFromString("100.000000000000001")
	// lps[1].avg = aev1.Add(aev1) // double
	es.AvgTradeValue(tradeVal)
	for _, l := range lps {
		aev := es.AvgEntryValuation(l.id)
		require.True(t, l.avg.Equals(es.AvgEntryValuation(l.id)), fmt.Sprintf("FAIL => expected %s, got %s", l.avg, aev))
	}
	lps[1].amt = num.NewUint(150) // reduce LP
	es.SetPartyStake(lps[1].id, lps[1].amt)
	for _, l := range lps {
		aev := es.AvgEntryValuation(l.id)
		require.True(t, l.avg.Equals(es.AvgEntryValuation(l.id)), fmt.Sprintf("FAIL => expected %s, got %s", l.avg, aev))
	}
	// now simulate negative growth (ie r == 0)
	tradeVal = num.DecimalFromFloat(1000)
	es.AvgTradeValue(tradeVal)
	// avg should line up with physical stake once more
	// lps[1].avg = num.DecimalFromFloat(150)
	for _, l := range lps {
		aev := es.AvgEntryValuation(l.id)
		require.True(t, l.avg.Equals(es.AvgEntryValuation(l.id)), fmt.Sprintf("FAIL => expected %s, got %s", l.avg, aev))
	}
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

	es := future.NewEquityShares(num.DecimalFromFloat(100))

	// Set LP1
	es.SetPartyStake("LP1", num.NewUint(100))
	t.Run("LP1", func(t *testing.T) {
		s := es.SharesExcept(map[string]struct{}{})
		assert.True(t, one.Equal(s["LP1"]))
	})

	// Set LP2
	es.SetPartyStake("LP2", num.NewUint(200))
	t.Run("LP2", func(t *testing.T) {
		s := es.SharesExcept(map[string]struct{}{})
		lp1, lp2 := s["LP1"], s["LP2"]

		assert.Equal(t, oneThird, lp1)
		assert.Equal(t, twoThirds, lp2)
		assert.True(t, one.Equal(lp1.Add(lp2)))
	})

	// Set LP3
	es.SetPartyStake("LP3", num.NewUint(300))
	t.Run("LP3", func(t *testing.T) {
		s := es.SharesExcept(map[string]struct{}{})

		lp1, lp2, lp3 := s["LP1"], s["LP2"], s["LP3"]

		assert.Equal(t, oneSixth, lp1)
		assert.Equal(t, oneThird, lp2)
		assert.Equal(t, half, lp3)
		assert.True(t, one.Equal(lp1.Add(lp2).Add(lp3)))
	})

	// LP2 is undeployed
	t.Run("LP3", func(t *testing.T) {
		// pass LP as undeployed
		s := es.SharesExcept(map[string]struct{}{"LP2": {}})

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
	t.Helper()
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)

	return &equityShareMarket{
		t:         t,
		tm:        getTestMarket(t, now, nil, &types.AuctionDuration{Duration: 1}),
		parties:   map[string]struct{}{},
		Now:       now,
		ClosingAt: closingAt,
	}
}

func (esm *equityShareMarket) TestMarket() *testMarket { return esm.tm }

func (esm *equityShareMarket) BuildOrder(id, party string, side types.Side, price uint64) *types.Order {
	return &types.Order{
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTT,
		Status:      types.OrderStatusActive,
		ID:          id,
		Side:        side,
		Party:       party,
		MarketID:    esm.tm.market.GetID(),
		Size:        1,
		Price:       num.NewUint(price),
		Remaining:   1,
		CreatedAt:   esm.Now.UnixNano(),
		ExpiresAt:   esm.ClosingAt.UnixNano(),
	}
}

func (esm *equityShareMarket) createPartyIfMissing(t *testing.T, party string) {
	t.Helper()
	if _, ok := esm.parties[party]; !ok {
		esm.parties[party] = struct{}{}
		addAccount(t, esm.tm, party)
	}
}

func (esm *equityShareMarket) SubmitOrder(t *testing.T, ctx context.Context, order *types.Order) (*types.OrderConfirmation, error) {
	t.Helper()
	esm.createPartyIfMissing(t, order.Party)
	return esm.tm.market.SubmitOrder(ctx, order)
}

func (esm *equityShareMarket) WithSubmittedOrder(t *testing.T, id, party string, side types.Side, price uint64) *equityShareMarket {
	t.Helper()
	ctx := context.Background()
	order := esm.BuildOrder(id, party, side, price)

	_, err := esm.SubmitOrder(t, ctx, order)
	require.NoError(esm.t, err)
	return esm
}

func (esm *equityShareMarket) WithSubmittedLiquidityProvision(t *testing.T, party, id string, amount uint64, fee string, buys, sells []*types.LiquidityOrder) *equityShareMarket {
	t.Helper()
	esm.createPartyIfMissing(t, party)
	esm.tm.WithSubmittedLiquidityProvision(esm.t, party, amount, fee, buys, sells)
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
		ctx = vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
		// as we will split fees in 1/3 and 2/3
		// we use 900000 cause we need this number to be divisible by 3
		matchingPrice = uint64(900000)
		one           = uint64(1)
	)

	// Setup a market with a set of non-matching orders and Liquidity Provision
	// Submissions from 2 parties.
	esm := newEquityShareMarket(t).
		WithSubmittedOrder(t, "some-id-1", "party1", types.SideSell, matchingPrice+one).
		WithSubmittedOrder(t, "some-id-2", "party2", types.SideBuy, matchingPrice-one).
		WithSubmittedOrder(t, "some-id-3", "party1", types.SideSell, matchingPrice).
		WithSubmittedOrder(t, "some-id-4", "party2", types.SideBuy, matchingPrice). // Need to generate a trade to leave opening auction
		// party1 (commitment: 2000) should get 2/3 of the fee
		WithSubmittedLiquidityProvision(t, "party1", "lp-id-1", 2000000, "0.5", []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 11, 1),
		}, []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 10, 1),
		}).
		// party2 (commitment: 1000) should get 1/3 of the fee
		WithSubmittedLiquidityProvision(t, "party2", "lp-id-2", 1000000, "0.5", []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 10, 1),
		}, []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 11, 1),
		})

	// tm is the testMarket instance
	var (
		tm      = esm.TestMarket()
		curTime = esm.Now
	)

	// End opening auction
	curTime = curTime.Add(2 * time.Second)
	tm.now = curTime
	tm.market.OnTick(ctx, curTime)

	md := esm.tm.market.GetMarketData()
	require.NotNil(t, md)
	fmt.Printf("Target stake: %s\nSupplied: %s\n\n", md.TargetStake, md.SuppliedStake)
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	t.Run("WhenNoTrades", func(t *testing.T) {
		// clean up previous events
		tm.events = []events.Event{}

		// Trigger Fee distribution
		curTime = curTime.Add(1 * time.Second)
		tm.now = curTime
		tm.market.OnTick(ctx, curTime)

		// Assert the event
		var evt *events.LedgerMovements
		for _, e := range tm.events {
			if e.Type() == events.LedgerMovementsEvent {
				evt = e.(*events.LedgerMovements)
			}
		}
		require.Nil(t, evt, "should receive no TransferEvent")
	})

	// Match a pair of orders (same price) to trigger a fee distribution.
	conf, err := esm.
		WithSubmittedOrder(t, "some-id-3", "party1", types.SideSell, matchingPrice).
		SubmitOrder(t, context.Background(), esm.BuildOrder("some-id-4", "party2", types.SideBuy, matchingPrice))
	require.NoError(t, err)
	require.Len(t, conf.Trades, 1)

	// Retrieve both MarketLiquidityFee account balance and Party Balance
	// before the fee distribution.
	var (
		originalBalance = esm.LiquidityFeeAccount().Balance.Clone()
		party1Balance   = esm.PartyGeneralAccount("party1").Balance.Clone()
		party2Balance   = esm.PartyGeneralAccount("party2").Balance.Clone()
	)

	curTime = curTime.Add(1 * time.Second)
	tm.now = curTime
	tm.market.OnTick(ctx, curTime)

	md = esm.tm.market.GetMarketData()
	require.NotNil(t, md)
	fmt.Printf("Target stake: %s\nSupplied: %s\n\n", md.TargetStake, md.SuppliedStake)
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	oneU := num.NewUint(1)
	assert.True(t, esm.LiquidityFeeAccount().Balance.EQ(oneU),
		"LiquidityFeeAccount should have a balance of 1 (remainder)")

	// exp = originalBalance*(2/3)
	exp := num.UintZero().Mul(num.Sum(oneU, oneU), originalBalance)
	exp = exp.Div(exp, num.Sum(oneU, oneU, oneU))
	actual := num.UintZero().Sub(esm.PartyGeneralAccount("party1").Balance, party1Balance)
	assert.True(t,
		exp.EQ(actual),
		"party1 should get 2/3 of the fees (got %s expected %s)", actual.String(), exp.String(),
	)

	// exp = originalBalance*(1/3)
	exp = num.UintZero().Div(originalBalance, num.Sum(oneU, oneU, oneU))
	// minus the remainder
	exp.Sub(exp, oneU)
	actual = num.UintZero().Sub(esm.PartyGeneralAccount("party2").Balance, party2Balance)
	assert.True(t,
		exp.EQ(actual),
		"party2 should get 2/3 of the fees (got %s expected %s)", actual.String(), exp.String(),
	)
}

func getHash(es *future.EquityShares) []byte {
	state := es.GetState()
	esproto := state.IntoProto()
	bytes, _ := proto.Marshal(esproto)
	return crypto.Hash(bytes)
}

func TestSnapshotEmpty(t *testing.T) {
	es := future.NewEquityShares(num.DecimalFromFloat(100))

	// Get the hash of an empty object
	hash1 := getHash(es)

	// Create a new object and load the snapshot into it
	es2 := future.NewEquitySharesFromSnapshot(es.GetState())

	// Check the hash matches
	hash2 := getHash(es2)
	assert.Equal(t, hash1, hash2)
}

func TestSnapshotWithChanges(t *testing.T) {
	es := future.NewEquityShares(num.DecimalFromFloat(100))

	// Get the hash of an empty object
	hash1 := getHash(es)

	// Make changes to the original object
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("ID%05d", i)
		es.SetPartyStake(id, num.NewUint(uint64(i*100)))
	}

	// Check the hash has changed
	hash2 := getHash(es)
	assert.NotEqual(t, hash1, hash2)

	// Restore the state into a new object
	es2 := future.NewEquitySharesFromSnapshot(es.GetState())

	// Check the hashes match
	hash3 := getHash(es2)
	assert.Equal(t, hash2, hash3)
}
