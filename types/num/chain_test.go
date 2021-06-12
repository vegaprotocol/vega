package num_test

import (
	"testing"

	"code.vegaprotocol.io/vega/types/num"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestUintChain(t *testing.T) {
	start := num.NewUint(100)
	// use the non-chainable API to get the expected result
	vals := map[uint64]*num.Uint{
		1: nil,
		2: nil,
		3: nil,
		4: nil,
		5: nil,
		6: nil,
		7: nil,
		8: nil,
		9: nil,
	}
	for k := range vals {
		vals[k] = num.NewUint(k)
	}
	expect := num.Sum(start, vals[3], vals[5]) // some stuff
	expect.AddSum(vals[9], vals[1])
	expect.Sub(expect, vals[7])
	expect.Mul(expect, vals[4])
	expect.Div(expect, vals[2])

	// write as one-liner
	got := num.UintChain(num.Sum(start, vals[3], vals[5])).Add(vals[9], vals[1]).Sub(vals[7]).Mul(vals[4]).Div(vals[2]).Get()

	require.Equal(t, expect.String(), got.String())
}

func TestDecChain(t *testing.T) {
	start := decimal.NewFromInt(100)
	vals := map[int64]num.Decimal{
		1: decimal.Zero,
		2: decimal.Zero,
		3: decimal.Zero,
		4: decimal.Zero,
		5: decimal.Zero,
		6: decimal.Zero,
		7: decimal.Zero,
		8: decimal.Zero,
		9: decimal.Zero,
	}
	for k := range vals {
		vals[k] = decimal.NewFromInt(k)
	}

	expect := start.Add(vals[3]).Add(vals[5]).Add(vals[9].Add(vals[1])).Sub(vals[7]).Mul(vals[4]).Div(vals[2])

	// we could just assign got at the end, but we want to check for rounding
	chain := num.DecChain(start).Add(vals[3], vals[5], vals[9].Add(vals[1])).Sub(vals[7]).Mul(vals[4]).Div(vals[2])

	require.Equal(t, expect, chain.Get())

	t.Run("compare rounding division", func(t *testing.T) {
		expect = expect.DivRound(vals[8], 1) // 27.75 -> 27.8
		chain.DivRound(vals[8], 1)
		require.Equal(t, expect, chain.Get())
	})

	t.Run("compare Uint rounding", func(t *testing.T) {
		floor, round, ceil := expect.Floor(), expect.Round(0), expect.Ceil()
		uf, _ := chain.GetUint(num.DecFloor)
		require.Equal(t, floor.String(), uf.String())
		ur, _ := chain.GetUint(num.DecRound)
		require.Equal(t, round.String(), ur.String())
		uc, _ := chain.GetUint(num.DecCeil)
		require.Equal(t, ceil.String(), uc.String())
	})
}

// same as the TestDecChain, only we're starting with a Uint
func TestDecChainFromUint(t *testing.T) {
	sInt := int64(100)
	start := decimal.NewFromInt(sInt)
	uStart := num.NewUint(uint64(sInt))
	vals := map[int64]num.Decimal{
		1: decimal.Zero,
		2: decimal.Zero,
		3: decimal.Zero,
		4: decimal.Zero,
		5: decimal.Zero,
		6: decimal.Zero,
		7: decimal.Zero,
		8: decimal.Zero,
		9: decimal.Zero,
	}
	for k := range vals {
		vals[k] = decimal.NewFromInt(k)
	}

	expect := start.Add(vals[3]).Add(vals[5]).Add(vals[9].Add(vals[1])).Sub(vals[7]).Mul(vals[4]).Div(vals[2])

	// we could just assign got at the end, but we want to check for rounding
	chain := num.UintDecChain(uStart).Add(vals[3], vals[5], vals[9].Add(vals[1])).Sub(vals[7]).Mul(vals[4]).Div(vals[2])

	require.Equal(t, expect, chain.Get())

	t.Run("compare rounding division", func(t *testing.T) {
		expect = expect.DivRound(vals[8], 1) // 27.75 -> 27.8
		chain.DivRound(vals[8], 1)
		require.Equal(t, expect, chain.Get())
	})

	t.Run("compare Uint rounding", func(t *testing.T) {
		floor, round, ceil := expect.Floor(), expect.Round(0), expect.Ceil()
		uf, _ := chain.GetUint(num.DecFloor)
		require.Equal(t, floor.String(), uf.String())
		ur, _ := chain.GetUint(num.DecRound)
		require.Equal(t, round.String(), ur.String())
		uc, _ := chain.GetUint(num.DecCeil)
		require.Equal(t, ceil.String(), uc.String())
	})
}
