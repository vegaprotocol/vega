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

func PartyShouldHaveOneMarginAccountPerMarket(
	broker *stubs.BrokerStub,
	owner string,
) error {
	assets := map[string]struct{}{}

	accounts := broker.GetAccounts()

	for _, acc := range accounts {
		if acc.Owner == owner && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			if _, ok := assets[acc.MarketId]; ok {
				return errMultipleMarginAccountForMarket(owner, acc)
			}
			assets[acc.MarketId] = struct{}{}
		}
	}
	return nil
}

func errMultipleMarginAccountForMarket(owner string, acc types.Account) error {
	return fmt.Errorf("party=%v have multiple account for market=%v", owner, acc.MarketId)
}
