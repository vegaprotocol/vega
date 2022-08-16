// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities_test

import (
	"testing"
)

func TestLiquidityProvision(t *testing.T) {
	t.Run("should parse all valid prices", testParseAllValidPrices)
	t.Run("should return zero for prices if string is empty", testParseEmptyPrices)
	t.Run("should return error if an invalid price string is provided", testParseInvalidPriceString)
	t.Run("should parse valid market data records successfully", testParseMarketDataSuccessfully)
}
