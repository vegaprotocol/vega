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

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

func PartiesDepositTheFollowingAssets(
	collateralEngine *collateral.Engine,
	broker *stubs.BrokerStub,
	netDeposits *num.Uint,
	table *godog.Table,
) error {
	ctx := context.Background()

	for _, r := range parseDepositAssetTable(table) {
		row := depositAssetRow{row: r}
		amount := row.Amount()
		_, err := collateralEngine.Deposit(
			ctx,
			row.Party(),
			row.Asset(),
			amount,
		)
		if err := checkExpectedError(row, err, nil); err != nil {
			return err
		}

		_, err = broker.GetPartyGeneralAccount(row.Party(), row.Asset())
		if err != nil {
			return errNoGeneralAccountForParty(row, err)
		}
		netDeposits.Add(netDeposits, amount)
		// event an email manually here as we're not going via the banking flow in integration tests
		broker.Send(events.NewDepositEvent(ctx,
			types.Deposit{
				PartyID: row.Party(),
				Asset:   row.Asset(),
				Amount:  amount,
				Status:  types.DepositStatusFinalized,
			}))
	}
	return nil
}

func errNoGeneralAccountForParty(party depositAssetRow, err error) error {
	return fmt.Errorf("party(%v) has no general account for asset(%v): %s",
		party.Party(),
		party.Asset(),
		err.Error(),
	)
}

func parseDepositAssetTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"asset",
		"amount",
	}, []string{
		"error",
	})
}

type depositAssetRow struct {
	row RowWrapper
}

func (r depositAssetRow) Party() string {
	return r.row.MustStr("party")
}

func (r depositAssetRow) Asset() string {
	return r.row.MustStr("asset")
}

func (r depositAssetRow) Amount() *num.Uint {
	return r.row.MustUint("amount")
}

func (r depositAssetRow) Error() string {
	return r.row.Str("error")
}

func (r depositAssetRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r depositAssetRow) Reference() string {
	return fmt.Sprintf("%s-%s-%d", r.Party(), r.Party(), r.Amount())
}
