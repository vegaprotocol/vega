package statevar_test

import (
	"testing"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
	"github.com/stretchr/testify/require"
)

func TestFloatScalar(t *testing.T) {
	t.Run("test equality of two float scalars", testFloatEquality)
	t.Run("test two scalar floats are within tolerance of each other", testScalarWithinTol)
	t.Run("test converion of float scalar to a decimal scalar", testScalarToDecimal)
	t.Run("test conversion to proto", testScalarToProto)
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
		Tolerance: num.DecimalFromInt64(1),
	})

	kvb2 := &statevar.KeyValueBundle{}
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.FloatValue{Val: 6.54321},
		Tolerance: num.DecimalFromInt64(1),
	})

	require.False(t, kvb1.WithinTolerance(kvb2))

	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.FloatValue{Val: 2.23456},
		Tolerance: num.DecimalFromInt64(1),
	})
	require.True(t, kvb1.WithinTolerance(kvb3))
}

// testScalarToDecimal tests conversion to decimal
func testScalarToDecimal(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.FloatValue{Val: 1.23456},
		Tolerance: num.DecimalFromInt64(1),
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

func testScalarToProto(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.FloatValue{Val: 1.23456},
		Tolerance: num.DecimalFromInt64(1),
	})
	res := kvb1.ToProto()
	require.Equal(t, 1, len(res))
	require.Equal(t, "scalar value", res[0].Key)
	require.Equal(t, "1", res[0].Tolerance)
	switch v := res[0].Value.Value.(type) {
	case *vega.StateVarValue_ScalarVal:
		require.Equal(t, 1.23456, v.ScalarVal.Value)
	default:
		t.Fail()
	}

	kvb2 := statevar.KeyValueBundleFromProto(res)
	require.Equal(t, kvb1, kvb2)
}
