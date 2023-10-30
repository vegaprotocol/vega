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

//go:build qa
// +build qa

package statevar_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/statevar"
	types "code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/require"
)

func TestAddNoise(t *testing.T) {
	tol1 := num.DecimalFromFloat(1e-6)
	tol2 := num.DecimalFromFloat(1e-8)
	vec := []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}
	for i := 0; i < 100; i++ {
		kvb1 := &types.KeyValueBundle{}
		kvb1.KVT = append(kvb1.KVT, types.KeyValueTol{
			Key:       "scalar value",
			Val:       &types.DecimalScalar{Val: num.DecimalFromFloat(1.23456)},
			Tolerance: tol1,
		})
		kvb1.KVT = append(kvb1.KVT, types.KeyValueTol{
			Key:       "vector value",
			Val:       &types.DecimalVector{Val: vec},
			Tolerance: tol2,
		})

		kvb2, _ := types.KeyValueBundleFromProto(statevar.AddNoise(kvb1.ToProto()))
		val1 := kvb2.KVT[0].Val.(*types.DecimalScalar).Val
		require.True(t, val1.Sub(num.DecimalFromFloat(1.23456)).LessThan(tol1))

		val2 := kvb2.KVT[1].Val.(*types.DecimalVector).Val
		for i, v := range val2 {
			require.True(t, v.Sub(vec[i]).LessThan(tol2))
		}

	}
}
