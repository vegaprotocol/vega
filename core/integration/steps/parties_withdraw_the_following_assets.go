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
	"fmt"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/cucumber/godog"
)

func PartiesWithdrawTheFollowingAssets(
	collateral *collateral.Engine,
	broker *stubs.BrokerStub,
	netDeposits *num.Uint,
	table *godog.Table,
) error {
	ctx := context.Background()
	for _, r := range parseWithdrawAssetTable(table) {
		row := withdrawAssetRow{row: r}
		amount := row.Amount()
		_, err := collateral.Withdraw(ctx, row.Party(), row.Asset(), amount)
		if err := checkExpectedError(row, err, nil); err != nil {
			return err
		}
		// emit an event manually here as we're not going via the banking flow in integration tests
		broker.Send(events.NewWithdrawalEvent(ctx,
			types.Withdrawal{
				PartyID: row.Party(),
				Asset:   row.Asset(),
				Amount:  amount,
				Status:  types.WithdrawalStatusFinalized,
			}))
		netDeposits = netDeposits.Sub(netDeposits, amount)
	}
	return nil
}

func parseWithdrawAssetTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"asset",
		"amount",
	}, []string{
		"error",
	})
}

type withdrawAssetRow struct {
	row RowWrapper
}

func (r withdrawAssetRow) Party() string {
	return r.row.MustStr("party")
}

func (r withdrawAssetRow) Asset() string {
	return r.row.MustStr("asset")
}

func (r withdrawAssetRow) Amount() *num.Uint {
	return r.row.MustUint("amount")
}

func (r withdrawAssetRow) Reference() string {
	return fmt.Sprintf("%s-%s-%d", r.Party(), r.Party(), r.Amount())
}

func (r withdrawAssetRow) Error() string {
	return r.row.Str("error")
}

func (r withdrawAssetRow) ExpectError() bool {
	return r.row.HasColumn("error")
}
