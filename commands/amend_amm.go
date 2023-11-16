// Copyright (C) 2023  Gobalsky Labs Limited
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
	"math/big"

	"code.vegaprotocol.io/vega/libs/num"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckAmendAMM(cmd *commandspb.AmendAMM) error {
	return checkAmendAMM(cmd).ErrorOrNil()
}

func checkAmendAMM(cmd *commandspb.AmendAMM) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("amend_amm", ErrIsRequired)
	}

	if len(cmd.MarketId) <= 0 {
		errs.AddForProperty("amend_amm.market_id", ErrIsRequired)
	} else if !IsVegaID(cmd.MarketId) {
		errs.AddForProperty("amend_amm.market_id", ErrShouldBeAValidVegaID)
	}

	if len(cmd.SlippageTolerance) <= 0 {
		errs.AddForProperty("amend_amm.slippage_tolerance", ErrIsRequired)
	} else if slippageTolerance, err := num.DecimalFromString(cmd.SlippageTolerance); err != nil {
		errs.AddForProperty("amend_amm.slippage_tolerance", ErrIsNotValidNumber)
	} else if slippageTolerance.LessThanOrEqual(num.DecimalZero()) || slippageTolerance.GreaterThan(num.DecimalOne()) {
		errs.AddForProperty("amend_amm.slippage_tolerance", ErrMustBeBetween01)
	}

	var hasUpdate bool

	if cmd.CommitmentAmount != nil {
		hasUpdate = true
		if amount, _ := big.NewInt(0).SetString(*cmd.CommitmentAmount, 10); amount == nil {
			errs.FinalAddForProperty("amend_amm.commitment_amount", ErrIsNotValidNumber)
		} else if amount.Cmp(big.NewInt(0)) <= 0 {
			errs.AddForProperty("amend_amm.commitment_amount", ErrMustBePositive)
		}
	}

	if cmd.ConcentratedLiquidityParameters != nil {
		if cmd.ConcentratedLiquidityParameters.Base != nil {
			hasUpdate = true
			if amount, _ := big.NewInt(0).SetString(*cmd.ConcentratedLiquidityParameters.Base, 10); amount == nil {
				errs.FinalAddForProperty("amend_amm.concentrated_liquidity_parameters.base", ErrIsNotValidNumber)
			} else if amount.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.base", ErrMustBePositive)
			}
		}
		if cmd.ConcentratedLiquidityParameters.LowerBound != nil {
			hasUpdate = true
			if amount, _ := big.NewInt(0).SetString(*cmd.ConcentratedLiquidityParameters.LowerBound, 10); amount == nil {
				errs.FinalAddForProperty("amend_amm.concentrated_liquidity_parameters.lower_bound", ErrIsNotValidNumber)
			} else if amount.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.lower_bound", ErrMustBePositive)
			}
		}
		if cmd.ConcentratedLiquidityParameters.UpperBound != nil {
			hasUpdate = true
			if amount, _ := big.NewInt(0).SetString(*cmd.ConcentratedLiquidityParameters.UpperBound, 10); amount == nil {
				errs.FinalAddForProperty("amend_amm.concentrated_liquidity_parameters.upper_bound", ErrIsNotValidNumber)
			} else if amount.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.upper_bound", ErrMustBePositive)
			}
		}

		if cmd.ConcentratedLiquidityParameters.MarginRatioAtUpperBound != nil {
			hasUpdate = true
			if marginRatio, err := num.DecimalFromString(*cmd.ConcentratedLiquidityParameters.MarginRatioAtUpperBound); err != nil {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.margin_ratio_at_upper_bound", ErrIsNotValidNumber)
			} else if marginRatio.LessThan(num.DecimalZero()) {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.margin_ratio_at_upper_bound", ErrMustBePositive)
			}
		}

		if cmd.ConcentratedLiquidityParameters.MarginRatioAtLowerBound != nil {
			hasUpdate = true
			if marginRatio, err := num.DecimalFromString(*cmd.ConcentratedLiquidityParameters.MarginRatioAtLowerBound); err != nil {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.margin_ratio_at_lower_bound", ErrIsNotValidNumber)
			} else if marginRatio.LessThan(num.DecimalZero()) {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.margin_ratio_at_lower_bound", ErrMustBePositive)
			}
		}

	}

	// no update, but also no error, invalid
	if !hasUpdate && errs.Empty() {
		errs.FinalAdd(ErrNoUpdatesProvided)
	}

	return errs
}
