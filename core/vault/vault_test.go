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
	m3 := map[string]num.Decimal{"1": num.MustDecimalFromString("0.3"), "2": num.MustDecimalFromString("0.8")}
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

	// deposit 150 to the vault, at this point we have only one share holder with 100% of the shares
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

func TestUpdateSharesOnRedeem(t *testing.T) {
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

	// deposit 150 to the vault, at this point we have only one share holder with 100% of the shares
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

	// we have a balance of 200 with 3/4 - 1/4 shares to p1 and p2
	// now party p1 wants to withdraw 100
	// the balance of the vault after this should be 100 and the shares should be updated to 50-50
	col.EXPECT().WithdrawFromVault(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&types.LedgerMovement{}, nil).Times(1)
	require.NoError(t, vault.UpdateSharesOnRedeem(ctx, num.NewUint(200), "p1", num.NewUint(100)))

	shares = vault.GetVaultShares()
	require.Equal(t, 2, len(shares))
	require.Equal(t, "0.5", shares["p1"].String())
	require.Equal(t, "0.5", shares["p2"].String())

	// now let p2 withdraw 20 out of the remaining 100
	// p1 has 50/80 = 0.625
	// p2 has 30/80 = 0.375

	col.EXPECT().WithdrawFromVault(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&types.LedgerMovement{}, nil).Times(1)
	require.NoError(t, vault.UpdateSharesOnRedeem(ctx, num.NewUint(100), "p2", num.NewUint(20)))

	shares = vault.GetVaultShares()
	require.Equal(t, 2, len(shares))
	require.Equal(t, "0.625", shares["p1"].String())
	require.Equal(t, "0.375", shares["p2"].String())

	// because in the test cases above we assumed there were no gains and losses, investment amount should reflect the balance
	// of the vault. Lets setup now a case where we have gains, so the investment amount is < vault total balance
	// the current investment amount is 75, let say the actual balance of the vault is 150 now with the share holding as above,
	// p1 wants to withdraw a 80

	col.EXPECT().WithdrawFromVault(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&types.LedgerMovement{}, nil).Times(1)
	require.NoError(t, vault.UpdateSharesOnRedeem(ctx, num.NewUint(150), "p1", num.NewUint(80)))
}

func TestGetRedemptionRequestForDateLastDate(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logging.NewTestLogger()
	col := mocks.NewMockCollateral(ctrl)
	broker := bmocks.NewMockBroker(ctrl)

	now := time.Unix(1729503411, 0)

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
		RedemptionDates: []*types.RedemptionDate{
			{RedemptionType: types.RedemptionTypeFreeCashOnly, RedemptionDate: now, MaxFraction: num.DecimalFromFloat(0.1)},
		},
	}, col, time.Now(), broker)
	ctx := context.Background()
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.UintZero(), nil).Times(1)
	col.EXPECT().DepositToVault(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&types.LedgerMovement{}, nil).AnyTimes()
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

	// there is only 1 redemption date, meaning it is the last, therefore we expect both parties to be included in the redemption list
	redemptionRequests := vault.GetRedemptionRequestForDate(now)
	require.Equal(t, 2, len(redemptionRequests))
}

func TestGetRedemptionRequestForADate(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logging.NewTestLogger()
	col := mocks.NewMockCollateral(ctrl)
	broker := bmocks.NewMockBroker(ctrl)

	now := time.Unix(1729503411, 0)

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
		CutOffPeriodLength:   3,
		RedemptionDates: []*types.RedemptionDate{
			{RedemptionType: types.RedemptionTypeFreeCashOnly, RedemptionDate: now.Add(3 * 24 * time.Hour), MaxFraction: num.DecimalFromFloat(0.1)},
			{RedemptionType: types.RedemptionTypeFreeCashOnly, RedemptionDate: now.Add(5 * 24 * time.Hour), MaxFraction: num.DecimalFromFloat(0.1)},
		},
	}, col, time.Now(), broker)
	ctx := context.Background()
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.UintZero(), nil).Times(1)
	col.EXPECT().DepositToVault(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&types.LedgerMovement{}, nil).AnyTimes()
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

	// p1 is making a request 3 days before the next redemption date
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(1)
	vault.WithdrawFromVault(ctx, "p1", num.NewUint(25), now)
	// p2 is making a request 2 days before the next redemption date
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(1)
	vault.WithdrawFromVault(ctx, "p2", num.NewUint(25), now.Add(24*time.Hour))

	// with a cutoff of 3 days we expect only p1's request to be included in the first redemption date
	redemptionRequests := vault.GetRedemptionRequestForDate(now.Add(3 * 24 * time.Hour))
	require.Equal(t, 1, len(redemptionRequests))
}

func TestPrepareRedemptions(t *testing.T) {
	// empty vault
	requests := []*vault.RedeemRequest{
		{Party: "p1", Date: time.Time{}, Amount: num.NewUint(100), Remaining: num.NewUint(100), Status: types.RedeemStatusPending},
	}
	partyToRedeemed, lateRedemptions := vault.PrepareRedemptions(map[string]num.Decimal{}, requests, num.NewUint(0), num.NewUint(0), types.RedemptionTypeFreeCashOnly, num.DecimalOne())

	require.Equal(t, 0, len(partyToRedeemed))
	require.Equal(t, 0, len(lateRedemptions))
	for _, rr := range requests {
		require.True(t, rr.Status == types.RedeemStatusCompleted)
	}

	partyToRedeemed, lateRedemptions = vault.PrepareRedemptions(map[string]num.Decimal{}, requests, num.NewUint(100), num.NewUint(200), types.RedemptionTypeFreeCashOnly, num.DecimalOne())

	require.Equal(t, 0, len(partyToRedeemed))
	require.Equal(t, 0, len(lateRedemptions))
	for _, rr := range requests {
		require.True(t, rr.Status == types.RedeemStatusCompleted)
	}

	// vault balance is 160 in cash and 200 in total
	// party 1 has 0.75 share
	// party 2 has 0.25 share
	// party 1 requests to withdraw less than their share = 50
	// party 2 requests to withdraw less than their share = 30
	// type of redemption is cash only
	// party 1 should be able to withdraw 50
	// party 2 should be allowed to withdraw 30 only
	// the redemption is completed as it is cash only
	requests = []*vault.RedeemRequest{
		{Party: "p1", Date: time.Time{}, Amount: num.NewUint(50), Remaining: num.NewUint(50), Status: types.RedeemStatusPending},
		{Party: "p2", Date: time.Time{}, Amount: num.NewUint(30), Remaining: num.NewUint(30), Status: types.RedeemStatusPending},
	}
	partyToRedeemed, lateRedemptions = vault.PrepareRedemptions(map[string]num.Decimal{"p1": num.DecimalFromFloat(0.75), "p2": num.DecimalFromFloat(0.25)}, requests, num.NewUint(200), num.NewUint(160), types.RedemptionTypeFreeCashOnly, num.DecimalOne())

	require.Equal(t, 2, len(partyToRedeemed))
	require.Equal(t, "50", partyToRedeemed["p1"].String())
	require.Equal(t, "30", partyToRedeemed["p2"].String())
	require.Equal(t, 0, len(lateRedemptions))
	for _, rr := range requests {
		require.True(t, rr.Status == types.RedeemStatusCompleted)
		require.True(t, rr.Remaining.IsZero())
	}

	// vault balance is 100 in cash and 200 in total
	// party 1 has 0.75 share
	// party 2 has 0.25 share
	// party 1 requests to withdraw more than their share = 80
	// party 2 requests to withdraw more than their share = 30
	// type of redemption is cash only
	// party 1 should be able to withdraw 50 only
	// party 2 should be allowed to withdraw 25 only
	// the redemption is completed as it is cash only
	requests = []*vault.RedeemRequest{
		{Party: "p1", Date: time.Time{}, Amount: num.NewUint(80), Remaining: num.NewUint(80), Status: types.RedeemStatusPending},
		{Party: "p2", Date: time.Time{}, Amount: num.NewUint(30), Remaining: num.NewUint(30), Status: types.RedeemStatusPending},
	}
	partyToRedeemed, lateRedemptions = vault.PrepareRedemptions(map[string]num.Decimal{"p1": num.DecimalFromFloat(0.75), "p2": num.DecimalFromFloat(0.25)}, requests, num.NewUint(200), num.NewUint(100), types.RedemptionTypeFreeCashOnly, num.DecimalOne())
	require.Equal(t, 2, len(partyToRedeemed))
	require.Equal(t, "75", partyToRedeemed["p1"].String())
	require.Equal(t, "25", partyToRedeemed["p2"].String())
	require.Equal(t, 0, len(lateRedemptions))
	for _, rr := range requests {
		require.True(t, rr.Status == types.RedeemStatusCompleted)
		require.True(t, rr.Remaining.IsZero())
	}

	// party1 has multiple requests such that total is less than the maximum they can withdraw
	// vault balance is 100 in cash and 200 in total
	// party 1 has 0.75 share
	// party 1 requests to withdraw more than their share = 40
	// party 1 requests again to withdraw more than their share = 30
	// party 1 requests again to withdraw more than their share = 10 but in total more than their share in cash
	// type of redemption is cash only
	// party 1 should be able to withdraw 75 only
	requests = []*vault.RedeemRequest{
		{Party: "p1", Date: time.Time{}, Amount: num.NewUint(40), Remaining: num.NewUint(40), Status: types.RedeemStatusPending},
		{Party: "p1", Date: time.Time{}, Amount: num.NewUint(30), Remaining: num.NewUint(30), Status: types.RedeemStatusPending},
		{Party: "p1", Date: time.Time{}, Amount: num.NewUint(10), Remaining: num.NewUint(10), Status: types.RedeemStatusPending},
	}
	partyToRedeemed, lateRedemptions = vault.PrepareRedemptions(map[string]num.Decimal{"p1": num.DecimalFromFloat(0.75), "p2": num.DecimalFromFloat(0.25)}, requests, num.NewUint(200), num.NewUint(100), types.RedemptionTypeFreeCashOnly, num.DecimalOne())
	require.Equal(t, 1, len(partyToRedeemed))
	require.Equal(t, "75", partyToRedeemed["p1"].String())
	require.Equal(t, 0, len(lateRedemptions))
	for _, rr := range requests {
		require.True(t, rr.Status == types.RedeemStatusCompleted)
		require.True(t, rr.Remaining.IsZero())
	}

	// party1
	// vault balance is 100 in cash and 200 in total
	// party 1 has 0.75 share
	// party 1 requests to withdraw more than their share = 40
	// party 1 requests again to withdraw more than their share = 30
	// party 1 requests again to withdraw more than their share = 10 but in total more than their share in cash
	// type of redemption is cash only
	// party 1 should be able to withdraw 75 only
	requests = []*vault.RedeemRequest{
		{Party: "p1", Date: time.Time{}, Amount: num.NewUint(40), Remaining: num.NewUint(40), Status: types.RedeemStatusPending},
		{Party: "p1", Date: time.Time{}, Amount: num.NewUint(30), Remaining: num.NewUint(30), Status: types.RedeemStatusPending},
		{Party: "p1", Date: time.Time{}, Amount: num.NewUint(10), Remaining: num.NewUint(10), Status: types.RedeemStatusPending},
	}
	partyToRedeemed, lateRedemptions = vault.PrepareRedemptions(map[string]num.Decimal{"p1": num.DecimalFromFloat(0.75), "p2": num.DecimalFromFloat(0.25)}, requests, num.NewUint(200), num.NewUint(100), types.RedemptionTypeFreeCashOnly, num.DecimalOne())
	require.Equal(t, 1, len(partyToRedeemed))
	require.Equal(t, "75", partyToRedeemed["p1"].String())
	require.Equal(t, 0, len(lateRedemptions))
	for _, rr := range requests {
		require.True(t, rr.Status == types.RedeemStatusCompleted)
		require.True(t, rr.Remaining.IsZero())
	}

	// party1 and party2 both want to withdraw more than is available in cash but less than their share in a normal redemption day
	requests = []*vault.RedeemRequest{
		{Party: "p1", Date: time.Time{}, Amount: num.NewUint(100), Remaining: num.NewUint(100), Status: types.RedeemStatusPending},
		{Party: "p2", Date: time.Time{}, Amount: num.NewUint(50), Remaining: num.NewUint(50), Status: types.RedeemStatusPending},
	}
	partyToRedeemed, lateRedemptions = vault.PrepareRedemptions(map[string]num.Decimal{"p1": num.DecimalFromFloat(0.75), "p2": num.DecimalFromFloat(0.25)}, requests, num.NewUint(200), num.NewUint(100), types.RedemptionTypeNormal, num.DecimalOne())
	require.Equal(t, 2, len(partyToRedeemed))
	require.Equal(t, "75", partyToRedeemed["p1"].String())
	require.Equal(t, "25", partyToRedeemed["p2"].String())
	require.Equal(t, 2, len(lateRedemptions))
	require.Equal(t, "25", lateRedemptions[0].Remaining.String())
	require.Equal(t, "p1", lateRedemptions[0].Party)
	require.Equal(t, types.RedeemStatusLate, lateRedemptions[0].Status)
	require.Equal(t, "25", lateRedemptions[1].Remaining.String())
	require.Equal(t, "p2", lateRedemptions[1].Party)
	require.Equal(t, types.RedeemStatusLate, lateRedemptions[1].Status)

	// party1 and party2 both want to withdraw more than is available in cash and more than their share in a normal redemption day
	// vault has 200 in total and 80 in cash
	// party1 requests to withdraw 200 - their share of the total amount is 150 - so their late redeem request has 90 remaining
	// party1 requests to withdraw 100 - their share of the total amount is 50 - so their late redeem request has 30 remaining
	requests = []*vault.RedeemRequest{
		{Party: "p1", Date: time.Time{}, Amount: num.NewUint(200), Remaining: num.NewUint(200), Status: types.RedeemStatusPending},
		{Party: "p2", Date: time.Time{}, Amount: num.NewUint(100), Remaining: num.NewUint(100), Status: types.RedeemStatusPending},
	}
	partyToRedeemed, lateRedemptions = vault.PrepareRedemptions(map[string]num.Decimal{"p1": num.DecimalFromFloat(0.75), "p2": num.DecimalFromFloat(0.25)}, requests, num.NewUint(200), num.NewUint(80), types.RedemptionTypeNormal, num.DecimalOne())
	require.Equal(t, 2, len(partyToRedeemed))
	require.Equal(t, "60", partyToRedeemed["p1"].String())
	require.Equal(t, "20", partyToRedeemed["p2"].String())
	require.Equal(t, 2, len(lateRedemptions))
	require.Equal(t, "90", lateRedemptions[0].Remaining.String())
	require.Equal(t, "p1", lateRedemptions[0].Party)
	require.Equal(t, types.RedeemStatusLate, lateRedemptions[0].Status)
	require.Equal(t, "30", lateRedemptions[1].Remaining.String())
	require.Equal(t, "p2", lateRedemptions[1].Party)
	require.Equal(t, types.RedeemStatusLate, lateRedemptions[1].Status)

	// this is the last redemption date so all is up for redemption
	requests = []*vault.RedeemRequest{
		{Party: "p1", Date: time.Time{}, Amount: num.UintZero(), Remaining: num.UintZero(), Status: types.RedeemStatusPending},
		{Party: "p2", Date: time.Time{}, Amount: num.UintZero(), Remaining: num.UintZero(), Status: types.RedeemStatusPending},
	}
	partyToRedeemed, lateRedemptions = vault.PrepareRedemptions(map[string]num.Decimal{"p1": num.DecimalFromFloat(0.75), "p2": num.DecimalFromFloat(0.25)}, requests, num.NewUint(200), num.NewUint(100), types.RedemptionTypeNormal, num.DecimalOne())
	require.Equal(t, 2, len(partyToRedeemed))
	require.Equal(t, "75", partyToRedeemed["p1"].String())
	require.Equal(t, "25", partyToRedeemed["p2"].String())
	require.Equal(t, 2, len(lateRedemptions))
	require.Equal(t, "0", lateRedemptions[0].Remaining.String())
	require.Equal(t, "p1", lateRedemptions[0].Party)
	require.Equal(t, types.RedeemStatusLate, lateRedemptions[0].Status)
	require.Equal(t, "0", lateRedemptions[1].Remaining.String())
	require.Equal(t, "p2", lateRedemptions[1].Party)
	require.Equal(t, types.RedeemStatusLate, lateRedemptions[1].Status)

	// this is the last redemption date so all is up for redemption
	// party 1 has multiple redeem requests
	requests = []*vault.RedeemRequest{
		{Party: "p1", Date: time.Time{}, Amount: num.UintZero(), Remaining: num.UintZero(), Status: types.RedeemStatusPending},
		{Party: "p1", Date: time.Time{}, Amount: num.UintZero(), Remaining: num.UintZero(), Status: types.RedeemStatusPending},
		{Party: "p1", Date: time.Time{}, Amount: num.UintZero(), Remaining: num.UintZero(), Status: types.RedeemStatusPending},
		{Party: "p2", Date: time.Time{}, Amount: num.UintZero(), Remaining: num.UintZero(), Status: types.RedeemStatusPending},
	}
	partyToRedeemed, lateRedemptions = vault.PrepareRedemptions(map[string]num.Decimal{"p1": num.DecimalFromFloat(0.75), "p2": num.DecimalFromFloat(0.25)}, requests, num.NewUint(200), num.NewUint(100), types.RedemptionTypeNormal, num.DecimalOne())
	require.Equal(t, 2, len(partyToRedeemed))
	require.Equal(t, "75", partyToRedeemed["p1"].String())
	require.Equal(t, "25", partyToRedeemed["p2"].String())
	require.Equal(t, 2, len(lateRedemptions))
	require.Equal(t, "0", lateRedemptions[0].Remaining.String())
	require.Equal(t, "p1", lateRedemptions[0].Party)
	require.Equal(t, types.RedeemStatusLate, lateRedemptions[0].Status)
	require.Equal(t, "0", lateRedemptions[1].Remaining.String())
	require.Equal(t, "p2", lateRedemptions[1].Party)
	require.Equal(t, types.RedeemStatusLate, lateRedemptions[1].Status)
}

func TestProcessWithdrawals(t *testing.T) {
	vault := setupVault(t)
	ctx := context.Background()

	// party1 has 75% of the vault (balance is 200)
	// time now is 3 days before the first withdraw date so we don't expect anything to happen when we process withdrawals
	vault.col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(2)
	vault.WithdrawFromVault(ctx, "p1", num.NewUint(40), vault.now)
	vault.WithdrawFromVault(ctx, "p1", num.NewUint(30), vault.now.Add(24*time.Hour))

	// we're ahead of the first redemption date so nothing should happen
	vault.ProcessWithdrawals(ctx, vault.now)
	require.Equal(t, types.VaultStatusActive, vault.GetVaultStatus())

	// only the first withdraw is in scope because the cutoff is 3 days
	// max fraction is 0.1 so the cash amount available for withdrawal on this date is actually 20
	// therefore we expect p1 to be able to withdraw 15
	vault.col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(2)
	vault.col.EXPECT().GetVaultLiquidBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(1)
	vault.col.EXPECT().WithdrawFromVault(ctx, "1", "ETH", "p1", num.NewUint(15)).Return(&types.LedgerMovement{}, nil).Times(1)
	vault.broker.EXPECT().Send(gomock.Any()).Times(1)
	vault.ProcessWithdrawals(ctx, vault.now.Add(3*24*time.Hour))
	require.Equal(t, "185", vault.GetInvestmentTotal().String())

	// moving on to the next withdraw which is a normal one
	// we have one withdrawal for p1 which is for 20 which we'll make greater than the available cash balance
	vault.col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(2)
	vault.col.EXPECT().GetVaultLiquidBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(1)
	// the liquid balance is 200 so we can only redeem a total of 20 out of which 14 (0.7297297297*185) can go towards p1
	// the rest will be postponed to late redemption
	vault.col.EXPECT().WithdrawFromVault(ctx, "1", "ETH", "p1", num.NewUint(14)).Return(&types.LedgerMovement{}, nil).Times(1)
	vault.broker.EXPECT().Send(gomock.Any()).Times(1)
	vault.ProcessWithdrawals(ctx, vault.now.Add(4*24*time.Hour))
	require.Equal(t, "172", vault.GetInvestmentTotal().String())

	// at this point we have 1 late redeem request for 16
	vault.col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(2)
	vault.col.EXPECT().GetVaultLiquidBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(150), nil).Times(1)
	vault.col.EXPECT().WithdrawFromVault(ctx, "1", "ETH", "p1", num.NewUint(16)).Return(&types.LedgerMovement{}, nil).Times(1)
	vault.broker.EXPECT().Send(gomock.Any()).Times(1)
	vault.ProcessLateRedemptions(ctx)
	require.Equal(t, "158", vault.GetInvestmentTotal().String())

	// now lets get to the final redemption. The redemption is setup as cash only but it doesn't matter as
	// it would be treated as all with no factor
	vault.col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(1)
	vault.col.EXPECT().GetVaultLiquidBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(1)
	vault.col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(200), nil).Times(1)
	vault.col.EXPECT().GetVaultLiquidBalance(gomock.Any(), gomock.Any()).Return(num.NewUint(1), nil).Times(1)
	vault.col.EXPECT().WithdrawFromVault(ctx, "1", "ETH", "p1", num.NewUint(136)).Return(&types.LedgerMovement{}, nil).Times(1)
	vault.col.EXPECT().WithdrawFromVault(ctx, "1", "ETH", "p2", num.NewUint(63)).Return(&types.LedgerMovement{}, nil).Times(1)
	vault.col.EXPECT().WithdrawFromVault(ctx, "1", "ETH", "zohar", num.NewUint(1)).Return(&types.LedgerMovement{}, nil).Times(1)
	vault.col.EXPECT().CloseVaultAccount(ctx, "1")
	vault.broker.EXPECT().Send(gomock.Any()).Times(3)

	vault.ProcessWithdrawals(ctx, vault.now.Add(5*24*time.Hour))
	require.Equal(t, "0", vault.GetInvestmentTotal().String())
}

type testVault struct {
	*vault.VaultState
	col    *mocks.MockCollateral
	broker *bmocks.MockBroker
	now    time.Time
}

func setupVault(t *testing.T) *testVault {
	t.Helper()
	ctrl := gomock.NewController(t)
	logger := logging.NewTestLogger()
	col := mocks.NewMockCollateral(ctrl)
	broker := bmocks.NewMockBroker(ctrl)

	now := time.Unix(1729503411, 0)

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
		CutOffPeriodLength:   3,
		RedemptionDates: []*types.RedemptionDate{
			{RedemptionType: types.RedemptionTypeFreeCashOnly, RedemptionDate: now.Add(3 * 24 * time.Hour), MaxFraction: num.DecimalFromFloat(0.1)},
			{RedemptionType: types.RedemptionTypeNormal, RedemptionDate: now.Add(4 * 24 * time.Hour), MaxFraction: num.DecimalFromFloat(0.1)},
			{RedemptionType: types.RedemptionTypeFreeCashOnly, RedemptionDate: now.Add(5 * 24 * time.Hour), MaxFraction: num.DecimalFromFloat(0.1)},
		},
	}, col, time.Now(), broker)
	ctx := context.Background()
	col.EXPECT().GetVaultBalance(gomock.Any(), gomock.Any()).Return(num.UintZero(), nil).Times(1)
	col.EXPECT().DepositToVault(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&types.LedgerMovement{}, nil).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).Times(2)

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

	return &testVault{
		vault,
		col,
		broker,
		now,
	}
}
