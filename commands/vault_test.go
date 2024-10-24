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

package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckCreateVault(t *testing.T) {
	cases := []struct {
		submission commandspb.CreateVault
		errStr     string
	}{
		{
			submission: commandspb.CreateVault{},
			errStr:     "create_vault.asset (is required)",
		},
		{
			submission: commandspb.CreateVault{
				Asset: "notavalidassetid",
			},
			errStr: "create_vault.asset (should be a valid Vega ID)",
		},
		{
			submission: commandspb.CreateVault{
				VaultMetadata: &vega.VaultMetaData{},
			},
			errStr: "create_vault.metadata.name (is required)",
		},
		{
			submission: commandspb.CreateVault{
				ManagementFeeFactor: "abc",
			},
			errStr: "create_vault.management_fee_factor (is not a valid number)",
		},
		{
			submission: commandspb.CreateVault{
				ManagementFeeFactor: "",
			},
			errStr: "create_vault.management_fee_factor (is required)",
		},
		{
			submission: commandspb.CreateVault{
				ManagementFeeFactor: "-1",
			},
			errStr: "create_vault.management_fee_factor (must be positive or zero)",
		},
		{
			submission: commandspb.CreateVault{
				PerformanceFeeFactor: "abc",
			},
			errStr: "create_vault.performance_fee_factor (is not a valid number)",
		},
		{
			submission: commandspb.CreateVault{
				PerformanceFeeFactor: "",
			},
			errStr: "create_vault.performance_fee_factor (is required)",
		},
		{
			submission: commandspb.CreateVault{
				PerformanceFeeFactor: "-1",
			},
			errStr: "create_vault.performance_fee_factor (must be positive or zero)",
		},
		{
			submission: commandspb.CreateVault{
				FeePeriod: "",
			},
			errStr: "create_vault.fee_period (is required)",
		},
		{
			submission: commandspb.CreateVault{
				FeePeriod: "sdjkhfjk",
			},
			errStr: "create_vault.fee_period (is not a valid value)",
		},
		{
			submission: commandspb.CreateVault{
				CutOffPeriodLength: -5,
			},
			errStr: "create_vault.cut_off_period_length (must be positive)",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{},
			},
			errStr: "create_vault.redemption_dates (is required)",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionDate: -1234,
					},
				},
			},
			errStr: "create_vault.redemption_dates.0.redemption_date (is not a valid value)",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionDate: 1234,
					},
					{
						RedemptionDate: -1234,
					},
				},
			},
			errStr: "create_vault.redemption_dates.1.redemption_date (is not a valid value)",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionType: 0,
					},
				},
			},
			errStr: "create_vault.redemption_dates.0.redemption_type (is not a valid value)",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionType: 1,
					},
					{
						RedemptionType: 2,
					},
					{
						RedemptionType: 0,
					},
				},
			},
			errStr: "create_vault.redemption_dates.2.redemption_type (is not a valid value)",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionType: 4,
					},
				},
			},
			errStr: "create_vault.redemption_dates.0.redemption_type (is not a valid value)",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionType: 1,
					},
					{
						RedemptionType: 2,
					},
					{
						RedemptionType: 4,
					},
				},
			},
			errStr: "create_vault.redemption_dates.2.redemption_type (is not a valid value)",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "",
					},
				},
			},
			errStr: "create_vault.redemption_dates.0.max_fraction (is required)",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "bbbb",
					},
				},
			},
			errStr: "create_vault.redemption_dates.0.max_fraction (is not a valid value)",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "-0.5",
					},
				},
			},
			errStr: "create_vault.redemption_dates.0.max_fraction (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "0",
					},
				},
			},
			errStr: "create_vault.redemption_dates.0.max_fraction (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "1.1",
					},
				},
			},
			errStr: "create_vault.redemption_dates.0.max_fraction (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "1",
					},
					{
						MaxFraction: "0.1",
					},
					{
						MaxFraction: "",
					},
				},
			},
			errStr: "create_vault.redemption_dates.2.max_fraction (is required)",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "1",
					},
					{
						MaxFraction: "0.1",
					},
					{
						MaxFraction: "bbbb",
					},
				},
			},
			errStr: "create_vault.redemption_dates.2.max_fraction (is not a valid value)",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "1",
					},
					{
						MaxFraction: "0.1",
					},
					{
						MaxFraction: "-0.5",
					},
				},
			},
			errStr: "create_vault.redemption_dates.2.max_fraction (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "1",
					},
					{
						MaxFraction: "0.1",
					},
					{
						MaxFraction: "0",
					},
				},
			},
			errStr: "create_vault.redemption_dates.2.max_fraction (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.CreateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "1",
					},
					{
						MaxFraction: "0.1",
					},
					{
						MaxFraction: "1.1",
					},
				},
			},
			errStr: "create_vault.redemption_dates.2.max_fraction (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.CreateVault{
				Asset:                "e9982447fb4128f9968f9981612c5ea85d19b62058ec2636efc812dcbbc745ca",
				ManagementFeeFactor:  "0.09",
				PerformanceFeeFactor: "0.03",
				FeePeriod:            "10h",
				VaultMetadata: &vega.VaultMetaData{
					Name:        "zohar",
					Description: "really good fund",
					Url:         "some url",
					ImageUrl:    "some image url",
				},
				CutOffPeriodLength: 1,
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionDate: 1234,
						RedemptionType: vega.RedemptionType_REDEMPTION_TYPE_FREE_CASH_ONLY,
						MaxFraction:    "0.1",
					},
				},
			},
		},
	}

	for n, c := range cases {
		if len(c.errStr) <= 0 {
			assert.NoError(t, commands.CheckCreateVault(&c.submission), n)
			continue
		}

		assert.Contains(t, checkCreateVault(&c.submission).Error(), c.errStr, n)
	}
}

func checkCreateVault(cmd *commandspb.CreateVault) commands.Errors {
	err := commands.CheckCreateVault(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}

func TestCheckUpdateVault(t *testing.T) {
	cases := []struct {
		submission commandspb.UpdateVault
		errStr     string
	}{
		{
			submission: commandspb.UpdateVault{
				VaultId: "",
			},
			errStr: "update_vault.vault_id (is required)",
		},
		{
			submission: commandspb.UpdateVault{
				VaultId: "dskjhfkjhjk",
			},
			errStr: "update_vault.vault_id (is not a valid vault identifier)",
		},
		{
			submission: commandspb.UpdateVault{
				VaultMetadata: &vega.VaultMetaData{},
			},
			errStr: "update_vault.metadata.name (is required)",
		},
		{
			submission: commandspb.UpdateVault{
				ManagementFeeFactor: "abc",
			},
			errStr: "update_vault.management_fee_factor (is not a valid number)",
		},
		{
			submission: commandspb.UpdateVault{
				ManagementFeeFactor: "",
			},
			errStr: "update_vault.management_fee_factor (is required)",
		},
		{
			submission: commandspb.UpdateVault{
				ManagementFeeFactor: "-1",
			},
			errStr: "update_vault.management_fee_factor (must be positive or zero)",
		},
		{
			submission: commandspb.UpdateVault{
				PerformanceFeeFactor: "abc",
			},
			errStr: "update_vault.performance_fee_factor (is not a valid number)",
		},
		{
			submission: commandspb.UpdateVault{
				PerformanceFeeFactor: "",
			},
			errStr: "update_vault.performance_fee_factor (is required)",
		},
		{
			submission: commandspb.UpdateVault{
				PerformanceFeeFactor: "-1",
			},
			errStr: "update_vault.performance_fee_factor (must be positive or zero)",
		},
		{
			submission: commandspb.UpdateVault{
				FeePeriod: "",
			},
			errStr: "update_vault.fee_period (is required)",
		},
		{
			submission: commandspb.UpdateVault{
				FeePeriod: "sdjkhfjk",
			},
			errStr: "update_vault.fee_period (is not a valid value)",
		},
		{
			submission: commandspb.UpdateVault{
				CutOffPeriodLength: -5,
			},
			errStr: "update_vault.cut_off_period_length (must be positive)",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{},
			},
			errStr: "update_vault.redemption_dates (is required)",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionDate: -1234,
					},
				},
			},
			errStr: "update_vault.redemption_dates.0.redemption_date (is not a valid value)",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionDate: 1234,
					},
					{
						RedemptionDate: -1234,
					},
				},
			},
			errStr: "update_vault.redemption_dates.1.redemption_date (is not a valid value)",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionType: 0,
					},
				},
			},
			errStr: "update_vault.redemption_dates.0.redemption_type (is not a valid value)",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionType: 2,
					},
					{
						RedemptionType: 2,
					},
					{
						RedemptionType: 0,
					},
				},
			},
			errStr: "update_vault.redemption_dates.2.redemption_type (is not a valid value)",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionType: 4,
					},
				},
			},
			errStr: "update_vault.redemption_dates.0.redemption_type (is not a valid value)",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionType: 1,
					},
					{
						RedemptionType: 1,
					},
					{
						RedemptionType: 4,
					},
				},
			},
			errStr: "update_vault.redemption_dates.2.redemption_type (is not a valid value)",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "",
					},
				},
			},
			errStr: "update_vault.redemption_dates.0.max_fraction (is required)",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "bbbb",
					},
				},
			},
			errStr: "update_vault.redemption_dates.0.max_fraction (is not a valid value)",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "-0.5",
					},
				},
			},
			errStr: "update_vault.redemption_dates.0.max_fraction (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "0",
					},
				},
			},
			errStr: "update_vault.redemption_dates.0.max_fraction (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "1.1",
					},
				},
			},
			errStr: "update_vault.redemption_dates.0.max_fraction (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "1",
					},
					{
						MaxFraction: "0.1",
					},
					{
						MaxFraction: "",
					},
				},
			},
			errStr: "update_vault.redemption_dates.2.max_fraction (is required)",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "1",
					},
					{
						MaxFraction: "0.1",
					},
					{
						MaxFraction: "bbbb",
					},
				},
			},
			errStr: "update_vault.redemption_dates.2.max_fraction (is not a valid value)",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "1",
					},
					{
						MaxFraction: "0.1",
					},
					{
						MaxFraction: "-0.5",
					},
				},
			},
			errStr: "update_vault.redemption_dates.2.max_fraction (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "1",
					},
					{
						MaxFraction: "0.1",
					},
					{
						MaxFraction: "0",
					},
				},
			},
			errStr: "update_vault.redemption_dates.2.max_fraction (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.UpdateVault{
				RedemptionDates: []*vega.RedemptionDate{
					{
						MaxFraction: "1",
					},
					{
						MaxFraction: "0.1",
					},
					{
						MaxFraction: "1.1",
					},
				},
			},
			errStr: "update_vault.redemption_dates.2.max_fraction (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.UpdateVault{
				VaultId:              "e9982447fb4128f9968f9981612c5ea85d19b62058ec2636efc812dcbbc745ca",
				ManagementFeeFactor:  "0.09",
				PerformanceFeeFactor: "0.03",
				FeePeriod:            "10h",
				VaultMetadata: &vega.VaultMetaData{
					Name:        "zohar",
					Description: "really good fund",
					Url:         "some url",
					ImageUrl:    "some image url",
				},
				CutOffPeriodLength: 1,
				RedemptionDates: []*vega.RedemptionDate{
					{
						RedemptionDate: 1234,
						RedemptionType: vega.RedemptionType_REDEMPTION_TYPE_FREE_CASH_ONLY,
						MaxFraction:    "0.1",
					},
				},
			},
		},
	}

	for n, c := range cases {
		if len(c.errStr) <= 0 {
			assert.NoError(t, commands.CheckUpdateVault(&c.submission), n)
			continue
		}

		assert.Contains(t, checkUpdateVault(&c.submission).Error(), c.errStr, n)
	}
}

func checkUpdateVault(cmd *commandspb.UpdateVault) commands.Errors {
	err := commands.CheckUpdateVault(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}

func TestCheckDepositToVault(t *testing.T) {
	cases := []struct {
		submission commandspb.DepositToVault
		errStr     string
	}{
		{
			submission: commandspb.DepositToVault{
				VaultId: "",
			},
			errStr: "deposit_to_vault.vault_id (is required)",
		},
		{
			submission: commandspb.DepositToVault{
				VaultId: "dskjhfkjhjk",
			},
			errStr: "deposit_to_vault.vault_id (is not a valid vault identifier)",
		},
		{
			submission: commandspb.DepositToVault{
				Amount: "",
			},
			errStr: "deposit_to_vault.amount (is required)",
		},
		{
			submission: commandspb.DepositToVault{
				Amount: "0",
			},
			errStr: "deposit_to_vault.amount (must be positive)",
		},
		{
			submission: commandspb.DepositToVault{
				Amount: "-10",
			},
			errStr: "deposit_to_vault.amount (must be positive)",
		},
		{
			submission: commandspb.DepositToVault{
				VaultId: "e9982447fb4128f9968f9981612c5ea85d19b62058ec2636efc812dcbbc745ca",
				Amount:  "10",
			},
		},
	}

	for n, c := range cases {
		if len(c.errStr) <= 0 {
			assert.NoError(t, commands.CheckDepositToVault(&c.submission), n)
			continue
		}

		assert.Contains(t, checkDepositToVault(&c.submission).Error(), c.errStr, n)
	}
}

func checkDepositToVault(cmd *commandspb.DepositToVault) commands.Errors {
	err := commands.CheckDepositToVault(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}

func TestCheckWithdrawFromVault(t *testing.T) {
	cases := []struct {
		submission commandspb.WithdrawFromVault
		errStr     string
	}{
		{
			submission: commandspb.WithdrawFromVault{
				VaultId: "",
			},
			errStr: "withdraw_from_vault.vault_id (is required)",
		},
		{
			submission: commandspb.WithdrawFromVault{
				VaultId: "dskjhfkjhjk",
			},
			errStr: "withdraw_from_vault.vault_id (is not a valid vault identifier)",
		},
		{
			submission: commandspb.WithdrawFromVault{
				Amount: "",
			},
			errStr: "withdraw_from_vault.amount (is required)",
		},
		{
			submission: commandspb.WithdrawFromVault{
				Amount: "0",
			},
			errStr: "withdraw_from_vault.amount (must be positive)",
		},
		{
			submission: commandspb.WithdrawFromVault{
				Amount: "-10",
			},
			errStr: "withdraw_from_vault.amount (must be positive)",
		},
		{
			submission: commandspb.WithdrawFromVault{
				VaultId: "e9982447fb4128f9968f9981612c5ea85d19b62058ec2636efc812dcbbc745ca",
				Amount:  "10",
			},
		},
	}

	for n, c := range cases {
		if len(c.errStr) <= 0 {
			assert.NoError(t, commands.CheckWithdrawFromVault(&c.submission), n)
			continue
		}

		assert.Contains(t, checkWithdrawFromVault(&c.submission).Error(), c.errStr, n)
	}
}

func checkWithdrawFromVault(cmd *commandspb.WithdrawFromVault) commands.Errors {
	err := commands.CheckWithdrawFromVault(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}

func TestCheckChangeVaultOwnership(t *testing.T) {
	cases := []struct {
		submission commandspb.ChangeVaultOwnership
		errStr     string
	}{
		{
			submission: commandspb.ChangeVaultOwnership{
				VaultId: "",
			},
			errStr: "change_vault_ownership.vault_id (is required)",
		},
		{
			submission: commandspb.ChangeVaultOwnership{
				VaultId: "dskjhfkjhjk",
			},
			errStr: "change_vault_ownership.vault_id (should be a valid Vega ID)",
		},
		{
			submission: commandspb.ChangeVaultOwnership{
				NewOwner: "",
			},
			errStr: "change_vault_ownership.new_owner (is required)",
		},
		{
			submission: commandspb.ChangeVaultOwnership{
				NewOwner: "dskjhfkjhjk",
			},
			errStr: "change_vault_ownership.new_owner (should be a valid Vega ID)",
		},
		{
			submission: commandspb.ChangeVaultOwnership{
				VaultId:  "e9982447fb4128f9968f9981612c5ea85d19b62058ec2636efc812dcbbc745ca",
				NewOwner: "e9982447fb4128f9968f9981612c5ea85d19b62058ec2636efc812dcbbc745ca",
			},
		},
	}

	for n, c := range cases {
		if len(c.errStr) <= 0 {
			assert.NoError(t, commands.CheckChangeVaultOwnership(&c.submission), n)
			continue
		}

		assert.Contains(t, checkChangeVaultOwnership(&c.submission).Error(), c.errStr, n)
	}
}

func checkChangeVaultOwnership(cmd *commandspb.ChangeVaultOwnership) commands.Errors {
	err := commands.CheckChangeVaultOwnership(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
