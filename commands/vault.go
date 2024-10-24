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

package commands

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckCreateVault(cmd *commandspb.CreateVault) error {
	return checkCreateVault(cmd).ErrorOrNil()
}

func CheckUpdateVault(cmd *commandspb.UpdateVault) error {
	return checkUpdateVault(cmd).ErrorOrNil()
}

func CheckDepositToVault(cmd *commandspb.DepositToVault) error {
	return checkDepositToVault(cmd).ErrorOrNil()
}

func CheckWithdrawFromVault(cmd *commandspb.WithdrawFromVault) error {
	return checkWithdrawFromVault(cmd).ErrorOrNil()
}

func CheckChangeVaultOwnership(cmd *commandspb.ChangeVaultOwnership) error {
	return checkChangeVaultOwnership(cmd).ErrorOrNil()
}

func checkDepositToVault(cmd *commandspb.DepositToVault) Errors {
	errs := NewErrors()
	if cmd == nil {
		return errs.FinalAddForProperty("deposit_to_vault", ErrIsRequired)
	}
	if len(cmd.VaultId) == 0 {
		errs.AddForProperty("deposit_to_vault.vault_id", ErrIsRequired)
	} else if !IsVegaID(cmd.VaultId) {
		errs.AddForProperty("deposit_to_vault.vault_id", ErrInvalidVaultID)
	}
	if len(cmd.Amount) == 0 {
		errs.AddForProperty("deposit_to_vault.amount", ErrIsRequired)
	} else {
		amt, overflow := num.UintFromString(cmd.Amount, 10)
		if overflow || amt.IsNegative() || amt.IsZero() {
			errs.AddForProperty("deposit_to_vault.amount", ErrMustBePositive)
		}
	}
	return errs
}

func checkWithdrawFromVault(cmd *commandspb.WithdrawFromVault) Errors {
	errs := NewErrors()
	if cmd == nil {
		return errs.FinalAddForProperty("withdraw_from_vault", ErrIsRequired)
	}

	if len(cmd.VaultId) == 0 {
		errs.AddForProperty("withdraw_from_vault.vault_id", ErrIsRequired)
	} else if !IsVegaID(cmd.VaultId) {
		errs.AddForProperty("withdraw_from_vault.vault_id", ErrInvalidVaultID)
	}
	if len(cmd.Amount) == 0 {
		errs.AddForProperty("withdraw_from_vault.amount", ErrIsRequired)
	} else {
		amt, overflow := num.UintFromString(cmd.Amount, 10)
		if overflow || amt.IsNegative() || amt.IsZero() {
			errs.AddForProperty("withdraw_from_vault.amount", ErrMustBePositive)
		}
	}
	return errs
}

func checkUpdateVault(cmd *commandspb.UpdateVault) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("update_vault", ErrIsRequired)
	}

	if len(cmd.VaultId) == 0 {
		errs.AddForProperty("update_vault.vault_id", ErrIsRequired)
	} else if !IsVegaID(cmd.VaultId) {
		errs.AddForProperty("update_vault.vault_id", ErrInvalidVaultID)
	}

	if cmd.VaultMetadata != nil {
		if len(cmd.VaultMetadata.Name) == 0 {
			errs.AddForProperty("update_vault.metadata.name", ErrIsRequired)
		}
	}

	if len(cmd.FeePeriod) == 0 {
		errs.AddForProperty("update_vault.fee_period", ErrIsRequired)
	} else {
		_, err := time.ParseDuration(cmd.FeePeriod)
		if err != nil {
			errs.AddForProperty("update_vault.fee_period", ErrIsNotValid)
		}
	}

	if len(cmd.ManagementFeeFactor) == 0 {
		errs.AddForProperty("update_vault.management_fee_factor", ErrIsRequired)
	} else {
		managementFeeFactor, err := num.DecimalFromString(cmd.ManagementFeeFactor)
		if err != nil {
			errs.AddForProperty("update_vault.management_fee_factor", ErrIsNotValidNumber)
		} else {
			if managementFeeFactor.LessThan(num.DecimalZero()) {
				errs.AddForProperty("update_vault.management_fee_factor", ErrMustBePositiveOrZero)
			}
		}
	}

	if len(cmd.PerformanceFeeFactor) == 0 {
		errs.AddForProperty("update_vault.performance_fee_factor", ErrIsRequired)
	} else {
		performanceFeeFactor, err := num.DecimalFromString(cmd.PerformanceFeeFactor)
		if err != nil {
			errs.AddForProperty("update_vault.performance_fee_factor", ErrIsNotValidNumber)
		} else {
			if performanceFeeFactor.LessThan(num.DecimalZero()) {
				errs.AddForProperty("update_vault.performance_fee_factor", ErrMustBePositiveOrZero)
			}
		}
	}
	if cmd.CutOffPeriodLength < 0 {
		errs.AddForProperty("update_vault.cut_off_period_length", ErrMustBePositive)
	}
	if len(cmd.RedemptionDates) == 0 {
		errs.AddForProperty("update_vault.redemption_dates", ErrIsRequired)
	} else {
		for i, rd := range cmd.RedemptionDates {
			if rd.RedemptionType != vega.RedemptionType_REDEMPTION_TYPE_NORMAL && rd.RedemptionType != vega.RedemptionType_REDEMPTION_TYPE_FREE_CASH_ONLY {
				errs.AddForProperty(fmt.Sprintf("update_vault.redemption_dates.%d.redemption_type", i), ErrIsNotValid)
			}
			if len(rd.MaxFraction) == 0 {
				errs.AddForProperty(fmt.Sprintf("update_vault.redemption_dates.%d.max_fraction", i), ErrIsRequired)
			} else {
				maxFraction, err := num.DecimalFromString(rd.MaxFraction)
				if err != nil {
					errs.AddForProperty(fmt.Sprintf("update_vault.redemption_dates.%d.max_fraction", i), ErrIsNotValid)
				} else if !maxFraction.IsPositive() || maxFraction.GreaterThan(num.DecimalOne()) {
					errs.AddForProperty(fmt.Sprintf("update_vault.redemption_dates.%d.max_fraction", i), ErrMustBeBetween01)
				}
			}
			if rd.RedemptionDate < 0 {
				errs.AddForProperty(fmt.Sprintf("update_vault.redemption_dates.%d.redemption_date", i), ErrIsNotValid)
			}
			if i > 0 && rd.RedemptionDate <= cmd.RedemptionDates[i-1].RedemptionDate {
				errs.AddForProperty("update_vault.redemption_dates", fmt.Errorf("must be monotonically increasing"))
			}
		}
	}
	return errs
}

func checkCreateVault(cmd *commandspb.CreateVault) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("create_vault", ErrIsRequired)
	}

	if len(cmd.Asset) <= 0 {
		errs.AddForProperty("create_vault.asset", ErrIsRequired)
	} else if !IsVegaID(cmd.Asset) {
		errs.AddForProperty("create_vault.asset", ErrShouldBeAValidVegaID)
	}

	if cmd.VaultMetadata == nil {
		errs.AddForProperty("create_vault.metadata", ErrIsRequired)
	} else {
		if len(cmd.VaultMetadata.Name) == 0 {
			errs.AddForProperty("create_vault.metadata.name", ErrIsRequired)
		}
	}

	if len(cmd.FeePeriod) == 0 {
		errs.AddForProperty("create_vault.fee_period", ErrIsRequired)
	} else {
		_, err := time.ParseDuration(cmd.FeePeriod)
		if err != nil {
			errs.AddForProperty("create_vault.fee_period", ErrIsNotValid)
		}
	}

	if len(cmd.ManagementFeeFactor) == 0 {
		errs.AddForProperty("create_vault.management_fee_factor", ErrIsRequired)
	} else {
		managementFeeFactor, err := num.DecimalFromString(cmd.ManagementFeeFactor)
		if err != nil {
			errs.AddForProperty("create_vault.management_fee_factor", ErrIsNotValidNumber)
		} else {
			if managementFeeFactor.LessThan(num.DecimalZero()) {
				errs.AddForProperty("create_vault.management_fee_factor", ErrMustBePositiveOrZero)
			}
		}
	}

	if len(cmd.PerformanceFeeFactor) == 0 {
		errs.AddForProperty("create_vault.performance_fee_factor", ErrIsRequired)
	} else {
		performanceFeeFactor, err := num.DecimalFromString(cmd.PerformanceFeeFactor)
		if err != nil {
			errs.AddForProperty("create_vault.performance_fee_factor", ErrIsNotValidNumber)
		} else {
			if performanceFeeFactor.LessThan(num.DecimalZero()) {
				errs.AddForProperty("create_vault.performance_fee_factor", ErrMustBePositiveOrZero)
			}
		}
	}
	if cmd.CutOffPeriodLength < 0 {
		errs.AddForProperty("create_vault.cut_off_period_length", ErrMustBePositive)
	}
	if len(cmd.RedemptionDates) == 0 {
		errs.AddForProperty("create_vault.redemption_dates", ErrIsRequired)
	} else {
		for i, rd := range cmd.RedemptionDates {
			if rd.RedemptionType != vega.RedemptionType_REDEMPTION_TYPE_NORMAL && rd.RedemptionType != vega.RedemptionType_REDEMPTION_TYPE_FREE_CASH_ONLY {
				errs.AddForProperty(fmt.Sprintf("create_vault.redemption_dates.%d.redemption_type", i), ErrIsNotValid)
			}
			if len(rd.MaxFraction) == 0 {
				errs.AddForProperty(fmt.Sprintf("create_vault.redemption_dates.%d.max_fraction", i), ErrIsRequired)
			} else {
				maxFraction, err := num.DecimalFromString(rd.MaxFraction)
				if err != nil {
					errs.AddForProperty(fmt.Sprintf("create_vault.redemption_dates.%d.max_fraction", i), ErrIsNotValid)
				} else if !maxFraction.IsPositive() || maxFraction.GreaterThan(num.DecimalOne()) {
					errs.AddForProperty(fmt.Sprintf("create_vault.redemption_dates.%d.max_fraction", i), ErrMustBeBetween01)
				}
			}
			if rd.RedemptionDate < 0 {
				errs.AddForProperty(fmt.Sprintf("create_vault.redemption_dates.%d.redemption_date", i), ErrIsNotValid)
			}
			if i > 0 && rd.RedemptionDate <= cmd.RedemptionDates[i-1].RedemptionDate {
				errs.AddForProperty("create_vault.redemption_dates", fmt.Errorf("must be monotonically increasing"))
			}
		}
	}
	return errs
}

func checkChangeVaultOwnership(cmd *commandspb.ChangeVaultOwnership) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("change_vault_ownership", ErrIsRequired)
	}

	if len(cmd.VaultId) <= 0 {
		errs.AddForProperty("change_vault_ownership.vault_id", ErrIsRequired)
	} else if !IsVegaID(cmd.VaultId) {
		errs.AddForProperty("change_vault_ownership.vault_id", ErrShouldBeAValidVegaID)
	}

	if len(cmd.NewOwner) <= 0 {
		errs.AddForProperty("change_vault_ownership.new_owner", ErrIsRequired)
	} else if !IsVegaID(cmd.NewOwner) {
		errs.AddForProperty("change_vault_ownership.new_owner", ErrShouldBeAValidVegaID)
	}

	return errs
}
