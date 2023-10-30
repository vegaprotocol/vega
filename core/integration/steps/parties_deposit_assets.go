// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
		res, err := collateralEngine.Deposit(
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
		// emit an event manually here as we're not going via the banking flow in integration tests
		broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{res}))
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
