package statevar_test

import (
	"testing"

	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/statevar"
	"github.com/stretchr/testify/require"
)

func TestFloatScalar(t *testing.T) {
	t.Run("test equality of two float scalars", testFloatEquality)
	t.Run("test two scalar floats are within tolerance of each other", testScalarWithinTol)
	t.Run("test converion of float scalar to a decimal scalar", testScalarToDecimal)
}

// testFloatEquality tests that given the same key and equal/not equal value, equals function returns the correct value
func testFloatEquality(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key: "scalar value",
		Val: &statevar.FloatValue{Val: 1.23456},
	})

	kvb2 := &statevar.KeyValueBundle{}
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key: "scalar value",
		Val: &statevar.FloatValue{Val: 6.54321},
	})

	require.False(t, kvb1.Equals(kvb2))

	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key: "scalar value",
		Val: &statevar.FloatValue{Val: 1.23456},
	})
	require.True(t, kvb1.Equals(kvb3))
}

func testScalarWithinTol(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.FloatValue{Val: 1.23456},
		Tolerance: 1,
	})

	kvb2 := &statevar.KeyValueBundle{}
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.FloatValue{Val: 6.54321},
		Tolerance: 1,
	})

	require.False(t, kvb1.WithinTolerance(kvb2))

	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.FloatValue{Val: 2.23456},
		Tolerance: 1,
	})
	require.True(t, kvb1.WithinTolerance(kvb3))
}

// testScalarToDecimal tests conversion to decimal
func testScalarToDecimal(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.FloatValue{Val: 1.23456},
		Tolerance: 1,
	})

	res1 := kvb1.ToDecimal()
	res := res1.KeyDecimalValue[kvb1.KVT[0].Key]
	switch v := res.(type) {
	case *statevar.DecimalScalarValue:
		require.Equal(t, num.DecimalFromFloat(1.23456), v.Value)
	default:
		t.Fail()
	}
}
