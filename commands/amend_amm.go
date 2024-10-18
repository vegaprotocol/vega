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
	"errors"
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

	if cmd.ProposedFee != nil {
		hasUpdate = true
		if proposedFee, err := num.DecimalFromString(*cmd.ProposedFee); err != nil {
			errs.AddForProperty("amend_amm.proposed_fee", ErrIsNotValid)
		} else if proposedFee.LessThanOrEqual(num.DecimalZero()) {
			errs.AddForProperty("amend_amm.proposed_fee", ErrMustBePositive)
		}
	}

	if cmd.MinimumPriceChangeTrigger != nil {
		if minPriceChange, err := num.DecimalFromString(*cmd.MinimumPriceChangeTrigger); err != nil {
			errs.AddForProperty("submit_amm.mimimum_price_change_trigger", ErrIsNotValid)
		} else if minPriceChange.LessThan(num.DecimalZero()) {
			errs.AddForProperty("submit_amm.proposed_fee", ErrMustBePositiveOrZero)
		}
	}

	if cmd.ConcentratedLiquidityParameters != nil {
		var haveLower, haveUpper, emptyBase bool
		hasUpdate = true
		var base, lowerBound, upperBound *big.Int
		if len(cmd.ConcentratedLiquidityParameters.Base) == 0 {
			emptyBase = true
		} else if base, _ = big.NewInt(0).SetString(cmd.ConcentratedLiquidityParameters.Base, 10); base == nil {
			errs.FinalAddForProperty("amend_amm.concentrated_liquidity_parameters.base", ErrIsNotValidNumber)
		} else if base.Cmp(big.NewInt(0)) <= 0 {
			errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.base", ErrMustBePositive)
		}

		if cmd.ConcentratedLiquidityParameters.LowerBound != nil {
			haveLower = true
			if lowerBound, _ = big.NewInt(0).SetString(*cmd.ConcentratedLiquidityParameters.LowerBound, 10); lowerBound == nil {
				errs.FinalAddForProperty("amend_amm.concentrated_liquidity_parameters.lower_bound", ErrIsNotValidNumber)
			} else if lowerBound.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.lower_bound", ErrMustBePositive)
			}
		}
		if cmd.ConcentratedLiquidityParameters.UpperBound != nil {
			haveUpper = true
			if upperBound, _ = big.NewInt(0).SetString(*cmd.ConcentratedLiquidityParameters.UpperBound, 10); upperBound == nil {
				errs.FinalAddForProperty("amend_amm.concentrated_liquidity_parameters.upper_bound", ErrIsNotValidNumber)
			} else if upperBound.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.upper_bound", ErrMustBePositive)
			}
		}

		if !haveLower && !haveUpper {
			errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.lower_bound", errors.New("lower_bound and upper_bound cannot both be empty"))
		}

		if base != nil && lowerBound != nil && base.Cmp(lowerBound) <= 0 {
			errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.base", errors.New("should be a bigger value than lower_bound"))
		}

		if base != nil && upperBound != nil && base.Cmp(upperBound) >= 0 {
			errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.base", errors.New("should be a smaller value than upper_bound"))
		}

		if cmd.ConcentratedLiquidityParameters.LeverageAtUpperBound != nil {
			if leverage, err := num.DecimalFromString(*cmd.ConcentratedLiquidityParameters.LeverageAtUpperBound); err != nil {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.leverage_at_upper_bound", ErrIsNotValidNumber)
			} else if leverage.LessThan(num.DecimalZero()) {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.leverage_at_upper_bound", ErrMustBePositive)
			}
		}

		if cmd.ConcentratedLiquidityParameters.LeverageAtLowerBound != nil {
			if leverage, err := num.DecimalFromString(*cmd.ConcentratedLiquidityParameters.LeverageAtLowerBound); err != nil {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.leverage_at_lower_bound", ErrIsNotValidNumber)
			} else if leverage.LessThan(num.DecimalZero()) {
				errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.leverage_at_lower_bound", ErrMustBePositive)
			}
		}

		if len(cmd.SlippageTolerance) <= 0 {
			errs.AddForProperty("amend_amm.slippage_tolerance", ErrIsRequired)
		} else if slippageTolerance, err := num.DecimalFromString(cmd.SlippageTolerance); err != nil {
			errs.AddForProperty("amend_amm.slippage_tolerance", ErrIsNotValidNumber)
		} else if slippageTolerance.LessThan(num.DecimalZero()) {
			errs.AddForProperty("amend_amm.slippage_tolerance", ErrMustBePositive)
		}

		if cmd.ConcentratedLiquidityParameters.DataSourceId == nil && emptyBase {
			errs.AddForProperty("amend_amm.concentrated_liquidity_parameters.base", ErrIsRequired)
		}

		if cmd.ConcentratedLiquidityParameters.DataSourceId != nil && !IsVegaID(*cmd.ConcentratedLiquidityParameters.DataSourceId) {
			errs.AddForProperty("amend_amm.data_source_id", ErrShouldBeAValidVegaID)
		}
	}

	// no update, but also no error, invalid
	if !hasUpdate && errs.Empty() {
		errs.FinalAdd(ErrNoUpdatesProvided)
	}

	return errs
}
