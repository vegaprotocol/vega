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

package statevar_test

import (
	"testing"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/core/types/num"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"github.com/stretchr/testify/require"
)

func TestDecimalScalar(t *testing.T) {
	t.Run("test equality of two decimal scalars", testDecimalScalarEquality)
	t.Run("test two scalar decimals are within tolerance of each other", testScalarWithinTol)
	t.Run("test converion of decimal scalar to a decimal scalar", testScalarToDecimal)
	t.Run("test conversion to proto", testScalarToProto)
}

// testDecimalScalarEquality tests that given the same key and equal/not equal value, equals function returns the correct value.
func testDecimalScalarEquality(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key: "scalar value",
		Val: &statevar.DecimalScalar{Val: num.DecimalFromFloat(1.23456)},
	})

	kvb2 := &statevar.KeyValueBundle{}
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key: "scalar value",
		Val: &statevar.DecimalScalar{Val: num.DecimalFromFloat(6.54321)},
	})

	require.False(t, kvb1.Equals(kvb2))

	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key: "scalar value",
		Val: &statevar.DecimalScalar{Val: num.DecimalFromFloat(1.23456)},
	})
	require.True(t, kvb1.Equals(kvb3))
}

func testScalarWithinTol(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.DecimalScalar{Val: num.DecimalFromFloat(1.23456)},
		Tolerance: num.DecimalFromInt64(1),
	})

	kvb2 := &statevar.KeyValueBundle{}
	kvb2.KVT = append(kvb2.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.DecimalScalar{Val: num.DecimalFromFloat(6.54321)},
		Tolerance: num.DecimalFromInt64(1),
	})

	require.False(t, kvb1.WithinTolerance(kvb2))

	kvb3 := &statevar.KeyValueBundle{}
	kvb3.KVT = append(kvb3.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.DecimalScalar{Val: num.DecimalFromFloat(2.23456)},
		Tolerance: num.DecimalFromInt64(1),
	})
	require.True(t, kvb1.WithinTolerance(kvb3))
}

// testScalarToDecimal tests conversion to decimal.
func testScalarToDecimal(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.DecimalScalar{Val: num.DecimalFromFloat(1.23456)},
		Tolerance: num.DecimalFromInt64(1),
	})

	res := kvb1.KVT[0].Val
	switch v := res.(type) {
	case *statevar.DecimalScalar:
		require.Equal(t, num.DecimalFromFloat(1.23456), v.Val)
	default:
		t.Fail()
	}
}

func testScalarToProto(t *testing.T) {
	kvb1 := &statevar.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, statevar.KeyValueTol{
		Key:       "scalar value",
		Val:       &statevar.DecimalScalar{Val: num.DecimalFromFloat(1.23456)},
		Tolerance: num.DecimalFromInt64(1),
	})
	res := kvb1.ToProto()
	require.Equal(t, 1, len(res))
	require.Equal(t, "scalar value", res[0].Key)
	require.Equal(t, "1", res[0].Tolerance)
	switch v := res[0].Value.Value.(type) {
	case *vega.StateVarValue_ScalarVal:
		require.Equal(t, "1.23456", v.ScalarVal.Value)
	default:
		t.Fail()
	}

	kvb2, err := statevar.KeyValueBundleFromProto(res)
	require.NoError(t, err)
	require.Equal(t, kvb1, kvb2)
}
