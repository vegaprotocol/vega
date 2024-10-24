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

func CheckSubmitAMM(cmd *commandspb.SubmitAMM) error {
	return checkSubmitAMM(cmd).ErrorOrNil()
}

func checkSubmitAMM(cmd *commandspb.SubmitAMM) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("submit_amm", ErrIsRequired)
	}

	if len(cmd.MarketId) <= 0 {
		errs.AddForProperty("submit_amm.market_id", ErrIsRequired)
	} else if !IsVegaID(cmd.MarketId) {
		errs.AddForProperty("submit_amm.market_id", ErrShouldBeAValidVegaID)
	}

	if len(cmd.SlippageTolerance) <= 0 {
		errs.AddForProperty("submit_amm.slippage_tolerance", ErrIsRequired)
	} else if slippageTolerance, err := num.DecimalFromString(cmd.SlippageTolerance); err != nil {
		errs.AddForProperty("submit_amm.slippage_tolerance", ErrIsNotValidNumber)
	} else if slippageTolerance.LessThanOrEqual(num.DecimalZero()) || slippageTolerance.GreaterThan(num.DecimalOne()) {
		errs.AddForProperty("submit_amm.slippage_tolerance", ErrMustBeBetween01)
	}

	if len(cmd.CommitmentAmount) <= 0 {
		errs.FinalAddForProperty("submit_amm.commitment_amount", ErrIsRequired)
	} else if amount, _ := big.NewInt(0).SetString(cmd.CommitmentAmount, 10); amount == nil {
		errs.FinalAddForProperty("submit_amm.commitment_amount", ErrIsNotValidNumber)
	} else if amount.Cmp(big.NewInt(0)) <= 0 {
		errs.AddForProperty("submit_amm.commitment_amount", ErrMustBePositive)
	}

	if len(cmd.ProposedFee) <= 0 {
		errs.AddForProperty("submit_amm.proposed_fee", ErrIsRequired)
	} else if proposedFee, err := num.DecimalFromString(cmd.ProposedFee); err != nil {
		errs.AddForProperty("submit_amm.proposed_fee", ErrIsNotValid)
	} else if proposedFee.LessThanOrEqual(num.DecimalZero()) {
		errs.AddForProperty("submit_amm.proposed_fee", ErrMustBePositive)
	}

	if cmd.MinimumPriceChangeTrigger != nil {
		if minPriceChange, err := num.DecimalFromString(*cmd.MinimumPriceChangeTrigger); err != nil {
			errs.AddForProperty("submit_amm.mimimum_price_change_trigger", ErrIsNotValid)
		} else if minPriceChange.LessThan(num.DecimalZero()) {
			errs.AddForProperty("submit_amm.proposed_fee", ErrMustBePositiveOrZero)
		}
	}

	if cmd.ConcentratedLiquidityParameters == nil {
		errs.FinalAddForProperty("submit_amm.concentrated_liquidity_parameters", ErrIsRequired)
	} else {
		var emptyLower, emptyUpper, emptyBase bool
		var base, lowerBound, upperBound *big.Int

		if len(cmd.ConcentratedLiquidityParameters.Base) <= 0 {
			emptyBase = true
		} else if base, _ = big.NewInt(0).SetString(cmd.ConcentratedLiquidityParameters.Base, 10); base == nil {
			errs.FinalAddForProperty("submit_amm.concentrated_liquidity_parameters.base", ErrIsNotValidNumber)
		} else if base.Cmp(big.NewInt(0)) <= 0 {
			errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.base", ErrMustBePositive)
		}

		if cmd.ConcentratedLiquidityParameters.LowerBound == nil {
			emptyLower = true
		} else if len(*cmd.ConcentratedLiquidityParameters.LowerBound) <= 0 {
			errs.FinalAddForProperty("submit_amm.concentrated_liquidity_parameters.lower_bound", ErrIsNotValidNumber)
		} else if lowerBound, _ = big.NewInt(0).SetString(*cmd.ConcentratedLiquidityParameters.LowerBound, 10); lowerBound == nil {
			errs.FinalAddForProperty("submit_amm.concentrated_liquidity_parameters.lower_bound", ErrIsNotValidNumber)
		} else if lowerBound.Cmp(big.NewInt(0)) <= 0 {
			errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.lower_bound", ErrMustBePositive)
		}

		if cmd.ConcentratedLiquidityParameters.UpperBound == nil {
			emptyUpper = true
		} else if len(*cmd.ConcentratedLiquidityParameters.UpperBound) <= 0 {
			errs.FinalAddForProperty("submit_amm.concentrated_liquidity_parameters.upper_bound", ErrIsNotValidNumber)
		} else if upperBound, _ = big.NewInt(0).SetString(*cmd.ConcentratedLiquidityParameters.UpperBound, 10); upperBound == nil {
			errs.FinalAddForProperty("submit_amm.concentrated_liquidity_parameters.upper_bound", ErrIsNotValidNumber)
		} else if upperBound.Cmp(big.NewInt(0)) <= 0 {
			errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.upper_bound", ErrMustBePositive)
		}

		if emptyLower && emptyUpper {
			errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.lower_bound", errors.New("lower_bound and upper_bound cannot both be empty"))
		}

		if cmd.ConcentratedLiquidityParameters.LeverageAtUpperBound != nil {
			if len(*cmd.ConcentratedLiquidityParameters.LeverageAtUpperBound) <= 0 {
				errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.leverage_at_upper_bound", ErrIsNotValidNumber)
			} else if leverage, err := num.DecimalFromString(*cmd.ConcentratedLiquidityParameters.LeverageAtUpperBound); err != nil {
				errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.leverage_at_upper_bound", ErrIsNotValidNumber)
			} else if leverage.LessThan(num.DecimalZero()) {
				errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.leverage_at_upper_bound", ErrMustBePositive)
			}
		}

		if cmd.ConcentratedLiquidityParameters.LeverageAtLowerBound != nil {
			if len(*cmd.ConcentratedLiquidityParameters.LeverageAtLowerBound) <= 0 {
				errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.leverage_at_lower_bound", ErrIsNotValidNumber)
			} else if leverage, err := num.DecimalFromString(*cmd.ConcentratedLiquidityParameters.LeverageAtLowerBound); err != nil {
				errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.leverage_at_lower_bound", ErrIsNotValidNumber)
			} else if leverage.LessThan(num.DecimalZero()) {
				errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.leverage_at_lower_bound", ErrMustBePositive)
			}
		}

		if len(cmd.SlippageTolerance) <= 0 {
			errs.AddForProperty("submit_amm.slippage_tolerance", ErrIsRequired)
		} else if slippageTolerance, err := num.DecimalFromString(cmd.SlippageTolerance); err != nil {
			errs.AddForProperty("submit_amm.slippage_tolerance", ErrIsNotValidNumber)
		} else if slippageTolerance.LessThan(num.DecimalZero()) {
			errs.AddForProperty("submit_amm.slippage_tolerance", ErrMustBePositive)
		}

		// Base is <= to lower bound == error
		if base != nil && lowerBound != nil && base.Cmp(lowerBound) <= 0 {
			errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.base", errors.New("should be a bigger value than lower_bound"))
		}

		// base is >= to upper bound == error
		if base != nil && upperBound != nil && base.Cmp(upperBound) >= 0 {
			errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.base", errors.New("should be a smaller value than upper_bound"))
		}

		if cmd.ConcentratedLiquidityParameters.DataSourceId == nil && emptyBase {
			errs.AddForProperty("submit_amm.concentrated_liquidity_parameters.base", ErrIsRequired)
		}

		if cmd.ConcentratedLiquidityParameters.DataSourceId != nil && !IsVegaID(*cmd.ConcentratedLiquidityParameters.DataSourceId) {
			errs.AddForProperty("submit_amm.data_source_id", ErrShouldBeAValidVegaID)
		}
	}

	return errs
}
