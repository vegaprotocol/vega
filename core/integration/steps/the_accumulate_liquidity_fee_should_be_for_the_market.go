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
)

func TheAccumulatedLiquidityFeesShouldBeForTheMarket(
	broker *stubs.BrokerStub,
	amountStr, market string,
) error {
	amount, err := U64(amountStr)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	acc, err := broker.GetMarketLiquidityFeePoolAccount(market)
	if err != nil {
		return err
	}

	if stringToU64(acc.Balance) != amount {
		return errInvalidAmountInLiquidityFee(market, amount, stringToU64(acc.Balance))
	}

	return nil
}

func errInvalidAmountInLiquidityFee(market string, expected, got uint64) error {
	return fmt.Errorf("invalid amount in liquidity fee pool for market %s want %d got %d", market, expected, got)
}
