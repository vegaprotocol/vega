// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheSettlementAccountShouldHaveBalanceForMarket(
	broker *stubs.BrokerStub,
	amountStr, market string,
) error {
	amount := parseSettlementAccountBalance(amountStr)

	acc, err := broker.GetMarketSettlementAccount(market)
	if err != nil {
		return errCannotGetSettlementAccountForMarket(market, err)
	}

	if amount != stringToU64(acc.Balance) {
		return errInvalidSettlementAccountBalanceForMarket(amount, acc)
	}
	return nil
}

func parseSettlementAccountBalance(amountStr string) uint64 {
	amount, err := U64(amountStr)
	panicW("balance", err)
	return amount
}

func errCannotGetSettlementAccountForMarket(market string, err error) error {
	return fmt.Errorf("couldn't get settlement account for market(%s): %s", market, err.Error())
}

func errInvalidSettlementAccountBalanceForMarket(amount uint64, acc types.Account) error {
	return fmt.Errorf("invalid balance for market settlement account, expected %v, got %v", amount, acc.Balance)
}
