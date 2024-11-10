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
	"time"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/vault"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func VaultShouldHaveShareholders(broker *stubs.BrokerStub, vault string, table *godog.Table) error {
	vs := broker.GetVaultState(vault)
	if vs == nil {
		return fmt.Errorf("vault not found")
	}

	for _, r := range parseShareHolderTable(table) {
		row := shareHolderRow{row: r}
		found := false
		for _, ps := range vs.PartyShares {
			if ps.Party == row.Party() {
				found = true
				if ps.Share != row.Share() {
					return fmt.Errorf("expected party (%s) to have a share (%s) in vault (%s) got (%s)", row.Party(), row.Share(), vault, ps.Share)
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("expected party (%s) to have a share (%s) in vault (%s)- share not found ", row.Party(), row.Share(), vault)
		}
	}

	return nil
}

func VaultShouldHaveTheOwner(broker *stubs.BrokerStub, vault, owner string) error {
	vs := broker.GetVaultState(vault)
	if vs == nil {
		return fmt.Errorf("vault not found")
	}
	if vs.Vault.Owner != owner {
		return fmt.Errorf("invalid owner: expected (%s) got (%s)", owner, vs.Vault.Owner)
	}
	return nil
}

func VaultShouldHaveFollowingState(broker *stubs.BrokerStub, vault string, table *godog.Table) error {
	vs := broker.GetVaultState(vault)
	if vs == nil {
		return fmt.Errorf("vault not found")
	}
	for _, r := range parseVaultStateTable(table) {
		row := vaultStateRow{row: r}
		if row.Owner() != vs.Vault.Owner {
			return fmt.Errorf("incorrect owner for vault (%s) - expected (%s) got (%s)", vault, row.Owner(), vs.Vault.Owner)
		}
		if row.Asset() != vs.Vault.Asset {
			return fmt.Errorf("incorrect asset for vault (%s) - expected (%s) got (%s)", vault, row.Asset(), vs.Vault.Asset)
		}
		if row.Status() != vs.Status {
			return fmt.Errorf("incorrect status for vault (%s) - expected (%s) got (%s)", vault, row.Status().String(), vs.Status.String())
		}
		if row.InvestmentAmount() != vs.InvestedAmount {
			return fmt.Errorf("incorrect investment amount for vault (%s) - expected (%s) got (%s)", vault, row.InvestmentAmount(), vs.InvestedAmount)
		}
		if row.row.HasColumn("fee period") {
			if row.row.MustStr("fee period") != vs.Vault.FeePeriod {
				return fmt.Errorf("incorrect fee period for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("fee period"), vs.Vault.FeePeriod)
			}
		}
		if row.row.HasColumn("management fee factor") {
			if row.row.MustStr("management fee factor") != vs.Vault.ManagementFeeFactor {
				return fmt.Errorf("incorrect management fee factor for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("management fee factor"), vs.Vault.ManagementFeeFactor)
			}
		}
		if row.row.HasColumn("performance fee factor") {
			if row.row.MustStr("performance fee factor") != vs.Vault.PerformanceFeeFactor {
				return fmt.Errorf("incorrect performance fee factor for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("performance fee factor"), vs.Vault.PerformanceFeeFactor)
			}
		}
		if row.row.HasColumn("cutoff period length") {
			if row.row.MustI64("cutoff period length") != vs.Vault.CutOffPeriodLength {
				return fmt.Errorf("incorrect cutoff period length for vault (%s) - expected (%d) got (%d)", vault, row.row.MustI64("cutoff period length"), vs.Vault.CutOffPeriodLength)
			}
		}
		if row.row.HasColumn("name") {
			vaultName := "nil"
			if vs.Vault.VaultMetadata != nil {
				vaultName = vs.Vault.VaultMetadata.Name
			}
			if row.row.MustStr("name") != vaultName {
				return fmt.Errorf("incorrect name for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("name"), vaultName)
			}
		}
		if row.row.HasColumn("description") {
			vaultDescription := "nil"
			if vs.Vault.VaultMetadata != nil {
				vaultDescription = vs.Vault.VaultMetadata.Description
			}
			if row.row.MustStr("description") != vaultDescription {
				return fmt.Errorf("incorrect description for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("description"), vaultDescription)
			}
		}
		if row.row.HasColumn("url") {
			vaultURL := "nil"
			if vs.Vault.VaultMetadata != nil {
				vaultURL = vs.Vault.VaultMetadata.Url
			}
			if row.row.MustStr("url") != vaultURL {
				return fmt.Errorf("incorrect url for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("url"), vaultURL)
			}
		}
		if row.row.HasColumn("image url") {
			vaultImageURL := "nil"
			if vs.Vault.VaultMetadata != nil {
				vaultImageURL = vs.Vault.VaultMetadata.ImageUrl
			}
			if row.row.MustStr("image url") != vaultImageURL {
				return fmt.Errorf("incorrect url for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("image url"), vaultImageURL)
			}
		}
		if row.row.HasColumn("next fee calc") {
			if row.row.MustI64("next fee calc") != vs.NextFeeCalc {
				return fmt.Errorf("incorrect next fee calc time for vault (%s) - expected (%d) got (%d)", vault, row.row.MustI64("next fee calc"), vs.NextFeeCalc)
			}
		}
		if row.row.HasColumn("next redemption date") {
			if row.row.MustI64("next redemption date") != vs.NextFeeCalc {
				return fmt.Errorf("incorrect next redemption date for vault (%s) - expected (%d) got (%d)", vault, row.row.MustI64("next redemption date"), vs.NextRedemptionDate)
			}
		}

		if row.row.HasColumn("redemption date1 date") {
			if row.row.MustI64("redemption date1 date") != vs.Vault.RedemptionDates[0].RedemptionDate {
				return fmt.Errorf("incorrect redemption date1 date for vault (%s) - expected (%d) got (%d)", vault, row.row.MustI64("redemption date1 date"), vs.Vault.RedemptionDates[0].RedemptionDate)
			}
		}
		if row.row.HasColumn("redemption date1 max fraction") {
			if row.row.MustStr("redemption date1 max fraction") != vs.Vault.RedemptionDates[0].MaxFraction {
				return fmt.Errorf("incorrect redemption date1 max fraction for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("redemption date1 max fraction"), vs.Vault.RedemptionDates[0].MaxFraction)
			}
		}
		if row.row.HasColumn("redemption date1 type") {
			if row.row.MustStr("redemption date1 type") != vs.Vault.RedemptionDates[0].RedemptionType.String() {
				return fmt.Errorf("incorrect redemption date1 type for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("redemption date1 type"), vs.Vault.RedemptionDates[0].RedemptionType.String())
			}
		}
		if row.row.HasColumn("redemption date2 date") {
			if row.row.MustI64("redemption date2 date") != vs.Vault.RedemptionDates[1].RedemptionDate {
				return fmt.Errorf("incorrect redemption date2 date for vault (%s) - expected (%d) got (%d)", vault, row.row.MustI64("redemption date2 date"), vs.Vault.RedemptionDates[1].RedemptionDate)
			}
		}
		if row.row.HasColumn("redemption date2 max fraction") {
			if row.row.MustStr("redemption date2 max fraction") != vs.Vault.RedemptionDates[1].MaxFraction {
				return fmt.Errorf("incorrect redemption date2 max fraction for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("redemption date2 max fraction"), vs.Vault.RedemptionDates[1].MaxFraction)
			}
		}
		if row.row.HasColumn("redemption date2 type") {
			if row.row.MustStr("redemption date2 type") != vs.Vault.RedemptionDates[1].RedemptionType.String() {
				return fmt.Errorf("incorrect redemption date2 type for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("redemption date2 type"), vs.Vault.RedemptionDates[1].RedemptionType.String())
			}
		}
		if row.row.HasColumn("redemption date3 date") {
			if row.row.MustI64("redemption date3 date") != vs.Vault.RedemptionDates[2].RedemptionDate {
				return fmt.Errorf("incorrect redemption date3 date for vault (%s) - expected (%d) got (%d)", vault, row.row.MustI64("redemption date3 date"), vs.Vault.RedemptionDates[2].RedemptionDate)
			}
		}
		if row.row.HasColumn("redemption date3 max fraction") {
			if row.row.MustStr("redemption date3 max fraction") != vs.Vault.RedemptionDates[2].MaxFraction {
				return fmt.Errorf("incorrect redemption date3 max fraction for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("redemption date3 max fraction"), vs.Vault.RedemptionDates[2].MaxFraction)
			}
		}
		if row.row.HasColumn("redemption date3 type") {
			if row.row.MustStr("redemption date3 type") != vs.Vault.RedemptionDates[2].RedemptionType.String() {
				return fmt.Errorf("incorrect redemption date1 type for vault (%s) - expected (%s) got (%s)", vault, row.row.MustStr("redemption date3 type"), vs.Vault.RedemptionDates[2].RedemptionType.String())
			}
		}
	}

	return nil
}

func VaultChangeOwnership(vs *vault.VaultService, vault, currentOwner, newOwner string) error {
	return vs.ChangeVaultOwnership(context.Background(), vault, currentOwner, newOwner)
}

func PartiesUpdateVault(vs *vault.VaultService, table *godog.Table) error {
	for _, r := range parseVaultTable(table) {
		row := vaultRow{row: r}
		vault := prepareVaultFromTable(row)
		err := vs.UpdateVault(context.Background(), vault)
		var errRow ErroneousRow
		if errRow == nil || row.ExpectError() {
			errRow = row
		}
		if ceerr := checkExpectedError(errRow, err, errUpdateVault(vault, err)); ceerr != nil {
			return ceerr
		}
	}
	return nil
}

func PartiesCreateVault(vs *vault.VaultService, table *godog.Table) error {
	for _, r := range parseVaultTable(table) {
		row := vaultRow{row: r}
		vault := prepareVaultFromTable(row)
		err := vs.CreateVault(context.Background(), vault)
		var errRow ErroneousRow
		if errRow == nil || row.ExpectError() {
			errRow = row
		}
		if ceerr := checkExpectedError(errRow, err, errNewVault(vault, err)); ceerr != nil {
			return ceerr
		}
	}
	return nil
}

func prepareVaultFromTable(row vaultRow) *types.Vault {
	vault := &types.Vault{
		ID:                   row.ID(),
		Owner:                row.Owner(),
		Asset:                row.Asset(),
		FeePeriod:            row.FeePeriod(),
		ManagementFeeFactor:  row.ManagementFeeFactor(),
		PerformanceFeeFactor: row.PerformanceFeeFactorFeeFactor(),
		CutOffPeriodLength:   row.CutOffPeriodLength(),
	}

	var metadata *vega.VaultMetaData
	if row.row.HasColumn("name") {
		if metadata == nil {
			metadata = &vega.VaultMetaData{
				Name: row.row.mustColumn("name"),
			}
		} else {
			metadata.Name = row.row.mustColumn("name")
		}
	}
	if row.row.HasColumn("description") {
		if metadata == nil {
			metadata = &vega.VaultMetaData{
				Description: row.row.mustColumn("description"),
			}
		} else {
			metadata.Description = row.row.mustColumn("description")
		}
	}
	if row.row.HasColumn("url") {
		if metadata == nil {
			metadata = &vega.VaultMetaData{
				Url: row.row.mustColumn("url"),
			}
		} else {
			metadata.Url = row.row.mustColumn("url")
		}
	}
	if row.row.HasColumn("image url") {
		if metadata == nil {
			metadata = &vega.VaultMetaData{
				ImageUrl: row.row.mustColumn("image url"),
			}
		} else {
			metadata.ImageUrl = row.row.mustColumn("image url")
		}
	}

	if metadata != nil {
		vault.MetaData = metadata
	}

	redemptionDates := []*types.RedemptionDate{}
	redemptionDates = append(redemptionDates, &types.RedemptionDate{
		RedemptionType: vega.RedemptionType(row.row.MustI32("redemption date1 type")),
		RedemptionDate: time.Unix(row.row.MustI64("redemption date1 date"), 0),
		MaxFraction:    row.row.MustDecimal("redemption date1 max fraction"),
	})

	if row.row.HasColumn("redemption date2 type") {
		redemptionDates = append(redemptionDates, &types.RedemptionDate{
			RedemptionType: vega.RedemptionType(row.row.MustI32("redemption date2 type")),
			RedemptionDate: time.Unix(row.row.MustI64("redemption date2 date"), 0),
			MaxFraction:    row.row.MustDecimal("redemption date2 max fraction"),
		})
	}

	if row.row.HasColumn("redemption date3 type") {
		redemptionDates = append(redemptionDates, &types.RedemptionDate{
			RedemptionType: vega.RedemptionType(row.row.MustI32("redemption date3 type")),
			RedemptionDate: time.Unix(row.row.MustI64("redemption date3 date"), 0),
			MaxFraction:    row.row.MustDecimal("redemption date3 max fraction"),
		})
	}

	vault.RedemptionDates = redemptionDates
	return vault
}

func errNewVault(v *types.Vault, err error) error {
	return fmt.Errorf("failed to create vault [%s] for party %s: %v", v.ID, v.Owner, err)
}

func errUpdateVault(v *types.Vault, err error) error {
	return fmt.Errorf("failed to update vault [%s] for party %s: %v", v.ID, v.Owner, err)
}

func parseVaultTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"owner",
		"asset",
		"fee period",
		"management fee factor",
		"performance fee factor",
		"cutoff period length",
		"redemption date1 type",
		"redemption date1 date",
		"redemption date1 max fraction",
	}, []string{
		"name",
		"description",
		"url",
		"image url",
		"redemption date2 type",
		"redemption date2 date",
		"redemption date2 max fraction",
		"redemption date3 type",
		"redemption date3 date",
		"redemption date3 max fraction",
		"error",
	})
}

type vaultRow struct {
	row RowWrapper
}

func (r vaultRow) ID() string {
	return r.row.MustStr("id")
}

func (r vaultRow) Owner() string {
	return r.row.MustStr("owner")
}

func (r vaultRow) Asset() string {
	return r.row.MustStr("asset")
}

func (r vaultRow) FeePeriod() time.Duration {
	return r.row.MustDuration("fee period")
}

func (r vaultRow) ManagementFeeFactor() num.Decimal {
	return r.row.MustDecimal("management fee factor")
}

func (r vaultRow) PerformanceFeeFactorFeeFactor() num.Decimal {
	return r.row.MustDecimal("performance fee factor")
}

func (r vaultRow) CutOffPeriodLength() int64 {
	return r.row.MustI64("cutoff period length")
}

func (r vaultRow) Error() string {
	return r.row.Str("error")
}

func (r vaultRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r vaultRow) Reference() string {
	return r.row.MustStr("id")
}

func parseShareHolderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"share",
	}, []string{})
}

type shareHolderRow struct {
	row RowWrapper
}

func (r shareHolderRow) Party() string {
	return r.row.MustStr("party")
}

func (r shareHolderRow) Share() string {
	return r.row.MustStr("share")
}

func parseVaultStateTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"owner",
		"asset",
		"state",
		"investment amount",
	}, []string{
		"fee period",
		"management fee factor",
		"performance fee factor",
		"cutoff period length",
		"name",
		"description",
		"url",
		"image url",
		"redemption date1 type",
		"redemption date1 date",
		"redemption date1 max fraction",
		"redemption date2 type",
		"redemption date2 date",
		"redemption date2 max fraction",
		"redemption date3 type",
		"redemption date3 date",
		"redemption date3 max fraction",
		"next fee calc",
		"next redemption date",
	})
}

type vaultStateRow struct {
	row RowWrapper
}

func (r vaultStateRow) Owner() string {
	return r.row.MustStr("owner")
}

func (r vaultStateRow) Asset() string {
	return r.row.MustStr("asset")
}

func (r vaultStateRow) Status() vega.VaultStatus {
	return vega.VaultStatus(vega.VaultStatus_value[r.row.MustStr("status")])
}

func (r vaultStateRow) InvestmentAmount() string {
	return r.row.MustStr("investment amount")
}
