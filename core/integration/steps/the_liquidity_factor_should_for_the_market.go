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

package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/stubs"
)

func TheLiquidityFeeFactorShouldForTheMarket(
	broker *stubs.BrokerStub,
	feeStr, market string,
) error {
	mkt := broker.GetMarket(market)
	if mkt == nil {
		return fmt.Errorf("invalid market id %v", market)
	}

	got := mkt.Fees.Factors.LiquidityFee
	if got != feeStr {
		return errInvalidLiquidityFeeFactor(market, feeStr, got)
	}

	return nil
}

func errInvalidLiquidityFeeFactor(market string, expected, got string) error {
	return fmt.Errorf("invalid liquidity fee factor for market %s want %s got %s", market, expected, got)
}
