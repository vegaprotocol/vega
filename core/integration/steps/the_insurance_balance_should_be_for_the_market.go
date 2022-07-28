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

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/core/integration/stubs"
)

func TheInsurancePoolBalanceShouldBeForTheMarket(
	broker *stubs.BrokerStub,
	rawAmount, market string,
) error {
	amount := parseExpectedInsurancePoolBalance(rawAmount)

	acc, err := broker.GetMarketInsurancePoolAccount(market)
	if err != nil {
		return errCannotGetInsurancePoolAccountForMarket(market, err)
	}

	if amount != stringToU64(acc.Balance) {
		return errInvalidMarketInsurancePoolBalance(amount, acc)
	}
	return nil
}

func parseExpectedInsurancePoolBalance(rawAmount string) uint64 {
	amount, err := U64(rawAmount)
	panicW("balance", err)
	return amount
}

func errCannotGetInsurancePoolAccountForMarket(market string, err error) error {
	return fmt.Errorf("couldn't get insurance pool account for market(%s): %s", market, err.Error())
}

func errInvalidMarketInsurancePoolBalance(amount uint64, acc types.Account) error {
	return fmt.Errorf(
		"invalid balance for market insurance pool, expected %v, got %v",
		amount, acc.Balance,
	)
}
