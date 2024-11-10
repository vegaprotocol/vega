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

package sqlstore_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/require"
)

func setupVaultTest(t *testing.T) *sqlstore.Vault {
	t.Helper()
	plbs := sqlstore.NewVault(connectionSource)

	return plbs
}

func TestVaultState(t *testing.T) {
	vs := setupVaultTest(t)

	const (
		vault1 = "70432aa1dc6bc20a9b404d30f23e6a8def11a1692609dcef0ad8dc558d9df7db"
		party1 = "a696300fec90755c90e2489af68fe2dfede5744184711ea3acde0ca55ae19585"
		party2 = "2e7a16d9ef690f0d2beed115fba13ba2aaa16c8f971910ad88c72b9db010c7d4"
		party3 = "dfe522e234d67e6ae3f017859f898e576b3928ea57310b765398615a0d3fde2f"
	)

	ctx := tempTransaction(t)

	t.Run("return error if do not exists", func(t *testing.T) {
		_, err := vs.GetByVaultID(ctx, vault1)
		require.EqualError(t, err, "no resource corresponding to this id")
	})

	now := time.Now().Truncate(time.Millisecond)

	t.Run("can insert successfully", func(t *testing.T) {
		w := entities.VaultState{
			VaultID: entities.VaultID(vault1),
			Vault: &vega.Vault{
				VaultId:              vault1,
				Owner:                party1,
				Asset:                "some asset",
				FeePeriod:            "24h:00m",
				ManagementFeeFactor:  "0.5",
				PerformanceFeeFactor: "0.2",
				CutOffPeriodLength:   3,
			},
			PartyShares: []*entities.VaultPartyShare{
				{PartyID: entities.PartyID(party1), VaultID: entities.VaultID(vault1), Share: num.DecimalOne(), VegaTime: now},
			},
			InvestedAmount:     num.DecimalFromInt64(1000),
			Status:             entities.VaultStatus(vega.VaultStatus_VAULT_STATUS_STOPPING),
			NextFeeCalc:        now.Add(10 * time.Minute),
			NextRedemptionDate: now.Add(24 * time.Hour),
			VegaTime:           now,
		}

		require.NoError(t, vs.Add(ctx, &w))

		vault, err := vs.GetByVaultID(ctx, vault1)
		require.NoError(t, err)
		require.Equal(t, 1, len(vault.PartyShares))
		require.Equal(t, party1, vault.PartyShares[0].PartyID.String())
		require.Equal(t, "1", vault.PartyShares[0].Share.String())
		require.Equal(t, vault1, vault.PartyShares[0].VaultID.String())

		require.Equal(t, vault1, vault.VaultID.String())
		require.Equal(t, "1000", vault.InvestedAmount.String())
		require.Equal(t, now, vault.VegaTime)
		require.Equal(t, now.Add(24*time.Hour), vault.NextRedemptionDate)
		require.Equal(t, now.Add(10*time.Minute), vault.NextFeeCalc)
		require.Equal(t, entities.VaultStatus(vega.VaultStatus_VAULT_STATUS_STOPPING), vault.Status)

		require.Equal(t, vault1, vault.Vault.VaultId)
		require.Equal(t, party1, vault.Vault.Owner)
		require.Equal(t, "some asset", vault.Vault.Asset)
		require.Equal(t, "24h:00m", vault.Vault.FeePeriod)
		require.Equal(t, "0.5", vault.Vault.ManagementFeeFactor)
		require.Equal(t, "0.2", vault.Vault.PerformanceFeeFactor)
		require.Equal(t, int64(3), vault.Vault.CutOffPeriodLength)
	})

	now = now.Add(24 * time.Hour).Truncate(time.Millisecond)

	t.Run("can replace exisisting values", func(t *testing.T) {
		w := entities.VaultState{
			VaultID: entities.VaultID(vault1),
			Vault: &vega.Vault{
				VaultId:              vault1,
				Owner:                party1,
				Asset:                "some asset",
				FeePeriod:            "24h:00m",
				ManagementFeeFactor:  "0.7",
				PerformanceFeeFactor: "0.5",
				CutOffPeriodLength:   4,
			},
			PartyShares: []*entities.VaultPartyShare{
				{PartyID: entities.PartyID(party1), VaultID: entities.VaultID(vault1), Share: num.DecimalFromFloat(0.5), VegaTime: now},
				{PartyID: entities.PartyID(party2), VaultID: entities.VaultID(vault1), Share: num.DecimalFromFloat(0.3), VegaTime: now},
				{PartyID: entities.PartyID(party3), VaultID: entities.VaultID(vault1), Share: num.DecimalFromFloat(0.2), VegaTime: now},
			},
			InvestedAmount:     num.DecimalFromInt64(3000),
			Status:             entities.VaultStatus(vega.VaultStatus_VAULT_STATUS_ACTIVE),
			NextFeeCalc:        now.Add(10 * time.Minute),
			NextRedemptionDate: now.Add(24 * time.Hour),
			VegaTime:           now,
		}
		require.NoError(t, vs.Add(ctx, &w))
		vault, err := vs.GetByVaultID(ctx, vault1)
		require.NoError(t, err)
		require.Equal(t, 3, len(vault.PartyShares))
		require.Equal(t, party1, vault.PartyShares[0].PartyID.String())
		require.Equal(t, "0.5", vault.PartyShares[0].Share.String())
		require.Equal(t, vault1, vault.PartyShares[0].VaultID.String())
		require.Equal(t, party2, vault.PartyShares[1].PartyID.String())
		require.Equal(t, "0.3", vault.PartyShares[1].Share.String())
		require.Equal(t, vault1, vault.PartyShares[1].VaultID.String())
		require.Equal(t, party3, vault.PartyShares[2].PartyID.String())
		require.Equal(t, "0.2", vault.PartyShares[2].Share.String())
		require.Equal(t, vault1, vault.PartyShares[2].VaultID.String())

		require.Equal(t, vault1, vault.VaultID.String())
		require.Equal(t, "3000", vault.InvestedAmount.String())
		require.Equal(t, now, vault.VegaTime)
		require.Equal(t, now.Add(24*time.Hour), vault.NextRedemptionDate)
		require.Equal(t, now.Add(10*time.Minute), vault.NextFeeCalc)
		require.Equal(t, entities.VaultStatus(vega.VaultStatus_VAULT_STATUS_ACTIVE), vault.Status)

		require.Equal(t, vault1, vault.Vault.VaultId)
		require.Equal(t, party1, vault.Vault.Owner)
		require.Equal(t, "some asset", vault.Vault.Asset)
		require.Equal(t, "24h:00m", vault.Vault.FeePeriod)
		require.Equal(t, "0.7", vault.Vault.ManagementFeeFactor)
		require.Equal(t, "0.5", vault.Vault.PerformanceFeeFactor)
		require.Equal(t, int64(4), vault.Vault.CutOffPeriodLength)
	})
}

func TestListVaults(t *testing.T) {
	vs := setupVaultTest(t)

	const (
		vault1 = "70432aa1dc6bc20a9b404d30f23e6a8def11a1692609dcef0ad8dc558d9df7db"
		vault2 = "a696300fec90755c90e2489af68fe2dfede5744184711ea3acde0ca55ae19585"
		vault3 = "2e7a16d9ef690f0d2beed115fba13ba2aaa16c8f971910ad88c72b9db010c7d4"
		party1 = "dfe522e234d67e6ae3f017859f898e576b3928ea57310b765398615a0d3fde2f"
	)
	ctx := tempTransaction(t)
	now := time.Now().Truncate(time.Millisecond)
	w1 := entities.VaultState{
		VaultID: entities.VaultID(vault1),
		Vault: &vega.Vault{
			VaultId:              vault1,
			Owner:                party1,
			Asset:                "some asset 1",
			FeePeriod:            "24h:00m",
			ManagementFeeFactor:  "0.5",
			PerformanceFeeFactor: "0.2",
			CutOffPeriodLength:   3,
		},
		PartyShares:        []*entities.VaultPartyShare{},
		InvestedAmount:     num.DecimalZero(),
		Status:             entities.VaultStatus(vega.VaultStatus_VAULT_STATUS_STOPPED),
		NextFeeCalc:        time.Time{},
		NextRedemptionDate: time.Time{},
		VegaTime:           now,
	}

	require.NoError(t, vs.Add(ctx, &w1))
	now = now.Add(2 * time.Hour)
	w2 := entities.VaultState{
		VaultID: entities.VaultID(vault2),
		Vault: &vega.Vault{
			VaultId:              vault2,
			Owner:                party1,
			Asset:                "some asset 2",
			FeePeriod:            "24h:00m",
			ManagementFeeFactor:  "0.25",
			PerformanceFeeFactor: "0.1",
			CutOffPeriodLength:   2,
		},
		PartyShares: []*entities.VaultPartyShare{
			{PartyID: entities.PartyID(party1), VaultID: entities.VaultID(vault2), Share: num.DecimalOne(), VegaTime: now},
		},
		InvestedAmount:     num.DecimalFromInt64(1000),
		Status:             entities.VaultStatus(vega.VaultStatus_VAULT_STATUS_STOPPING),
		NextFeeCalc:        now.Add(10 * time.Minute),
		NextRedemptionDate: now.Add(24 * time.Hour),
		VegaTime:           now,
	}

	require.NoError(t, vs.Add(ctx, &w2))
	now = now.Add(2 * time.Hour)
	w3 := entities.VaultState{
		VaultID: entities.VaultID(vault3),
		Vault: &vega.Vault{
			VaultId:              vault3,
			Owner:                party1,
			Asset:                "some asset 3",
			FeePeriod:            "24h:00m",
			ManagementFeeFactor:  "0.125",
			PerformanceFeeFactor: "0.05",
			CutOffPeriodLength:   1,
		},
		PartyShares: []*entities.VaultPartyShare{
			{PartyID: entities.PartyID(party1), VaultID: entities.VaultID(vault3), Share: num.DecimalOne(), VegaTime: now},
		},
		InvestedAmount:     num.DecimalFromInt64(1000),
		Status:             entities.VaultStatus(vega.VaultStatus_VAULT_STATUS_ACTIVE),
		NextFeeCalc:        now.Add(10 * time.Minute),
		NextRedemptionDate: now.Add(24 * time.Hour),
		VegaTime:           now,
	}

	require.NoError(t, vs.Add(ctx, &w3))

	vaults, _, err := vs.ListVaultsWithCursor(ctx, []string{}, []string{}, false, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 3, len(vaults))
	require.Equal(t, w3.VaultID.String(), vaults[0].VaultID.String())
	require.Equal(t, w3.Vault.String(), vaults[0].Vault.String())
	require.Equal(t, w3.InvestedAmount.String(), vaults[0].InvestedAmount.String())
	require.Equal(t, w3.NextFeeCalc.UnixNano(), vaults[0].NextFeeCalc.UnixNano())
	require.Equal(t, w3.NextRedemptionDate.UnixNano(), vaults[0].NextRedemptionDate.UnixNano())
	require.Equal(t, w3.Status, vaults[0].Status)
	require.Equal(t, w3.PartyShares[0], vaults[0].PartyShares[0])

	require.Equal(t, w1.VaultID.String(), vaults[1].VaultID.String())
	require.Equal(t, w1.Vault.String(), vaults[1].Vault.String())
	require.Equal(t, w1.InvestedAmount.String(), vaults[1].InvestedAmount.String())
	require.Equal(t, w1.NextFeeCalc.UnixNano(), vaults[1].NextFeeCalc.UnixNano())
	require.Equal(t, w1.NextRedemptionDate.UnixNano(), vaults[1].NextRedemptionDate.UnixNano())
	require.Equal(t, w1.Status, vaults[1].Status)

	require.Equal(t, w2.VaultID.String(), vaults[2].VaultID.String())
	require.Equal(t, w2.Vault.String(), vaults[2].Vault.String())
	require.Equal(t, w2.InvestedAmount.String(), vaults[2].InvestedAmount.String())
	require.Equal(t, w2.NextFeeCalc.UnixNano(), vaults[2].NextFeeCalc.UnixNano())
	require.Equal(t, w2.NextRedemptionDate.UnixNano(), vaults[2].NextRedemptionDate.UnixNano())
	require.Equal(t, w2.Status, vaults[2].Status)
	require.Equal(t, w2.PartyShares[0], vaults[2].PartyShares[0])

	vaults, _, err = vs.ListVaultsWithCursor(ctx, []string{}, []string{}, true, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(vaults))
	require.Equal(t, w3.VaultID.String(), vaults[0].VaultID.String())
	require.Equal(t, w3.Vault.String(), vaults[0].Vault.String())
	require.Equal(t, w3.InvestedAmount.String(), vaults[0].InvestedAmount.String())
	require.Equal(t, w3.NextFeeCalc.UnixNano(), vaults[0].NextFeeCalc.UnixNano())
	require.Equal(t, w3.NextRedemptionDate.UnixNano(), vaults[0].NextRedemptionDate.UnixNano())
	require.Equal(t, w3.Status, vaults[0].Status)
	require.Equal(t, w3.PartyShares[0], vaults[0].PartyShares[0])

	require.Equal(t, w2.VaultID.String(), vaults[1].VaultID.String())
	require.Equal(t, w2.Vault.String(), vaults[1].Vault.String())
	require.Equal(t, w2.InvestedAmount.String(), vaults[1].InvestedAmount.String())
	require.Equal(t, w2.NextFeeCalc.UnixNano(), vaults[1].NextFeeCalc.UnixNano())
	require.Equal(t, w2.NextRedemptionDate.UnixNano(), vaults[1].NextRedemptionDate.UnixNano())
	require.Equal(t, w2.Status, vaults[1].Status)
	require.Equal(t, w2.PartyShares[0], vaults[1].PartyShares[0])

	vaults, _, err = vs.ListVaultsWithCursor(ctx, []string{w3.VaultID.String()}, []string{}, false, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 1, len(vaults))
	require.Equal(t, w3.VaultID.String(), vaults[0].VaultID.String())
	require.Equal(t, w3.Vault.String(), vaults[0].Vault.String())
	require.Equal(t, w3.InvestedAmount.String(), vaults[0].InvestedAmount.String())
	require.Equal(t, w3.NextFeeCalc.UnixNano(), vaults[0].NextFeeCalc.UnixNano())
	require.Equal(t, w3.NextRedemptionDate.UnixNano(), vaults[0].NextRedemptionDate.UnixNano())
	require.Equal(t, w3.Status, vaults[0].Status)
	require.Equal(t, w3.PartyShares[0], vaults[0].PartyShares[0])

	vaults, _, err = vs.ListVaultsWithCursor(ctx, []string{}, []string{w1.Vault.Asset}, false, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 1, len(vaults))
	require.Equal(t, w1.VaultID.String(), vaults[0].VaultID.String())
	require.Equal(t, w1.Vault.String(), vaults[0].Vault.String())
	require.Equal(t, w1.InvestedAmount.String(), vaults[0].InvestedAmount.String())
	require.Equal(t, w1.NextFeeCalc.UnixNano(), vaults[0].NextFeeCalc.UnixNano())
	require.Equal(t, w1.NextRedemptionDate.UnixNano(), vaults[0].NextRedemptionDate.UnixNano())
	require.Equal(t, w1.Status, vaults[0].Status)
	require.Equal(t, 0, len(vaults[0].PartyShares))

	vaults, _, err = vs.ListVaultsWithCursor(ctx, []string{}, []string{w1.Vault.Asset}, true, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 0, len(vaults))
}
