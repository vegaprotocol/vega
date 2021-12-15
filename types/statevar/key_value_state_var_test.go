package statevar_test

import (
	"testing"

	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
	"github.com/stretchr/testify/require"
)

func TestKVT(t *testing.T) {
	t.Run("test equality of two bundles", testEqualityAllMatch)
	t.Run("test two bundles are within tolerance of each other", testWithinTolerance)
	t.Run("test conversion of a bundle to decimal", testToDecimal)
}

func testEqualityAllMatch(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key: "scalar value",
		Val: &statevar.DecimalScalar{Val: num.DecimalFromFloat(1.23456)},
	})
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key: "vector value",
		Val: &statevar.DecimalVector{Val: []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}},
	})
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key: "matrix value",
		Val: &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, {num.DecimalFromFloat(4.4), num.DecimalFromFloat(3.3), num.DecimalFromFloat(2.2), num.DecimalFromFloat(1.1)}}},
	})

	require.True(t, kvb1.Equals(kvb1))

	// empty bundle shouldn't match
	kvb2 := &statevar.KeyValueBundle{}
	require.False(t, kvb2.Equals(kvb1))
	require.False(t, kvb1.Equals(kvb2))

	// bundle with differnet ordering of keys doesn't match
	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key: "matrix value",
		Val: &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, {num.DecimalFromFloat(4.4), num.DecimalFromFloat(3.3), num.DecimalFromFloat(2.2), num.DecimalFromFloat(1.1)}}},
	})
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key: "vector value",
		Val: &statevar.DecimalVector{Val: []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}},
	})
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key: "scalar value",
		Val: &statevar.DecimalScalar{Val: num.DecimalFromFloat(1.23456)},
	})
	require.False(t, kvb1.Equals(kvb3))
	require.False(t, kvb3.Equals(kvb1))
}

func testWithinTolerance(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.DecimalScalar{Val: num.DecimalFromFloat(1.23456)},
		Tolerance: num.DecimalFromInt64(1),
	})
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "vector value",
		Val:       &statevar.DecimalVector{Val: []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}},
		Tolerance: num.DecimalFromInt64(2),
	})
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, {num.DecimalFromFloat(4.4), num.DecimalFromFloat(3.3), num.DecimalFromFloat(2.2), num.DecimalFromFloat(1.1)}}},
		Tolerance: num.DecimalFromInt64(3),
	})

	kvb2 := &statevar.KeyValueBundle{}
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.DecimalScalar{Val: num.DecimalFromFloat(2.2)},
		Tolerance: num.DecimalFromInt64(1),
	})
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key:       "vector value",
		Val:       &statevar.DecimalVector{Val: []num.Decimal{num.DecimalFromFloat(3), num.DecimalFromFloat(4), num.DecimalFromFloat(1.3000000001), num.DecimalFromFloat(4)}},
		Tolerance: num.DecimalFromInt64(2),
	})
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(-1.1), num.DecimalFromFloat(1.1), num.DecimalFromFloat(0.31), num.DecimalFromFloat(2)}, {num.DecimalFromFloat(4.4), num.DecimalFromFloat(3.3), num.DecimalFromFloat(2.2), num.DecimalFromFloat(1.1)}}},
		Tolerance: num.DecimalFromInt64(3),
	})

	require.True(t, kvb1.WithinTolerance(kvb2))
	require.True(t, kvb2.WithinTolerance(kvb1))

	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.DecimalScalar{Val: num.DecimalFromFloat(0.2)}, // too far
		Tolerance: num.DecimalFromInt64(1),
	})
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key:       "vector value",
		Val:       &statevar.DecimalVector{Val: []num.Decimal{num.DecimalFromFloat(3), num.DecimalFromFloat(4), num.DecimalFromFloat(1.3000000001), num.DecimalFromFloat(4)}},
		Tolerance: num.DecimalFromInt64(2),
	})
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key:       "matrix value",
		Val:       &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(-1.1), num.DecimalFromFloat(1.1), num.DecimalFromFloat(0.31), num.DecimalFromFloat(2)}, {num.DecimalFromFloat(4.4), num.DecimalFromFloat(3.3), num.DecimalFromFloat(2.2), num.DecimalFromFloat(1.1)}}},
		Tolerance: num.DecimalFromInt64(3),
	})

	require.False(t, kvb1.WithinTolerance(kvb3))
	require.False(t, kvb3.WithinTolerance(kvb1))
}

func testToDecimal(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key: "scalar value",
		Val: &statevar.DecimalScalar{Val: num.DecimalFromFloat(1.23456)},
	})
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key: "vector value",
		Val: &statevar.DecimalVector{Val: []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}},
	})
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key: "matrix value",
		Val: &statevar.DecimalMatrix{Val: [][]num.Decimal{{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, {num.DecimalFromFloat(4.4), num.DecimalFromFloat(3.3), num.DecimalFromFloat(2.2), num.DecimalFromFloat(1.1)}}},
	})

	kvResult := kvb1.ToDecimal()
	scalarValue := kvResult.KeyDecimalValue["scalar value"]
	switch v := scalarValue.(type) {
	case *statevar.DecimalScalar:
		require.Equal(t, num.DecimalFromFloat(1.23456), v.Val)
	default:
		t.Fail()
	}

	vectorValue := kvResult.KeyDecimalValue["vector value"]
	switch v := vectorValue.(type) {
	case *statevar.DecimalVector:
		require.Equal(t, []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, v.Val)
	default:
		t.Fail()
	}

	matrixValue := kvResult.KeyDecimalValue["matrix value"]
	switch v := matrixValue.(type) {
	case *statevar.DecimalMatrix:
		require.Equal(t, [][]num.Decimal{
			{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)},
			{num.DecimalFromFloat(4.4), num.DecimalFromFloat(3.3), num.DecimalFromFloat(2.2), num.DecimalFromFloat(1.1)},
		},
			v.Val)
	default:
		t.Fail()
	}
}
