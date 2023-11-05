// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package statevar_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"

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

	scalarValue := kvb1.KVT[0].Val
	switch v := scalarValue.(type) {
	case *statevar.DecimalScalar:
		require.Equal(t, num.DecimalFromFloat(1.23456), v.Val)
	default:
		t.Fail()
	}

	vectorValue := kvb1.KVT[1].Val
	switch v := vectorValue.(type) {
	case *statevar.DecimalVector:
		require.Equal(t, []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, v.Val)
	default:
		t.Fail()
	}

	matrixValue := kvb1.KVT[2].Val
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
