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
	types "code.vegaprotocol.io/vega/protos/vega"
)

func TheLPLiquidityFeeBalanceShouldBeForTheMarket(
	broker *stubs.BrokerStub,
	party, rawAmount, market string,
) error {
	amount, err := U64(rawAmount)
	panicW("balance", err)

	acc, err := broker.GetMarketLPLiquidityFeePoolAccount(party, market)
	if err != nil {
		return errCannotGetLPLiquidityFeeAccountForPartyInMarket(party, market, err)
	}

	if amount != stringToU64(acc.Balance) {
		return errInvalidBalance(amount, acc)
	}
	return nil
}

func errCannotGetLPLiquidityFeeAccountForPartyInMarket(party, market string, err error) error {
	return fmt.Errorf("couldn't get LP liquidity fee account for party(%s) in market (%s): %s", party, market, err.Error())
}

func errInvalidBalance(amount uint64, acc types.Account) error {
	return fmt.Errorf(
		"invalid balance for party LP liquidity fee account, expected %v, got %v",
		amount, acc.Balance,
	)
}
