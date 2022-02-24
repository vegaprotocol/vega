//go:build qa
// +build qa

package statevar_test

import (
	"testing"

	"code.vegaprotocol.io/vega/statevar"
	"code.vegaprotocol.io/vega/types/num"
	types "code.vegaprotocol.io/vega/types/statevar"
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
