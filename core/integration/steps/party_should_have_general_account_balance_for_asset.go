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
	"strconv"

	"code.vegaprotocol.io/vega/core/integration/stubs"
)

func PartyShouldHaveGeneralAccountBalanceForAsset(
	broker *stubs.BrokerStub,
	party, asset, rawBalance string,
) error {
	balance, _ := strconv.ParseUint(rawBalance, 10, 0)
	acc, err := broker.GetPartyGeneralAccount(party, asset)
	if err != nil {
		return err
	}

	if stringToU64(acc.Balance) != balance {
		return fmt.Errorf("invalid general account balance for asset(%s) for party(%s), expected(%d) got(%s)",
			asset, party, balance, acc.Balance,
		)
	}

	return nil
}

func PartyShouldHaveVestingAccountBalanceForAsset(
	broker *stubs.BrokerStub,
	party, asset, rawBalance string,
) error {
	balance, _ := strconv.ParseUint(rawBalance, 10, 0)
	acc, err := broker.GetPartyVestingAccount(party, asset)
	if err != nil {
		return err
	}

	if stringToU64(acc.Balance) != balance {
		return fmt.Errorf("invalid vesting account balance for asset(%s) for party(%s), expected(%d) got(%s)",
			asset, party, balance, acc.Balance,
		)
	}

	return nil
}

func PartyShouldHaveHoldingAccountBalanceForAsset(
	broker *stubs.BrokerStub,
	party, asset, rawBalance string,
) error {
	balance, _ := strconv.ParseUint(rawBalance, 10, 0)
	acc, err := broker.GetPartyHoldingAccount(party, asset)
	if err != nil {
		return err
	}

	if stringToU64(acc.Balance) != balance {
		return fmt.Errorf("invalid holding account balance for asset(%s) for party(%s), expected(%d) got(%s)",
			asset, party, balance, acc.Balance,
		)
	}

	return nil
}
