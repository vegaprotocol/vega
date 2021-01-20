package execution_test

import (
	"testing"

	"code.vegaprotocol.io/vega/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEquityShares(t *testing.T) {
	t.Run("AverageEntryValuation", testAverageEntryValuation)
	t.Run("Shares", testShares)
}

// TestEquitySharesAverageEntryValuation is based on the spec example:
// https://github.com/vegaprotocol/product/blob/02af55e048a92a204e9ee7b7ae6b4475a198c7ff/specs/0042-setting-fees-and-rewarding-lps.md#calculating-liquidity-provider-equity-like-share
func testAverageEntryValuation(t *testing.T) {
	es := execution.NewEquityShares(100)

	es.SetPartyStake("LP1", 100)
	require.EqualValues(t, 100, es.AvgEntryValuation("LP1"))

	es.SetPartyStake("LP1", 200)
	require.EqualValues(t, 100, es.AvgEntryValuation("LP1"))

	es.WithMVP(200).SetPartyStake("LP2", 200)
	require.EqualValues(t, 200, es.AvgEntryValuation("LP2"))
	require.EqualValues(t, 100, es.AvgEntryValuation("LP1"))

	es.WithMVP(400).SetPartyStake("LP1", 300)
	require.EqualValues(t, 120, es.AvgEntryValuation("LP1"))

	es.SetPartyStake("LP1", 1)
	require.EqualValues(t, 120, es.AvgEntryValuation("LP1"))
	require.EqualValues(t, 200, es.AvgEntryValuation("LP2"))
}

func testShares(t *testing.T) {
	var (
		lp1, lp2, lp3 float64
		err           error

		oneSixth  = 1.0 / 6
		oneThird  = 1.0 / 3
		twoThirds = 2.0 / 3
		half      = 1.0 / 2
	)

	es := execution.NewEquityShares(100)

	// Set LP1
	es.SetPartyStake("LP1", 100)
	t.Run("LP1", func(t *testing.T) {
		lp1, err = es.Shares("LP1")
		require.NoError(t, err)
		assert.Equal(t, 1.0, lp1)
	})

	// Set LP2
	es.SetPartyStake("LP2", 200)
	t.Run("LP2", func(t *testing.T) {
		lp1, err = es.Shares("LP1")
		require.NoError(t, err)

		lp2, err = es.Shares("LP2")
		require.NoError(t, err)

		assert.Equal(t, oneThird, lp1)
		assert.Equal(t, twoThirds, lp2)
		assert.Equal(t, 1.0, lp1+lp2)
	})

	// Set LP3
	es.SetPartyStake("LP3", 300)
	t.Run("LP3", func(t *testing.T) {
		lp1, err = es.Shares("LP1")
		require.NoError(t, err)

		lp2, err = es.Shares("LP2")
		require.NoError(t, err)

		lp3, err = es.Shares("LP3")
		require.NoError(t, err)

		assert.Equal(t, oneSixth, lp1)
		assert.Equal(t, oneThird, lp2)
		assert.Equal(t, half, lp3)
		assert.Equal(t, 1.0, lp1+lp2+lp3)
	})
}
