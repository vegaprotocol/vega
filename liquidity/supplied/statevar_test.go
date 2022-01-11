package supplied

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/require"
)

func TestRange(t *testing.T) {
	price, prob := calculateRange(num.NewUint(99), num.NewUint(99), num.NewUint(100), num.DecimalFromInt64(1), false, pOfT)
	for _, p := range price {
		println(p.String())
	}
	require.Equal(t, 2, len(price))
	require.Equal(t, num.DecimalFromInt64(99), price[0])
	require.Equal(t, num.DecimalFromInt64(100), price[1])
	require.Equal(t, 2, len(prob))

	price, prob = calculateRange(num.NewUint(100), num.NewUint(100), num.NewUint(300), num.DecimalFromInt64(1), false, pOfT)
	for _, p := range price {
		println(p.String())
	}
	require.Equal(t, 101, len(price))
	require.Equal(t, num.DecimalFromInt64(100), price[0])
	require.Equal(t, num.DecimalFromInt64(300), price[100])
	require.Equal(t, 101, len(prob))
}

func pOfT(best, p *num.Uint, min, max, tauScaled num.Decimal, isBid bool, applyMinMax bool) num.Decimal {
	return num.DecimalFromFloat(0.005)
}

func getBest() (*num.Uint, *num.Uint, error) {
	return num.NewUint(120), num.NewUint(220), nil
}

func getBestError() (*num.Uint, *num.Uint, error) {
	return nil, nil, errors.New("some error")
}

func TestGetProbability(t *testing.T) {
	minProb := num.DecimalFromFloat(0.021)

	// no consensus, best prices returned error, use min probability
	require.Equal(t, minProb, getProbabilityOfTrading(getBestError, &probabilityOfTrading{}, num.DecimalFromInt64(101), true, minProb))
	require.Equal(t, minProb, getProbabilityOfTrading(getBestError, &probabilityOfTrading{}, num.DecimalFromInt64(101), false, minProb))

	// no consensus - price within 100 ticks from the best bid - use default probability
	require.Equal(t, defaultProbability, getProbabilityOfTrading(getBest, &probabilityOfTrading{}, num.DecimalFromInt64(20), true, minProb))

	// no consensus - price within 100 ticks from the best ask - use default probability
	require.Equal(t, defaultProbability, getProbabilityOfTrading(getBest, &probabilityOfTrading{}, num.DecimalFromInt64(320), false, minProb))

	// no consensus - price further than 100 ticks from the best bid - use min probability
	require.Equal(t, minProb, getProbabilityOfTrading(getBest, &probabilityOfTrading{}, num.DecimalFromInt64(19), true, minProb))

	// no consensus - price further than 100 ticks from the best ask - use min probability
	require.Equal(t, minProb, getProbabilityOfTrading(getBest, &probabilityOfTrading{}, num.DecimalFromInt64(321), false, minProb))

	// price is GTE best bid and LTE best ask - use defaultInRangeProbabilityOfTrading
	require.Equal(t, defaultInRangeProbabilityOfTrading, getProbabilityOfTrading(getBest, &probabilityOfTrading{}, num.DecimalFromInt64(120), true, minProb))
	require.Equal(t, defaultInRangeProbabilityOfTrading, getProbabilityOfTrading(getBest, &probabilityOfTrading{}, num.DecimalFromInt64(121), true, minProb))
	require.Equal(t, defaultInRangeProbabilityOfTrading, getProbabilityOfTrading(getBest, &probabilityOfTrading{}, num.DecimalFromInt64(220), false, minProb))
	require.Equal(t, defaultInRangeProbabilityOfTrading, getProbabilityOfTrading(getBest, &probabilityOfTrading{}, num.DecimalFromInt64(219), false, minProb))

	bidPrices := make([]num.Decimal, 0, 51)
	askPrices := make([]num.Decimal, 0, 51)
	bidProb := make([]num.Decimal, 0, 51)
	askProb := make([]num.Decimal, 0, 51)

	// make the price points skip by 4 so we can test linear interpolation meaninfully
	for i := uint64(0); i < 101; i += 4 {
		bidPrices = append(bidPrices, num.Sum(num.NewUint(20), num.NewUint(i)).ToDecimal())
		bidProb = append(bidProb, num.DecimalFromFloat(0.01).Mul(num.DecimalFromInt64(int64(i+1))))
		askPrices = append(askPrices, num.Sum(num.NewUint(220), num.NewUint(i)).ToDecimal())
		askProb = append(askProb, num.DecimalFromInt64(1).Sub(num.DecimalFromFloat(0.01).Mul(num.DecimalFromInt64(int64(i)))))
	}

	pot := &probabilityOfTrading{
		bidPrice:       bidPrices,
		bidProbability: bidProb,
		askPrice:       askPrices,
		askProbability: askProb,
	}

	// find exact match - bid of 20 has probability of 0.01 scaled by 0.5 => 0.005 but minProb > 0.005 => minProb
	require.Equal(t, minProb, getProbabilityOfTrading(getBest, pot, num.DecimalFromInt64(20), true, minProb))

	// find exact match - bid of 20 has probability of 0.05 scaled by 0.5 => 0.025
	require.Equal(t, "0.025", getProbabilityOfTrading(getBest, pot, num.DecimalFromInt64(24), true, minProb).String())

	// find exact match - bid of 116 has probability of 0.97 scaled by 0.5 => 0.485
	require.Equal(t, "0.485", getProbabilityOfTrading(getBest, pot, num.DecimalFromInt64(116), true, minProb).String())

	// find exact match - ask of 316 has probability of 0.04 scaled by 0.5 => 0.02 but minProb > 0.02 => minProb
	require.Equal(t, minProb, getProbabilityOfTrading(getBest, pot, num.DecimalFromInt64(316), false, minProb))

	// find exact match - ask of 280 has probability of 0.4 scaled by 0.5 => 0.2
	require.Equal(t, "0.2", getProbabilityOfTrading(getBest, pot, num.DecimalFromInt64(280), false, minProb).String())

	// find exact match - ask of 224 has probability of 0.96 scaled by 0.5 => 0.48
	require.Equal(t, "0.48", getProbabilityOfTrading(getBest, pot, num.DecimalFromInt64(224), false, minProb).String())

	// bid linear interpolation
	// 0.75*0.09 + 0.25*0.13 = 0.1 => scaled by 0.5 = 0.05 > minProb => 0.05
	require.Equal(t, "0.05", getProbabilityOfTrading(getBest, pot, num.DecimalFromInt64(29), true, minProb).String())
	// 0.5*0.09 + 0.5*0.13 = 0.11 => scaled by 0.5 => 0.055
	require.Equal(t, "0.055", getProbabilityOfTrading(getBest, pot, num.DecimalFromInt64(30), true, minProb).String())
	// 0.25*0.09 + 0.75*0.13 = 0.12 => scaled by 0.5 => 0.06
	require.Equal(t, "0.06", getProbabilityOfTrading(getBest, pot, num.DecimalFromInt64(31), true, minProb).String())

	// ask linear interpolation
	// 0.75*0.8 + 0.25*0.76 = 0.79 => scaled by 0.5 = 0.395 > minProb => 0.395
	require.Equal(t, "0.395", getProbabilityOfTrading(getBest, pot, num.DecimalFromInt64(241), false, minProb).String())
	// 0.5*0.8 + 0.5*0.76 = 0.78 => scaled by 0.5 => 0.39
	require.Equal(t, "0.39", getProbabilityOfTrading(getBest, pot, num.DecimalFromInt64(242), false, minProb).String())
	// 0.25*0.8 + 0.75*0.76 = 0.77 => scaled by 0.5 => 0.385
	require.Equal(t, "0.385", getProbabilityOfTrading(getBest, pot, num.DecimalFromInt64(243), false, minProb).String())
}
