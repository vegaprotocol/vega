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

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/vault"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

func PartiesDepositToVault(vs *vault.VaultService, vault string, table *godog.Table) error {
	for _, r := range parseVaultTxTable(table) {
		row := vaultTxRow{row: r}
		amt := num.MustUintFromString(row.Amount(), 10)
		err := vs.DepositToVault(context.Background(), row.Party(), vault, amt)
		if err == nil && row.row.HasColumn("error") && len(row.row.MustStr("error")) > 0 {
			return fmt.Errorf("expected error (%s) in deposit to vault (%s) by party (%s) but no error found", row.row.MustStr("error"), vault, row.Party())
		} else if err != nil && (!row.row.HasColumn("error") || len(row.row.MustStr("error")) == 0) {
			return fmt.Errorf("unexpected error (%s) in deposit to vault (%s) by party (%s) but no error found", err.Error(), vault, row.Party())
		} else if err != nil && row.row.HasColumn("error") && err.Error() != row.row.MustStr("error") {
			return fmt.Errorf("expected error (%s) in deposit to vault (%s) by party (%s) but got (%s)", row.row.MustStr("error"), vault, row.Party(), err.Error())
		}
	}
	return nil
}

func PartiesWithdrawFromVault(vs *vault.VaultService, vault string, table *godog.Table) error {
	for _, r := range parseVaultTxTable(table) {
		row := vaultTxRow{row: r}
		amt := num.MustUintFromString(row.Amount(), 10)
		err := vs.WithdrawFromVault(context.Background(), crypto.RandomHash(), row.Party(), vault, amt)
		if err == nil && row.row.HasColumn("error") && len(row.row.MustStr("error")) > 0 {
			return fmt.Errorf("expected error (%s) in withdraw from vault (%s) by party (%s) but no error found", row.row.MustStr("error"), vault, row.Party())
		} else if err != nil && (!row.row.HasColumn("error") || len(row.row.MustStr("error")) == 0) {
			return fmt.Errorf("unexpected error (%s) in withdraw from vault (%s) by party (%s) but no error found", err.Error(), vault, row.Party())
		} else if err != nil && row.row.HasColumn("error") && err.Error() != row.row.MustStr("error") {
			return fmt.Errorf("expected error (%s) in withdraw from vault (%s) by party (%s) but got (%s)", row.row.MustStr("error"), vault, row.Party(), err.Error())
		}
	}
	return nil
}

func RedemptionRequestsHasTheFollowingState(broker *stubs.BrokerStub, vs *vault.VaultService, vault string, table *godog.Table) error {
	rrEvents := broker.GetRedemptionRequestEvents()
	for _, r := range parseRedemptionStatusTable(table) {
		row := redemptionRequestRow{row: r}
		found := false
		for _, rre := range rrEvents {
			rr := rre.StreamMessage().GetRedemptionRequest()
			if rr.PartyId == row.Party() && rr.RequestedAmount == row.RequestedAmount() && rr.RemainingAmount == row.RemainingAmount() &&
				rr.Status.String() == row.Status() {
				if r.HasColumn("eligibility date") && row.row.MustI64("eligibility date") != rr.Date {
					continue
				}
				if r.HasColumn("last updated") && row.row.MustI64("last updated") != rr.LastUpdate {
					continue
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("could not find redemption request for vault (%s) with given details (party %s, requested %s, status %s)", vault, row.Party(), row.RequestedAmount(), row.Status())
		}
	}
	return nil
}

type vaultTxRow struct {
	row RowWrapper
}

func (r vaultTxRow) Party() string {
	return r.row.MustStr("party")
}

func (r vaultTxRow) Amount() string {
	return r.row.MustStr("amount")
}

func parseVaultTxTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"asset",
		"amount",
	}, []string{
		"error",
	})
}

func parseRedemptionStatusTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"status",
		"requested amount",
		"remaining amount",
	}, []string{
		"eligibility date",
		"last updated",
	})
}

type redemptionRequestRow struct {
	row RowWrapper
}

func (r redemptionRequestRow) Party() string {
	return r.row.MustStr("party")
}

func (r redemptionRequestRow) Status() string {
	return r.row.MustStr("status")
}

func (r redemptionRequestRow) RequestedAmount() string {
	return r.row.MustStr("requested amount")
}

func (r redemptionRequestRow) RemainingAmount() string {
	return r.row.MustStr("remaining amount")
}
