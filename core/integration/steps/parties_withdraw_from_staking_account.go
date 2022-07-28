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
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/integration/stubs"
)

func PartiesWithdrawFromStakingAccount(
	stakingAccountStub *stubs.StakingAccountStub,
	broker *stubs.BrokerStub,
	table *godog.Table,
) error {
	for _, r := range parseDepositAssetTable(table) {
		row := depositAssetRow{row: r}

		err := stakingAccountStub.DecrementBalance(row.Party(), row.Amount())

		if err := checkExpectedError(row, err, nil); err != nil {
			return err
		}
	}
	return nil
}
