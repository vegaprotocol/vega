package liquidity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvisionsSatisfyTarget(t *testing.T) {
	parties := ProvisionsPerParty{
		"p1": {CommitmentAmount: 10, Fee: "10.0"}, // 50
		"p2": {CommitmentAmount: 10, Fee: "0.01"}, // 10
		"p3": {CommitmentAmount: 10, Fee: "2.00"}, // 30
		"p4": {CommitmentAmount: 10, Fee: "3.00"}, // 40
		"p5": {CommitmentAmount: 10, Fee: "20.0"}, // 60
		"p6": {CommitmentAmount: 10, Fee: "0.10"}, // 20
	}

	tests := []struct {
		stake uint64
		fee   string
	}{
		{stake: 1, fee: "0.01"},
		{stake: 10, fee: "0.01"},
		{stake: 11, fee: "0.10"},
		{stake: 30, fee: "2.00"},
		{stake: 99, fee: "20.0"},
	}

	for i, test := range tests {
		got := parties.FeeForTarget(test.stake)
		assert.Equal(t, test.fee, got, "Case #%d", i)
	}

	t.Run("EmptyProvisions", func(t *testing.T) {
		parties := ProvisionsPerParty{}
		got := parties.FeeForTarget(100)
		require.Empty(t, got)
	})
}

func TestPartiesTotalStake(t *testing.T) {
	parties := ProvisionsPerParty{
		"p1": {CommitmentAmount: 10}, // 10
		"p2": {CommitmentAmount: 20}, // 30
		"p3": {CommitmentAmount: 30}, // 60
		"p4": {CommitmentAmount: 40}, // 100
		"p5": {CommitmentAmount: 50}, // 150
		"p6": {CommitmentAmount: 60}, // 210
	}
	assert.Equal(t, 210, int(parties.TotalStake()))
}
