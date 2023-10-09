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

func TheLPLiquidityBondBalanceShouldBeForTheMarket(
	broker *stubs.BrokerStub,
	party, rawAmount, market string,
) error {
	amount, err := U64(rawAmount)
	panicW("balance", err)

	acc, err := broker.GetMarketLPLiquidityBondAccount(party, market)
	if err != nil {
		return errCannotGetLPLiquidityBondAccountForPartyInMarket(party, market, err)
	}

	if amount != stringToU64(acc.Balance) {
		return errInvalidBalance(amount, acc)
	}
	return nil
}

func errCannotGetLPLiquidityBondAccountForPartyInMarket(party, market string, err error) error {
	return fmt.Errorf("couldn't get LP liquidity bond account for party(%s) in market (%s): %s", party, market, err.Error())
}
