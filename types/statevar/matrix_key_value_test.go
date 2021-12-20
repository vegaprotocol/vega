package statevar_test

import (
	"testing"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
	"github.com/stretchr/testify/require"
)

var tol, _ = num.DecimalFromString("0.1")

func TestDecimalMatrix(t *testing.T) {
	t.Run("test equality of two decimal matrices", testDecimalMatrixEquality)
	t.Run("test two matrices are within tolerance of each other", testDecimalMatrixWithinTol)
	t.Run("test converion of decimal matrix to a decimal matrix", testDecimalMatrixToDecimal)
	t.Run("test conversion to proto", testMatrixToProto)
}

// testFloatVectorEquality tests that given the same key and equal/not equal value, equals function returns the correct value.
func testDecimalMatrixEquality(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key: "matrix value",
		Val: &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, {num.DecimalFromFloat(4.4), num.DecimalFromFloat(3.3), num.DecimalFromFloat(2.2), num.DecimalFromFloat(1.1)}}},
	})

	kvb2 := &statevar.KeyValueBundle{}
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key: "matrix value",
		Val: &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3)}}},
	})

	// the matrices have different shape
	require.False(t, kvb1.Equals(kvb2))

	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key: "matrix value",
		Val: &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, {num.DecimalFromFloat(-4.4), num.DecimalFromFloat(-3.3), num.DecimalFromFloat(-2.2), num.DecimalFromFloat(-1.1)}}},
	})

	// the matrices have same shape but different values
	require.False(t, kvb1.Equals(kvb3))

	kvb4 := &statevar.KeyValueBundle{}
	kvb4.KVT = append(kvb4.KVT, statevar.KeyValueTol{
		Key: "matrix value",
		Val: &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, {num.DecimalFromFloat(4.4), num.DecimalFromFloat(3.3), num.DecimalFromFloat(2.2), num.DecimalFromFloat(1.1)}}},
	})

	// they are equal
	require.True(t, kvb1.Equals(kvb4))
}

// testDecimalMatrixWithinTol check that the values of the vectors are within tolerance.
func testDecimalMatrixWithinTol(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, {num.DecimalFromFloat(4.4), num.DecimalFromFloat(3.3), num.DecimalFromFloat(2.2), num.DecimalFromFloat(1.1)}}},
		Tolerance: tol,
	})

	kvb2 := &statevar.KeyValueBundle{}
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.2), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.2), num.DecimalFromFloat(4.5)}, {num.DecimalFromFloat(4), num.DecimalFromFloat(3.3), num.DecimalFromFloat(2.2), num.DecimalFromFloat(1.1)}}},
		Tolerance: tol,
	})

	// almost within tolerance but not
	require.False(t, kvb1.WithinTolerance(kvb2))

	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.09), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.21), num.DecimalFromFloat(4.49)}, {num.DecimalFromFloat(4.31), num.DecimalFromFloat(3.2199999), num.DecimalFromFloat(2.2), num.DecimalFromFloat(1.0999999999)}}},
		Tolerance: tol,
	})

	// within tolerance on either side
	require.True(t, kvb1.WithinTolerance(kvb3))
}

// testDecimalMatrixToDecimal tests conversion to decimal.
func testDecimalMatrixToDecimal(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, {num.DecimalFromFloat(-4.4), num.DecimalFromFloat(-3.3), num.DecimalFromFloat(-2.2), num.DecimalFromFloat(-1.1)}}},
		Tolerance: tol,
	})

	res := kvb1.KVT[0].Val
	switch v := res.(type) {
	case *statevar.DecimalMatrix:
		require.Equal(t, [][]num.Decimal{
			{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)},
			{num.DecimalFromFloat(-4.4), num.DecimalFromFloat(-3.3), num.DecimalFromFloat(-2.2), num.DecimalFromFloat(-1.1)},
		},
			v.Val)
	default:
		t.Fail()
	}
}

// testMatrixToProto tests conversion of matrix variable to proto.
func testMatrixToProto(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, {num.DecimalFromFloat(-4.4), num.DecimalFromFloat(-3.3), num.DecimalFromFloat(-2.2), num.DecimalFromFloat(-1.1)}}},
		Tolerance: tol,
	})
	res := kvb1.ToProto()
	require.Equal(t, 1, len(res))
	require.Equal(t, "matrix value", res[0].Key)
	require.Equal(t, "0.1", res[0].Tolerance)
	switch v := res[0].Value.Value.(type) {
	case *vega.StateVarValue_MatrixVal:
		require.Equal(t, []string{num.DecimalFromFloat(1.1).String(), num.DecimalFromFloat(2.2).String(), num.DecimalFromFloat(3.3).String(), num.DecimalFromFloat(4.4).String()}, v.MatrixVal.Value[0].Value)
		require.Equal(t, []string{num.DecimalFromFloat(-4.4).String(), num.DecimalFromFloat(-3.3).String(), num.DecimalFromFloat(-2.2).String(), num.DecimalFromFloat(-1.1).String()}, v.MatrixVal.Value[1].Value)
	default:
		t.Fail()
	}

	kvb2 := statevar.KeyValueBundleFromProto(res)
	require.Equal(t, kvb1, kvb2)
}
