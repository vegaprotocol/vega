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

package fee_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
)

func TestFeesStats(t *testing.T) {
	t.Run("test TotalTradingFeesPerParty", testFeesStatsTotalTradingFeesPerParty)
}

func testFeesStatsTotalTradingFeesPerParty(t *testing.T) {
	stats := fee.NewFeesStats()

	stats.RegisterTradingFees("party1", num.NewUint(10))
	stats.RegisterTradingFees("party2", num.NewUint(20))
	stats.RegisterTradingFees("party1", num.NewUint(5))
	stats.RegisterTradingFees("party1", num.NewUint(20))
	stats.RegisterTradingFees("party2", num.NewUint(20))

	expected := map[string]*num.Uint{"party1": num.NewUint(35), "party2": num.NewUint(40)}
	assert.Equal(t, expected, stats.TotalTradingFeesPerParty())
}
