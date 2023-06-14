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
			"stop_orders_submission.falls_below", errs, cmd.FallsBelow)
		if cmd.FallsBelow.OrderSubmission != nil {
			market1 = cmd.FallsBelow.OrderSubmission.MarketId
		}
	}

	if cmd.RisesAbove != nil {
		checkStopOrderSetup(
			"stop_orders_submission.rises_below", errs, cmd.RisesAbove)
		if cmd.RisesAbove.OrderSubmission != nil {
			market2 = cmd.RisesAbove.OrderSubmission.MarketId
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
	setup *commandspb.StopOrderSetup) Errors {
	if err := checkOrderSubmission(setup.OrderSubmission); !err.Empty() {
		errs.Merge(err)
	} else {
		if !setup.OrderSubmission.ReduceOnly {
			errs.AddForProperty(fmt.Sprintf("%s.order_submission.reduce_only", fieldName), ErrMustBeReduceOnly)
		}
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
			}
			if _, ok := types.StopOrder_ExpiryStrategy_name[int32(*setup.ExpiryStrategy)]; !ok {
				errs.AddForProperty(fmt.Sprintf("%s.expiry_strategy", fieldName), ErrIsNotValid)
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
					errs.AddForProperty(fmt.Sprintf("%s.trigger.trailing_percent_offset", fieldName), ErrTrailingPercentOffsetMinimalIncrementMustBe)
				}
			}
		}
	} else {
		errs.AddForProperty(fmt.Sprintf("%s.trigger", fieldName), ErrMustHaveAStopOrderTrigger)
	}

	return errs
}
