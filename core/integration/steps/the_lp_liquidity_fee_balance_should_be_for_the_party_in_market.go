// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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