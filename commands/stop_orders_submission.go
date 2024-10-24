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
	"math/big"

	"code.vegaprotocol.io/vega/libs/num"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckStopOrdersSubmission(cmd *commandspb.StopOrdersSubmission) error {
	return checkStopOrdersSubmission(cmd).ErrorOrNil()
}

func checkStopOrdersSubmission(cmd *commandspb.StopOrdersSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("stop_orders_submission", ErrIsRequired)
	}

	var market1, market2 string
	if cmd.FallsBelow != nil {
		checkStopOrderSetup(
			"stop_orders_submission.falls_below", errs, cmd.FallsBelow, cmd.RisesAbove != nil)
		if cmd.FallsBelow.OrderSubmission != nil {
			market1 = cmd.FallsBelow.OrderSubmission.MarketId
			if cmd.FallsBelow.OrderSubmission.VaultId != nil && !IsVegaID(*cmd.FallsBelow.OrderSubmission.VaultId) {
				errs.AddForProperty("stop_orders_submission.falls_below.vault_id", ErrInvalidVaultID)
			}
			if cmd.FallsBelow.SizeOverrideSetting != nil {
				if *cmd.FallsBelow.SizeOverrideSetting == types.StopOrder_SIZE_OVERRIDE_SETTING_POSITION {
					if cmd.FallsBelow.SizeOverrideValue != nil {
						value := cmd.FallsBelow.SizeOverrideValue.GetPercentage()
						// Check that the string represents a number between >0&&>=1
						percentage, err := num.DecimalFromString(value)
						if err != nil {
							return errs.FinalAddForProperty("stop_orders_submission.falls_below.size_override_value", ErrIsNotValidNumber)
						}
						if percentage.LessThanOrEqual(num.DecimalFromFloat(0.0)) {
							return errs.FinalAddForProperty("stop_orders_submission.falls_below.size_override_value", ErrMustBeBetween01)
						}
						if percentage.GreaterThan(num.DecimalFromFloat(1.0)) {
							return errs.FinalAddForProperty("stop_orders_submission.falls_below.size_override_value", ErrMustBeBetween01)
						}
					}
				}
			}
		}
	}

	if cmd.RisesAbove != nil {
		checkStopOrderSetup(
			"stop_orders_submission.rises_below", errs, cmd.RisesAbove, cmd.FallsBelow != nil)
		if cmd.RisesAbove.OrderSubmission != nil {
			if cmd.RisesAbove.OrderSubmission.VaultId != nil && !IsVegaID(*cmd.RisesAbove.OrderSubmission.VaultId) {
				errs.AddForProperty("stop_orders_submission.rises_above.vault_id", ErrInvalidVaultID)
			}
			market2 = cmd.RisesAbove.OrderSubmission.MarketId
			if cmd.RisesAbove.SizeOverrideSetting != nil {
				if *cmd.RisesAbove.SizeOverrideSetting == types.StopOrder_SIZE_OVERRIDE_SETTING_POSITION {
					if cmd.RisesAbove.SizeOverrideValue != nil {
						value := cmd.RisesAbove.SizeOverrideValue.GetPercentage()
						// Check that the string represents a number between >0&&>=1
						percentage, err := num.DecimalFromString(value)
						if err != nil {
							return errs.FinalAddForProperty("stop_orders_submission.rises_above.size_override_value", ErrIsNotValidNumber)
						}
						if percentage.LessThanOrEqual(num.DecimalFromFloat(0.0)) {
							return errs.FinalAddForProperty("stop_orders_submission.rises_above.size_override_value", ErrMustBeBetween01)
						}
						if percentage.GreaterThan(num.DecimalFromFloat(1.0)) {
							return errs.FinalAddForProperty("stop_orders_submission.rises_above.size_override_value", ErrMustBeBetween01)
						}
					}
				}
			}
		}
	}

	if cmd.FallsBelow == nil && cmd.RisesAbove == nil {
		return errs.FinalAdd(ErrMustHaveAtLeastOneOfRisesAboveOrFallsBelow)
	}

	if cmd.FallsBelow != nil && cmd.RisesAbove != nil && market1 != market2 {
		return errs.FinalAdd(ErrFallsBelowAndRiseAboveMarketIDMustBeTheSame)
	}

	return errs
}

func checkStopOrderSetup(
	fieldName string,
	errs Errors,
	setup *commandspb.StopOrderSetup,
	isOCO bool,
) {
	if err := checkOrderSubmission(setup.OrderSubmission); !err.Empty() {
		errs.Merge(err)
	}

	if setup.OrderSubmission != nil && setup.OrderSubmission.TimeInForce == types.Order_TIME_IN_FORCE_GFA {
		errs.AddForProperty(fmt.Sprintf("%s.order_submission.time_in_force", fieldName), ErrIsNotValid)
	}

	if setup.ExpiresAt != nil {
		if *setup.ExpiresAt < 0 {
			errs.AddForProperty(fmt.Sprintf("%s.expires_at", fieldName), ErrMustBePositive)
		}
		if setup.ExpiryStrategy == nil {
			errs.AddForProperty(fmt.Sprintf("%s.expiry_strategy", fieldName), ErrExpiryStrategyRequiredWhenExpiresAtSet)
		} else {
			if *setup.ExpiryStrategy == types.StopOrder_EXPIRY_STRATEGY_UNSPECIFIED {
				errs.AddForProperty(fmt.Sprintf("%s.expiry_strategy", fieldName), ErrIsRequired)
			} else if _, ok := types.StopOrder_ExpiryStrategy_name[int32(*setup.ExpiryStrategy)]; !ok {
				errs.AddForProperty(fmt.Sprintf("%s.expiry_strategy", fieldName), ErrIsNotValid)
			} else if isOCO && *setup.ExpiryStrategy == types.StopOrder_EXPIRY_STRATEGY_SUBMIT {
				errs.AddForProperty(fmt.Sprintf("%s.expiry_strategy", fieldName), ErrIsNotValidWithOCO)
			}
		}
	}

	if setup.Trigger != nil {
		switch t := setup.Trigger.(type) {
		case *commandspb.StopOrderSetup_Price:
			if len(t.Price) <= 0 {
				errs.AddForProperty(fmt.Sprintf("%s.trigger.price", fieldName), ErrIsRequired)
			} else {
				if price, ok := big.NewInt(0).SetString(t.Price, 10); !ok {
					errs.AddForProperty(fmt.Sprintf("%s.trigger.price", fieldName), ErrNotAValidInteger)
				} else if price.Cmp(big.NewInt(0)) != 1 {
					errs.AddForProperty(fmt.Sprintf("%s.trigger.price", fieldName), ErrMustBePositive)
				}
			}
		case *commandspb.StopOrderSetup_TrailingPercentOffset:
			if len(t.TrailingPercentOffset) <= 0 {
				errs.AddForProperty(fmt.Sprintf("%s.trigger.trailing_percent_offset", fieldName), ErrIsRequired)
			} else {
				if poffset, err := num.DecimalFromString(t.TrailingPercentOffset); err != nil {
					errs.AddForProperty(fmt.Sprintf("%s.trigger.trailing_percent_offset", fieldName), ErrNotAValidFloat)
				} else if poffset.LessThanOrEqual(num.DecimalZero()) || poffset.GreaterThanOrEqual(num.DecimalOne()) {
					errs.AddForProperty(fmt.Sprintf("%s.trigger.trailing_percent_offset", fieldName), ErrMustBeWithinRange01)
				} else if !poffset.Mod(num.MustDecimalFromString("0.001")).Equal(num.DecimalZero()) {
					errs.AddForProperty(fmt.Sprintf("%s.trigger.trailing_percent_offset", fieldName), ErrTrailingPercentOffsetMinimalIncrementNotReached)
				}
			}
		}
	} else {
		errs.AddForProperty(fmt.Sprintf("%s.trigger", fieldName), ErrMustHaveAStopOrderTrigger)
	}
}
