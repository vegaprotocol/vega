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
	"code.vegaprotocol.io/vega/core/types/num"
)

func TheCumulatedBalanceForAllAccountsShouldBeWorth(broker *stubs.BrokerStub, rawAmount string) error {
	amount, _ := num.UintFromString(rawAmount, 10)

	cumulatedBalance := num.UintZero()
	accounts := broker.GetAccounts()
	for _, v := range accounts {
		// remove vote token
		if v.Asset != "VOTE" {
			b, _ := num.UintFromString(v.Balance, 10)
			cumulatedBalance.AddSum(b)
		}
	}

	if !amount.EQ(cumulatedBalance) {
		return fmt.Errorf("expected cumulated balance to be %v but found %v",
			amount, cumulatedBalance,
		)
	}
	return nil
}
