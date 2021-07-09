package liquidity_test

import (
	"testing"

	"code.vegaprotocol.io/vega/liquidity"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvisionsSatisfyTarget(t *testing.T) {
	commitment := num.NewUint(10)
	parties := liquidity.ProvisionsPerParty{
		"p1": {CommitmentAmount: commitment.Clone(), Fee: num.DecimalFromFloat(10.0)}, // 50
		"p2": {CommitmentAmount: commitment.Clone(), Fee: num.DecimalFromFloat(0.01)}, // 10
		"p3": {CommitmentAmount: commitment.Clone(), Fee: num.DecimalFromFloat(2.00)}, // 30
		"p4": {CommitmentAmount: commitment.Clone(), Fee: num.DecimalFromFloat(3.00)}, // 40
		"p5": {CommitmentAmount: commitment.Clone(), Fee: num.DecimalFromFloat(20.0)}, // 60
		"p6": {CommitmentAmount: commitment.Clone(), Fee: num.DecimalFromFloat(0.10)}, // 20
	}

	tests := []struct {
		stake uint64
		fee   num.Decimal
	}{
		{stake: 1, fee: num.DecimalFromFloat(0.01)},
		{stake: 10, fee: num.DecimalFromFloat(0.01)},
		{stake: 11, fee: num.DecimalFromFloat(0.10)},
		{stake: 30, fee: num.DecimalFromFloat(2.00)},
		{stake: 99, fee: num.DecimalFromFloat(20.0)},
	}

	for i, test := range tests {
		got := parties.FeeForTarget(num.NewUint(test.stake))
		assert.Equal(t, test.fee, got, "Case #%d", i)
	}

	t.Run("EmptyProvisions", func(t *testing.T) {
		parties := liquidity.ProvisionsPerParty{}
		got := parties.FeeForTarget(num.NewUint(100))
		require.True(t, got.IsZero())
	})
}

func TestPartiesTotalStake(t *testing.T) {
	parties := liquidity.ProvisionsPerParty{
		"p1": {CommitmentAmount: num.NewUint(10)}, // 10
		"p2": {CommitmentAmount: num.NewUint(20)}, // 30
		"p3": {CommitmentAmount: num.NewUint(30)}, // 60
		"p4": {CommitmentAmount: num.NewUint(40)}, // 100
		"p5": {CommitmentAmount: num.NewUint(50)}, // 150
		"p6": {CommitmentAmount: num.NewUint(60)}, // 210
	}
	assert.Equal(t, num.NewUint(210), parties.TotalStake())
}
