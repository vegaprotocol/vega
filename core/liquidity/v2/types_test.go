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

package liquidity_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/libs/num"

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

	t.Run("Empty provisions", func(t *testing.T) {
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

func TestWeightAverageFee(t *testing.T) {
	got := liquidity.ProvisionsPerParty{
		"p1": {CommitmentAmount: num.NewUint(20), Fee: num.DecimalFromFloat(2.0)},
		"p2": {CommitmentAmount: num.NewUint(60), Fee: num.DecimalFromFloat(1.0)},
	}.FeeForWeightedAverage()

	// (20 * 2) + (60 * 1) / 80 = 1.25
	assert.Equal(t, num.DecimalFromFloat(1.25).String(), got.String())

	// no LPs
	got = liquidity.ProvisionsPerParty{}.FeeForWeightedAverage()
	assert.Equal(t, num.DecimalFromFloat(0).String(), got.String())

	// LPs but all with 0 commitment for whatever reason
	got = liquidity.ProvisionsPerParty{
		"p1": {CommitmentAmount: num.UintZero(), Fee: num.DecimalFromFloat(2.0)},
		"p2": {CommitmentAmount: num.UintZero(), Fee: num.DecimalFromFloat(1.0)},
	}.FeeForWeightedAverage()
	assert.Equal(t, num.DecimalFromFloat(0).String(), got.String())
}
