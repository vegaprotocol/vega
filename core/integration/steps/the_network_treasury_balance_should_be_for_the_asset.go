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

func TheNetworkTreasuryBalanceShouldBeForTheAsset(
	broker *stubs.BrokerStub,
	rawAmount, asset string,
) error {
	amount := parseExpectedInsurancePoolBalance(rawAmount)

	acc, err := broker.GetAssetNetworkTreasuryAccount(asset)
	if err != nil {
		return errCannotGetNetworkTreasuryAccountForAsset(asset, err)
	}

	if amount != stringToU64(acc.Balance) {
		return errInvalidAssetNetworkTreasuryBalance(amount, acc)
	}
	return nil
}

func errCannotGetNetworkTreasuryAccountForAsset(asset string, err error) error {
	return fmt.Errorf("couldn't get network treasury account for asset(%s): %s", asset, err.Error())
}

func errInvalidAssetNetworkTreasuryBalance(amount uint64, acc types.Account) error {
	return fmt.Errorf(
		"invalid balance for network treasury, expected %v, got %v",
		amount, acc.Balance,
	)
}
