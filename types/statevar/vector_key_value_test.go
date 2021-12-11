package statevar_test

import (
	"testing"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
	"github.com/stretchr/testify/require"
)

func TestFloatVector(t *testing.T) {
	t.Run("test equality of two float vectors", testFloatVectorEquality)
	t.Run("test two vectors are within tolerance of each other", testFloatVectorWithinTol)
	t.Run("test converion of float vector to a decimal vector", testFloatVectorToDecimal)
	t.Run("test conversion to proto", testVectorToProto)
}

// testFloatVectorEquality tests that given the same key and equal/not equal value, equals function returns the correct value.
func testFloatVectorEquality(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key: "vector value",
		Val: &statevar.FloatVector{Val: []float64{1.1, 2.2, 3.3, 4.4}},
	})

	kvb2 := &statevar.KeyValueBundle{}
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key: "vector value",
		Val: &statevar.FloatVector{Val: []float64{1.1, 2.2, 3.3}},
	})

	// the vectors have different shape
	require.False(t, kvb1.Equals(kvb2))

	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key: "vector value",
		Val: &statevar.FloatVector{Val: []float64{2.2, 3.3, 4.4, 1.1}},
	})

	// the vectors have different values
	require.False(t, kvb1.Equals(kvb3))

	kvb4 := &statevar.KeyValueBundle{}
	kvb4.KVT = append(kvb4.KVT, statevar.KeyValueTol{
		Key: "vector value",
		Val: &statevar.FloatVector{Val: []float64{1.1, 2.2, 3.3, 4.4}},
	})

	// they are equal
	require.True(t, kvb1.Equals(kvb4))
}

// testFloatVectorWithinTol check that the values of the vectors are within tolerance.
func testFloatVectorWithinTol(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "vector value",
		Val:       &statevar.FloatVector{Val: []float64{1.1, 2.2, 3.3, 4.4}},
		Tolerance: tol,
	})

	kvb2 := &statevar.KeyValueBundle{}
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key:       "vector value",
		Val:       &statevar.FloatVector{Val: []float64{1.2, 2.3, 3.4, 4.6}},
		Tolerance: tol,
	})

	// almost within tolerance but not
	require.False(t, kvb1.WithinTolerance(kvb2))

	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key:       "vector value",
		Val:       &statevar.FloatVector{Val: []float64{1.099999999, 2.100000005, 3.3999999, 4.3000001}},
		Tolerance: tol,
	})

	// within tolerance on either side
	require.True(t, kvb1.WithinTolerance(kvb3))
}

// testFloatVectorToDecimal tests conversion to decimal.
func testFloatVectorToDecimal(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "vector value",
		Val:       &statevar.FloatVector{Val: []float64{1.1, 2.2, 3.3, 4.4}},
		Tolerance: tol,
	})

	res1 := kvb1.ToDecimal()
	res := res1.KeyDecimalValue[kvb1.KVT[0].Key]
	switch v := res.(type) {
	case *statevar.DecimalVectorValue:
		require.Equal(t, []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, v.Value)
	default:
		t.Fail()
	}
}

// testVectorToProto tests conversion of vector variable to proto.
func testVectorToProto(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "vector value",
		Val:       &statevar.FloatVector{Val: []float64{1.1, 2.2, 3.3, 4.4}},
		Tolerance: tol,
	})
	res := kvb1.ToProto()
	require.Equal(t, 1, len(res))
	require.Equal(t, "vector value", res[0].Key)
	require.Equal(t, "0.1", res[0].Tolerance)
	switch v := res[0].Value.Value.(type) {
	case *vega.StateVarValue_VectorVal:
		require.Equal(t, []float64{1.1, 2.2, 3.3, 4.4}, v.VectorVal.Value)
	default:
		t.Fail()
	}

	kvb2 := statevar.KeyValueBundleFromProto(res)
	require.Equal(t, kvb1, kvb2)
}
