package statevar_test

import (
	"testing"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
	"github.com/stretchr/testify/require"
)

var tol, _ = num.DecimalFromString("0.1")

func TestFloatMatrix(t *testing.T) {
	t.Run("test equality of two float matrices", testFloatMatrixEquality)
	t.Run("test two matrices are within tolerance of each other", testFloatMatrixWithinTol)
	t.Run("test converion of float matrix to a decimal matrix", testFloatMatrixToDecimal)
	t.Run("test conversion to proto", testMatrixToProto)
}

// testFloatVectorEquality tests that given the same key and equal/not equal value, equals function returns the correct value.
func testFloatMatrixEquality(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key: "matrix value",
		Val: &statevar.FloatMatrix{Val: [][]float64{[]float64{1.1, 2.2, 3.3, 4.4}, []float64{4.4, 3.3, 2.2, 1.1}}},
	})

	kvb2 := &statevar.KeyValueBundle{}
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key: "matrix value",
		Val: &statevar.FloatMatrix{Val: [][]float64{[]float64{1.1, 2.2, 3.3}}},
	})

	// the matrices have different shape
	require.False(t, kvb1.Equals(kvb2))

	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key: "matrix value",
		Val: &statevar.FloatMatrix{Val: [][]float64{[]float64{1.1, 2.2, 3.3, 4.4}, []float64{-4.4, -3.3, -2.2, -1.1}}},
	})

	// the matrices have same shape but different values
	require.False(t, kvb1.Equals(kvb3))

	kvb4 := &statevar.KeyValueBundle{}
	kvb4.KVT = append(kvb4.KVT, statevar.KeyValueTol{
		Key: "matrix value",
		Val: &statevar.FloatMatrix{Val: [][]float64{[]float64{1.1, 2.2, 3.3, 4.4}, []float64{4.4, 3.3, 2.2, 1.1}}},
	})

	// they are equal
	require.True(t, kvb1.Equals(kvb4))
}

// testFloatVectorWithinTol check that the values of the vectors are within tolerance.
func testFloatMatrixWithinTol(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.FloatMatrix{Val: [][]float64{[]float64{1.1, 2.2, 3.3, 4.4}, []float64{4.4, 3.3, 2.2, 1.1}}},
		Tolerance: tol,
	})

	kvb2 := &statevar.KeyValueBundle{}
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.FloatMatrix{Val: [][]float64{[]float64{1.2, 2.2, 3.2, 4.5}, []float64{4, 3.3, 2.2, 1.1}}},
		Tolerance: tol,
	})

	// almost within tolerance but not
	require.False(t, kvb1.WithinTolerance(kvb2))

	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.FloatMatrix{Val: [][]float64{[]float64{1.09, 2.2, 3.21, 4.49}, []float64{4.31, 3.2199999, 2.2, 1.0999999999}}},
		Tolerance: tol,
	})

	// within tolerance on either side
	require.True(t, kvb1.WithinTolerance(kvb3))
}

// testFloatVectorToDecimal tests conversion to decimal.
func testFloatMatrixToDecimal(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.FloatMatrix{Val: [][]float64{[]float64{1.1, 2.2, 3.3, 4.4}, []float64{-4.4, -3.3, -2.2, -1.1}}},
		Tolerance: tol,
	})

	res1 := kvb1.ToDecimal()
	res := res1.KeyDecimalValue[kvb1.KVT[0].Key]
	switch v := res.(type) {
	case *statevar.DecimalMatrixValue:
		require.Equal(t, [][]num.Decimal{
			[]num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)},
			[]num.Decimal{num.DecimalFromFloat(-4.4), num.DecimalFromFloat(-3.3), num.DecimalFromFloat(-2.2), num.DecimalFromFloat(-1.1)}},
			v.Value)
	default:
		t.Fail()
	}
}

// testMatrixToProto tests conversion of matrix variable to proto.
func testMatrixToProto(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.FloatMatrix{Val: [][]float64{[]float64{1.1, 2.2, 3.3, 4.4}, []float64{-4.4, -3.3, -2.2, -1.1}}},
		Tolerance: tol,
	})
	res := kvb1.ToProto()
	require.Equal(t, 1, len(res))
	require.Equal(t, "matrix value", res[0].Key)
	require.Equal(t, "0.1", res[0].Tolerance)
	switch v := res[0].Value.Value.(type) {
	case *vega.StateVarValue_MatrixVal:
		require.Equal(t, []float64{1.1, 2.2, 3.3, 4.4}, v.MatrixVal.Value[0].Value)
		require.Equal(t, []float64{-4.4, -3.3, -2.2, -1.1}, v.MatrixVal.Value[1].Value)
	default:
		t.Fail()
	}

	kvb2 := statevar.KeyValueBundleFromProto(res)
	require.Equal(t, kvb1, kvb2)

}
