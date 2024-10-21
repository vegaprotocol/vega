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

package vault_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/vault"
	"code.vegaprotocol.io/vega/core/vault/mocks"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestDeposit(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logging.NewTestLogger()
	col := mocks.NewMockCollateral(ctrl)
	broker := bmocks.NewMockBroker(ctrl)

	vault := vault.NewVaultState(logger, &types.Vault{
		ID:    "1",
		Owner: "zohar",
		Asset: "ETH",
		MetaData: &vega.VaultMetaData{
			Name:        "some meta",
			Description: "no desc",
			Url:         "",
			ImageUrl:    "",
		},
		FeePeriod:            time.Hour * 24,
		ManagementFeeFactor:  num.DecimalZero(),
		PerformanceFeeFactor: num.DecimalZero(),
		CutOffPeriodLength:   5,
		RedemptionDates:      []*types.RedemptionDate{},
	}, col, time.Now(), broker)
	ctx := context.Background()
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.UintZero(), nil).Times(1)
	col.EXPECT().DepositToVault(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&types.LedgerMovement{}, nil).Times(3)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// deposit 100 to the vault, at this point we have only one share holder with 100% of the shares
	require.NoError(t, vault.DepositToVault(ctx, "p1", num.NewUint(150)))
	shares := vault.GetVaultShares()
	require.Equal(t, 1, len(shares))
	require.Equal(t, "1", shares["p1"].String())

	// now deposit 50 to the vault to a new party
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(150), nil).Times(1)
	require.NoError(t, vault.DepositToVault(ctx, "p2", num.NewUint(50)))
	shares = vault.GetVaultShares()
	require.Equal(t, 2, len(shares))
	require.Equal(t, "0.75", shares["p1"].String())
	require.Equal(t, "0.25", shares["p2"].String())

	// finally add another 50 to the vault for p1
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(1)
	require.NoError(t, vault.DepositToVault(ctx, "p1", num.NewUint(50)))
	shares = vault.GetVaultShares()
	require.Equal(t, 2, len(shares))
	require.Equal(t, "0.8", shares["p1"].String())
	require.Equal(t, "0.2", shares["p2"].String())

	// now handle error in get vault balance
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(250), fmt.Errorf("some error")).Times(1)
	require.Equal(t, "some error", vault.DepositToVault(ctx, "p1", num.NewUint(50)).Error())

	// now handle error in collateral deposit
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(250), nil).Times(1)
	col.EXPECT().DepositToVault(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&types.LedgerMovement{}, fmt.Errorf("some error in collateral")).Times(1)
	require.Equal(t, "some error in collateral", vault.DepositToVault(ctx, "p1", num.NewUint(50)).Error())
}

func TestVerifyAndCapAt1(t *testing.T) {
	// total is less than 1 do nothing
	m1 := map[string]num.Decimal{"1": num.DecimalFromFloat(0.1), "2": num.DecimalFromFloat(0.8)}
	vault.VerifyAndCapAt1(m1)
	require.Equal(t, "0.1", m1["1"].String())
	require.Equal(t, "0.8", m1["2"].String())

	// total is equal to 1 do nothing
	m2 := map[string]num.Decimal{"1": num.DecimalFromFloat(0.2), "2": num.DecimalFromFloat(0.8)}
	vault.VerifyAndCapAt1(m2)
	require.Equal(t, "0.2", m2["1"].String())
	require.Equal(t, "0.8", m2["2"].String())

	// total is greater than 1 - adjustment is made to the max
	m3 := map[string]num.Decimal{"1": num.DecimalFromFloat(0.3), "2": num.DecimalFromFloat(0.8)}
	vault.VerifyAndCapAt1(m3)
	require.Equal(t, "0.3", m3["1"].String())
	require.Equal(t, "0.7", m3["2"].String())
}

func TestChangeOwner(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logging.NewTestLogger()
	col := mocks.NewMockCollateral(ctrl)
	broker := bmocks.NewMockBroker(ctrl)

	v := &types.Vault{
		ID:    "1",
		Owner: "zohar",
		Asset: "ETH",
		MetaData: &vega.VaultMetaData{
			Name:        "some meta",
			Description: "no desc",
			Url:         "",
			ImageUrl:    "",
		},
		FeePeriod:            time.Hour * 24,
		ManagementFeeFactor:  num.DecimalZero(),
		PerformanceFeeFactor: num.DecimalZero(),
		CutOffPeriodLength:   5,
		RedemptionDates:      []*types.RedemptionDate{},
	}
	vault := vault.NewVaultState(logger, v, col, time.Now(), broker)
	ctx := context.Background()

	// trying to change the owner giving a party that is not the current owner as the current owner
	require.Error(t, vault.ChangeOwner(ctx, "not zohar", "someone"))

	// giving the right current owner should succeed in changing ownership
	require.NoError(t, vault.ChangeOwner(ctx, "zohar", "someone"))
	require.Equal(t, "someone", v.Owner)
}

func TestUpdateVault(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logging.NewTestLogger()
	col := mocks.NewMockCollateral(ctrl)
	broker := bmocks.NewMockBroker(ctrl)

	now := time.Unix(1729503411, 0)

	v := &types.Vault{
		ID:    "1",
		Owner: "zohar",
		Asset: "ETH",
		MetaData: &vega.VaultMetaData{
			Name:        "some meta",
			Description: "no desc",
			Url:         "",
			ImageUrl:    "",
		},
		FeePeriod:            time.Hour * 24,
		ManagementFeeFactor:  num.DecimalZero(),
		PerformanceFeeFactor: num.DecimalZero(),
		CutOffPeriodLength:   5,
		RedemptionDates: []*types.RedemptionDate{
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(2 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.5),
			},
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(3 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.3),
			},
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(10 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.3),
			},
		},
	}
	vault := vault.NewVaultState(logger, v, col, time.Now(), broker)

	// try to insert a day that is earlier than now should fail
	v1 := &types.Vault{
		ID:    "1",
		Owner: "zohar",
		Asset: "ETH",
		MetaData: &vega.VaultMetaData{
			Name:        "some meta",
			Description: "no desc",
			Url:         "",
			ImageUrl:    "",
		},
		FeePeriod:            time.Hour * 24,
		ManagementFeeFactor:  num.DecimalZero(),
		PerformanceFeeFactor: num.DecimalZero(),
		CutOffPeriodLength:   5,
		RedemptionDates: []*types.RedemptionDate{
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(-24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.5),
			},
		},
	}
	require.Equal(t, "redemptions dates are not allowed to be in the past", vault.UpdateVault(v1, now, 1).Error())

	// trying to remove (or change the date) the next redemption date even that it's after the cutoff
	v2 := &types.Vault{
		ID:    "1",
		Owner: "zohar",
		Asset: "ETH",
		MetaData: &vega.VaultMetaData{
			Name:        "some meta",
			Description: "no desc",
			Url:         "",
			ImageUrl:    "",
		},
		FeePeriod:            time.Hour * 24,
		ManagementFeeFactor:  num.DecimalZero(),
		PerformanceFeeFactor: num.DecimalZero(),
		CutOffPeriodLength:   5,
		RedemptionDates: []*types.RedemptionDate{
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(3 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.3),
			},
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(10 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.3),
			},
		},
	}
	require.Equal(t, "next redemption date is not allowed to change", vault.UpdateVault(v2, now, 1).Error())

	// trying to change the params of the next redemption date (regardless of the cutoff)
	v3_1 := &types.Vault{
		ID:    "1",
		Owner: "zohar",
		Asset: "ETH",
		MetaData: &vega.VaultMetaData{
			Name:        "some meta",
			Description: "no desc",
			Url:         "",
			ImageUrl:    "",
		},
		FeePeriod:            time.Hour * 24,
		ManagementFeeFactor:  num.DecimalZero(),
		PerformanceFeeFactor: num.DecimalZero(),
		CutOffPeriodLength:   5,
		RedemptionDates: []*types.RedemptionDate{
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(2 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.99),
			},
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(3 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.3),
			},
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(10 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.3),
			},
		},
	}
	require.Equal(t, "next redemption date is not allowed to change", vault.UpdateVault(v3_1, now, 1).Error())

	v3_2 := v3_1
	v3_2.RedemptionDates[0].MaxFraction = num.DecimalFromFloat(0.5)
	v3_2.RedemptionDates[0].RedemptionType = types.RedemptionTypeNormal
	require.Equal(t, "next redemption date is not allowed to change", vault.UpdateVault(v3_1, now, 1).Error())

	// try to change dates that are within the cutoff
	// try to change the redemption type
	v4_1 := &types.Vault{
		ID:    "1",
		Owner: "zohar",
		Asset: "ETH",
		MetaData: &vega.VaultMetaData{
			Name:        "some meta",
			Description: "no desc",
			Url:         "",
			ImageUrl:    "",
		},
		FeePeriod:            time.Hour * 24,
		ManagementFeeFactor:  num.DecimalZero(),
		PerformanceFeeFactor: num.DecimalZero(),
		CutOffPeriodLength:   5,
		RedemptionDates: []*types.RedemptionDate{
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(2 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.5),
			},
			{
				RedemptionType: types.RedemptionTypeNormal,
				RedemptionDate: now.Add(3 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.3),
			},
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(10 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.3),
			},
		},
	}
	require.Equal(t, "redemption dates within notice period are not allowed to change", vault.UpdateVault(v4_1, now, 5).Error())

	// try to change the max fraction
	v4_2 := v4_1
	v4_2.RedemptionDates[1].RedemptionType = types.RedemptionTypeFreeCashOnly
	v4_2.RedemptionDates[1].MaxFraction = num.DecimalFromFloat(0.99)
	require.Equal(t, "redemption dates within notice period are not allowed to change", vault.UpdateVault(v4_2, now, 5).Error())

	// try to change the date
	v4_3 := v4_1
	v4_3.RedemptionDates[1].RedemptionType = types.RedemptionTypeFreeCashOnly
	v4_3.RedemptionDates[1].MaxFraction = num.DecimalFromFloat(0.3)
	v4_3.RedemptionDates[1].RedemptionDate = now.Add(4 * 24 * time.Hour)
	require.Equal(t, "redemption dates within notice period are not allowed to change", vault.UpdateVault(v4_3, now, 5).Error())

	// changing the redemption dates outside the cutoff should be fine
	v5_1 := &types.Vault{
		ID:    "1",
		Owner: "zohar",
		Asset: "ETH",
		MetaData: &vega.VaultMetaData{
			Name:        "some meta",
			Description: "no desc",
			Url:         "",
			ImageUrl:    "",
		},
		FeePeriod:            time.Hour * 24,
		ManagementFeeFactor:  num.DecimalZero(),
		PerformanceFeeFactor: num.DecimalZero(),
		CutOffPeriodLength:   5,
		RedemptionDates: []*types.RedemptionDate{
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(2 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.5),
			},
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(3 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.3),
			},
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: now.Add(10 * 24 * time.Hour),
				MaxFraction:    num.DecimalFromFloat(0.3),
			},
		},
	}

	// change type for a redemption date after notice period
	v5_1.RedemptionDates[1].RedemptionType = types.RedemptionTypeNormal
	require.NoError(t, vault.UpdateVault(v5_1, now, 2))

	// change max fraction for a redemption date after notice period
	v5_2 := v5_1
	v5_2.RedemptionDates[1].MaxFraction = num.DecimalFromFloat(0.99)
	require.NoError(t, vault.UpdateVault(v5_2, now, 2))

	v5_3 := v5_2
	v5_3.RedemptionDates[2].RedemptionDate = now.Add(5 * 24 * time.Hour)
	require.NoError(t, vault.UpdateVault(v5_3, now, 2))
}

func TestProcessFees(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logging.NewTestLogger()
	col := mocks.NewMockCollateral(ctrl)
	broker := bmocks.NewMockBroker(ctrl)

	vault := vault.NewVaultState(logger, &types.Vault{
		ID:    "1",
		Owner: "zohar",
		Asset: "ETH",
		MetaData: &vega.VaultMetaData{
			Name:        "some meta",
			Description: "no desc",
			Url:         "",
			ImageUrl:    "",
		},
		FeePeriod:            time.Hour * 24,
		ManagementFeeFactor:  num.DecimalFromFloat(0.02),
		PerformanceFeeFactor: num.DecimalFromFloat(0.01),
		CutOffPeriodLength:   5,
		RedemptionDates:      []*types.RedemptionDate{},
	}, col, time.Now(), broker)
	ctx := context.Background()
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.UintZero(), nil).Times(1)
	col.EXPECT().DepositToVault(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&types.LedgerMovement{}, nil).Times(2)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// deposit 100 to the vault, at this point we have only one share holder with 100% of the shares
	require.NoError(t, vault.DepositToVault(ctx, "p1", num.NewUint(150)))
	shares := vault.GetVaultShares()
	require.Equal(t, 1, len(shares))
	require.Equal(t, "1", shares["p1"].String())

	// now deposit 50 to the vault to a new party
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(150), nil).Times(1)
	require.NoError(t, vault.DepositToVault(ctx, "p2", num.NewUint(50)))
	shares = vault.GetVaultShares()
	require.Equal(t, 2, len(shares))
	require.Equal(t, "0.75", shares["p1"].String())
	require.Equal(t, "0.25", shares["p2"].String())

	// let the total value of the fund at the time of processing fees is 250 (invested=200)
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(250), nil).Times(1)
	vault.ProcessFees(time.Now())

	// management fees = 0.02 * 250 = 5
	// performance fees = 0.01 * (250-200) = 0.5
	// owner share = 5.5/250 = 0.022
	// p1 share = 0.75 * (250-5.5) / 250 = 0.7335
	// p2 share = 0.25 * (250-5.5) / 250 = 0.2445
	shares = vault.GetVaultShares()
	require.Equal(t, 3, len(shares))
	require.Equal(t, "0.022", shares["zohar"].String())
	require.Equal(t, "0.7335", shares["p1"].String())
	require.Equal(t, "0.2445", shares["p2"].String())

	// lets do another round, assume now we had some losses so the total value of the vault is back to 200
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(1)
	vault.ProcessFees(time.Now())

	// no gains so no performance fee
	// management fees = 0.02 * 200 = 4
	// p1 share = 0.7335 * (200-4) / 200 = 0.71883
	// p2 share = 0.2445 * (200-4) / 200 = 0.23961
	shares = vault.GetVaultShares()
	require.Equal(t, 3, len(shares))
	require.Equal(t, "0.04156", shares["zohar"].String())
	require.Equal(t, "0.71883", shares["p1"].String())
	require.Equal(t, "0.23961", shares["p2"].String())
}
