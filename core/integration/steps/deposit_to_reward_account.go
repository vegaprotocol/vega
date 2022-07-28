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
	"context"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/types/num"
)

func DepositToRewardAccount(
	collateralEngine *collateral.Engine,
	table *godog.Table,
	netDeposits *num.Uint,
) error {
	for _, r := range parseRewardDepositTable(table) {
		row := rewardDeposit{row: r}

		rewardAccount, _ := collateralEngine.GetGlobalRewardAccount(row.Asset())
		collateralEngine.IncrementBalance(context.Background(), rewardAccount.ID, row.Amount())
		netDeposits.Add(netDeposits, row.Amount())
	}
	return nil
}

func parseRewardDepositTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"asset",
		"amount",
	}, nil)
}

type rewardDeposit struct {
	row RowWrapper
}

func (r rewardDeposit) Asset() string {
	return r.row.MustStr("asset")
}

func (r rewardDeposit) Amount() *num.Uint {
	return r.row.MustUint("amount")
}
