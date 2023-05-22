package spot_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/core/execution/spot"
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
	t.Run("Average entry valuation after 6063 spec change", testAvgEntryUpdate)
}

// replicate the example given in spec file (protocol/0042-LIQF-setting_fees_and_rewarding_lps.md).
func testAvgEntryUpdate(t *testing.T) {
	es := spot.NewEquityShares(num.DecimalZero())
	es.OpeningAuctionEnded()
	initial := lpdata{
		id:  "initial",
		amt: num.NewUint(900),
		avg: num.DecimalFromFloat(900),
	}
	es.SetPartyStake(initial.id, initial.amt, initial.amt, num.DecimalOne())
	require.True(t, initial.avg.Equals(es.AvgEntryValuation(initial.id)), es.AvgEntryValuation(initial.id).String())
	// step 1 from the example: LP commitment of 100 with an existing commitment of 1k:
	step1 := lpdata{
		id:  "step1",
		amt: num.NewUint(100),
		avg: num.NewDecimalFromFloat(1000),
	}
	es.SetPartyStake(step1.id, step1.amt, step1.amt, num.DecimalOne())
	require.True(t, step1.avg.Equals(es.AvgEntryValuation(step1.id)), es.AvgEntryValuation(step1.id).String())
	// get sum of all vStake to 2K as per example in the spec
	// total vStake == 1.1k => avg is now 1.1k
	inc := lpdata{
		id:  "topup",
		amt: num.NewUint(990),
		avg: num.DecimalFromFloat(1990),
	}
	es.SetPartyStake(inc.id, inc.amt, inc.amt, num.DecimalOne())
	require.True(t, inc.avg.Equals(es.AvgEntryValuation(inc.id)), es.AvgEntryValuation(inc.id).String())
	// Example 2: We have a total vStake of 2k -> step1 party increases the commitment amount to 110 (so +10)
	step1.amt = num.NewUint(110)
	step1.avg, _ = num.DecimalFromString("1090.9090909090909091")
	es.SetPartyStake(step1.id, step1.amt, step1.amt, num.DecimalOne())
	require.True(t, step1.avg.Equals(es.AvgEntryValuation(step1.id)), es.AvgEntryValuation(step1.id).String())
	// increase total vStake to be 3k using a new LP party
	testAvgEntryUpdateStep3New(t, es)
	// example 3 when total vStake is 3k -> decrease commitment by 20
	step1.amt = num.NewUint(90)
	es.SetPartyStake(step1.id, step1.amt, step1.amt, num.DecimalOne())
	require.True(t, step1.avg.Equals(es.AvgEntryValuation(step1.id)), es.AvgEntryValuation(step1.id).String())
	// set up example 3 again, this time by increasing the commitment of an existing party, their AEV should be updated accordingly
	testAvgEntryUpdateStep3Add(t, es, &initial)
	step1.amt = num.NewUint(70) // decrease by another 20
	es.SetPartyStake(step1.id, step1.amt, step1.amt, num.DecimalOne())
	require.True(t, step1.avg.Equals(es.AvgEntryValuation(step1.id)), es.AvgEntryValuation(step1.id).String())
	// set up for example 3, this time by increasing the total vStake to 3k using growth
	testAvgEntryUpdateStep3Growth(t, es)
	step1.amt = num.NewUint(50) // decrease by another 20
	es.SetPartyStake(step1.id, step1.amt, step1.amt, num.DecimalOne())
	require.True(t, step1.avg.Equals(es.AvgEntryValuation(step1.id)), es.AvgEntryValuation(step1.id).String())
}

// continue based on testAvgEntryUpdate setup, just add new LP to get the total up to 3k.
func testAvgEntryUpdateStep3New(t *testing.T, es *spot.EquityShares) {
	t.Helper()
	// we have 1000 + 110 + 990 (2000)
	// AEV == 2000
	inc := lpdata{
		id:  "another",
		amt: num.NewUint(1000),
		avg: num.DecimalFromFloat(3000),
	}
	es.SetPartyStake(inc.id, inc.amt, inc.amt, num.DecimalOne())
	require.True(t, inc.avg.Equals(es.AvgEntryValuation(inc.id)), es.AvgEntryValuation(inc.id).String())
}

func testAvgEntryUpdateStep3Add(t *testing.T, es *spot.EquityShares, inc *lpdata) {
	t.Helper()
	// at this point, the total vStake is 2980, get it back up to 3k
	// calc for delta 10: (average entry valuation) x S / (S + Delta S) + (entry valuation) x (Delta S) / (S + Delta S)
	// using LP0 => 900 * 900 / 920 + 2980 * 20 / 920 == 945.6521739130434783
	inc.amt.Add(inc.amt, num.NewUint(20))
	inc.avg, _ = num.DecimalFromString("945.6521739130434783")
	es.SetPartyStake(inc.id, inc.amt, inc.amt, num.DecimalOne())
	require.True(t, inc.avg.Equals(es.AvgEntryValuation(inc.id)), es.AvgEntryValuation(inc.id).String())
}

func testAvgEntryUpdateStep3Growth(t *testing.T, es *spot.EquityShares) {
	t.Helper()
	// first, set the initial avg trade value
	val := num.DecimalFromFloat(1000000) // 1 million
	es.AvgTradeValue(val, num.DecimalOne())
	vStake := num.DecimalFromFloat(2980)
	delta := num.DecimalFromFloat(20)
	factor := delta.Div(vStake)
	val = val.Add(factor.Mul(val)) // increase the value by 20/total_v_stake * previous value => growth rate should increase vStake back up to 3k
	// this actually is going to set total vStake to 3000.000000000000136. Not perfect, but it's pretty close
	es.AvgTradeValue(val, num.DecimalOne())
}

func testAvgEntryValuationGrowth(t *testing.T) {
	es := spot.NewEquityShares(num.DecimalZero())
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
		es.SetPartyStake(l.id, l.amt, l.amt, num.DecimalOne())
		require.Equal(t, l.avg.String(), es.AvgEntryValuation(l.id).String(), es.AvgEntryValuation(l.id).String())
	}
	es.OpeningAuctionEnded()

	// lps[1].avg = num.DecimalFromFloat(100)
	// set trade value at auction end
	es.AvgTradeValue(tradeVal, num.DecimalOne())
	for _, l := range lps {
		aev := es.AvgEntryValuation(l.id)
		require.True(t, l.avg.Equals(es.AvgEntryValuation(l.id)), fmt.Sprintf("FAIL ==> expected %s, got %s", l.avg, aev))
	}

	// growth
	tradeVal = num.DecimalFromFloat(1100)
	// aev1, _ := num.DecimalFromString("100.000000000000001")
	// lps[1].avg = aev1.Add(aev1) // double
	es.AvgTradeValue(tradeVal, num.DecimalOne())
	for _, l := range lps {
		aev := es.AvgEntryValuation(l.id)
		require.True(t, l.avg.Equals(es.AvgEntryValuation(l.id)), fmt.Sprintf("FAIL => expected %s, got %s", l.avg, aev))
	}
	lps[1].amt = num.NewUint(150) // reduce LP
	es.SetPartyStake(lps[1].id, lps[1].amt, lps[1].amt, num.DecimalOne())
	for _, l := range lps {
		aev := es.AvgEntryValuation(l.id)
		require.True(t, l.avg.Equals(es.AvgEntryValuation(l.id)), fmt.Sprintf("FAIL => expected %s, got %s", l.avg, aev))
	}
	// now simulate negative growth (ie r == 0)
	tradeVal = num.DecimalFromFloat(1000)
	es.AvgTradeValue(tradeVal, num.DecimalOne())
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

	es := spot.NewEquityShares(num.DecimalFromFloat(100))

	// Set LP1
	es.SetPartyStake("LP1", num.NewUint(100), num.NewUint(100), num.DecimalOne())
	t.Run("LP1", func(t *testing.T) {
		s := es.SharesExcept(map[string]struct{}{})
		assert.True(t, one.Equal(s["LP1"]))
	})

	// Set LP2
	es.SetPartyStake("LP2", num.NewUint(200), num.NewUint(200), num.DecimalOne())
	t.Run("LP2", func(t *testing.T) {
		s := es.SharesExcept(map[string]struct{}{})
		lp1, lp2 := s["LP1"], s["LP2"]

		assert.Equal(t, oneThird, lp1)
		assert.Equal(t, twoThirds, lp2)
		assert.True(t, one.Equal(lp1.Add(lp2)))
	})

	// Set LP3
	es.SetPartyStake("LP3", num.NewUint(300), num.NewUint(300), num.DecimalOne())
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

func getHash(es *spot.EquityShares) []byte {
	state := es.GetState()
	esproto := state.IntoProto()
	bytes, _ := proto.Marshal(esproto)
	return crypto.Hash(bytes)
}

func TestSnapshotEmpty(t *testing.T) {
	es := spot.NewEquityShares(num.DecimalFromFloat(100))

	// Get the hash of an empty object
	hash1 := getHash(es)

	// Create a new object and load the snapshot into it
	es2 := spot.NewEquitySharesFromSnapshot(es.GetState())

	// Check the hash matches
	hash2 := getHash(es2)
	assert.Equal(t, hash1, hash2)
}

func TestSnapshotWithChanges(t *testing.T) {
	es := spot.NewEquityShares(num.DecimalFromFloat(100))

	// Get the hash of an empty object
	hash1 := getHash(es)

	// Make changes to the original object
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("ID%05d", i)
		es.SetPartyStake(id, num.NewUint(uint64(i*100)), num.NewUint(uint64(i*100)), num.DecimalOne())
	}

	// Check the hash has changed
	hash2 := getHash(es)
	assert.NotEqual(t, hash1, hash2)

	// Restore the state into a new object
	es2 := spot.NewEquitySharesFromSnapshot(es.GetState())

	// Check the hashes match
	hash3 := getHash(es2)
	assert.Equal(t, hash2, hash3)
}

func TestPartyStakeImbalancedCommitment(t *testing.T) {
	es := spot.NewEquityShares(num.DecimalZero())
	// new party, new stake, imbalanced, smaller buy

	// buy commitment = 1000
	// sell commitment = 20 (6000 in quote asset equivalent given the mark price)
	es.SetPartyStake("zohar", num.NewUint(1000), num.NewUint(20), num.DecimalFromInt64(300))

	// the smaller between then is 1000 so expect the physical commitment to be 1000
	// as this is the first stake expect the virtual stake to be equal the physical stake
	require.Equal(t, num.DecimalFromInt64(1000), es.GetTotalStake())
	require.Equal(t, num.DecimalFromInt64(1000), es.GetTotalVStake())

	// now lets swap them - i.e increase the buy stake and decrease the sell stake
	es.SetPartyStake("zohar", num.NewUint(1000), num.NewUint(1), num.DecimalFromInt64(300))

	// now the smaller is 300 which is the sell, expect the physical stake to reflect that
	require.Equal(t, num.DecimalFromInt64(300), es.GetTotalStake())
	require.Equal(t, "300", es.GetTotalVStake().String())

	// get some growth
	es.AvgTradeValue(num.DecimalFromInt64(320), num.DecimalFromInt64(300))
	es.AvgTradeValue(num.DecimalFromInt64(330), num.DecimalFromInt64(300))

	// physical stake not changed
	require.Equal(t, num.DecimalFromInt64(300), es.GetTotalStake())

	// virtual stake up = 300*330/320
	require.Equal(t, "309.375", es.GetTotalVStake().String())

	// flip the sides of what make the virtual stake, i.e. change the commitment such that now the smaller commitment is buy again
	// now the total physical commitment goes up
	es.SetPartyStake("zohar", num.NewUint(1000), num.NewUint(20), num.DecimalFromInt64(300))
	// physical stake: b= 1000 s= 20 p= 1000
	// virtual stake: 1031.25 s= 20.03125 v= 1031.25
	require.Equal(t, "1000", es.GetTotalStake().String())
	require.Equal(t, "1031.25", es.GetTotalVStake().String())

	// now set the buy commitment to 0
	es.SetPartyStake("zohar", nil, num.NewUint(20), num.DecimalFromInt64(300))
	require.Equal(t, "0", es.GetTotalStake().String())
	require.Equal(t, "0", es.GetTotalVStake().String())

	// now set the sell commitment to 0
	es.SetPartyStake("zohar", nil, nil, num.DecimalFromInt64(300))
	require.Equal(t, "0", es.GetTotalStake().String())
	require.Equal(t, "0", es.GetTotalVStake().String())
}
