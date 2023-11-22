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

	stats.RegisterTradingFees("maker-1", "taker-1", num.NewUint(10))
	stats.RegisterTradingFees("maker-1", "taker-2", num.NewUint(20))
	stats.RegisterTradingFees("taker-1", "maker-1", num.NewUint(5))

	expected := map[string]*num.Uint{"maker-1": num.NewUint(35), "taker-1": num.NewUint(15), "taker-2": num.NewUint(20)}
	assert.Equal(t, expected, stats.TotalTradingFeesPerParty())
}
