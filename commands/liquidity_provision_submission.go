package commands

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"

	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

var (
	ErrOrderInShapeWithoutReference         = errors.New("order in shape without reference")
	ErrOrderInShapeWithoutProportion        = errors.New("order in shape without a proportion")
	ErrOrderInBuySideShapeWithBestAskPrice  = errors.New("order in buy side shape with best ask price reference")
	ErrOrderInBuySideShapeOffsetInf0        = errors.New("order in buy side shape offset must be >= 0")
	ErrOrderInBuySideShapeOffsetInfEq0      = errors.New("order in buy side shape offset must be > 0")
	ErrOrderInSellSideShapeOffsetInf0       = errors.New("order in sell shape offset must be >= 0")
	ErrOrderInSellSideShapeWithBestBidPrice = errors.New("order in sell side shape with best bid price reference")
	ErrOrderInSellSideShapeOffsetInfEq0     = errors.New("order in sell shape offset must be > 0")
)

func CheckLiquidityProvisionSubmission(cmd *commandspb.LiquidityProvisionSubmission) error {
	return checkLiquidityProvisionSubmission(cmd).ErrorOrNil()
}

func checkLiquidityProvisionSubmission(cmd *commandspb.LiquidityProvisionSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("liquidity_provision_submission", ErrIsRequired)
	}

	if len(cmd.MarketId) <= 0 {
		errs.AddForProperty("liquidity_provision_submission.market_id", ErrIsRequired)
	} else if !IsVegaPubkey(cmd.MarketId) {
		errs.AddForProperty("liquidity_provision_submission.market_id", ErrShouldBeAValidVegaID)
	}

	if len(cmd.Reference) > ReferenceMaxLen {
		errs.AddForProperty("liquidity_provision_submission.reference", ErrReferenceTooLong)
	}

	// if the commitment amount is 0, then the command should be interpreted as
	// a cancellation of the liquidity provision. As a result, the validation
	// shouldn't be made on the rest of the field.
	// However, since the user might by sending an blank command to probe the
	// validation, we want to return as many error message as possible.
	// A cancellation is only valid if a market is specified, and the commitment is
	// 0. In any case the core will consider that as a cancellation, so we return
	// the error that we go from the market id check.

	if len(cmd.CommitmentAmount) > 0 {
		if commitment, ok := big.NewInt(0).SetString(cmd.CommitmentAmount, 10); !ok {
			errs.AddForProperty("liquidity_provision_submission.commitment_amount", ErrNotAValidInteger)
		} else if commitment.Cmp(big.NewInt(0)) <= 0 {
			return errs.FinalAddForProperty("liquidity_provision_submission.commitment_amount", ErrIsNotValidNumber)
		}
	} else {
		errs.AddForProperty("liquidity_provision_submission.commitment_amount", ErrIsRequired)
	}

	if len(cmd.Fee) <= 0 {
		errs.AddForProperty("liquidity_provision_submission.fee", ErrIsRequired)
	} else {
		if fee, err := strconv.ParseFloat(cmd.Fee, 64); err != nil {
			errs.AddForProperty(
				"liquidity_provision_submission.fee",
				ErrIsNotValid,
			)
		} else if fee < 0 {
			errs.AddForProperty("liquidity_provision_submission.fee", ErrMustBePositive)
		}
	}

	errs.Merge(checkLiquidityProvisionShape(cmd.Buys, types.Side_SIDE_BUY, false))
	errs.Merge(checkLiquidityProvisionShape(cmd.Sells, types.Side_SIDE_SELL, false))

	return errs
}

func checkLiquidityProvisionShape(
	orders []*types.LiquidityOrder, side types.Side, isAmendment bool,
) Errors {
	var (
		errs           = NewErrors()
		shapeSideField = "liquidity_provision_submission.buys"
	)
	if side == types.Side_SIDE_SELL {
		shapeSideField = "liquidity_provision_submission.sells"
	}

	if len(orders) <= 0 && !isAmendment {
		errs.AddForProperty(shapeSideField, errors.New("empty shape"))
		return errs
	}

	zero := big.NewInt(0)

	for idx, order := range orders {
		if _, ok := types.PeggedReference_name[int32(order.Reference)]; !ok {
			errs.AddForProperty(
				fmt.Sprintf("%v.%d.reference", shapeSideField, idx),
				ErrIsNotValid,
			)
		}
		if order.Reference == types.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED {
			errs.AddForProperty(
				fmt.Sprintf("%v.%d.reference", shapeSideField, idx),
				ErrOrderInShapeWithoutReference,
			)
		}
		if order.Proportion == 0 {
			errs.AddForProperty(
				fmt.Sprintf("%v.%d.proportion", shapeSideField, idx),
				ErrOrderInShapeWithoutProportion,
			)
		}

		if side == types.Side_SIDE_BUY {
			switch order.Reference {
			case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
				errs.AddForProperty(
					fmt.Sprintf("%v.%d.reference", shapeSideField, idx),
					ErrOrderInBuySideShapeWithBestAskPrice,
				)
			case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
				if offset, ok := big.NewInt(0).SetString(order.Offset, 10); !ok {
					errs.AddForProperty(
						fmt.Sprintf("%v.%d.offset", shapeSideField, idx),
						ErrNotAValidInteger,
					)
				} else if offset.Cmp(zero) == -1 {
					errs.AddForProperty(
						fmt.Sprintf("%v.%d.offset", shapeSideField, idx),
						ErrOrderInBuySideShapeOffsetInf0,
					)
				}
			case types.PeggedReference_PEGGED_REFERENCE_MID:
				if offset, ok := big.NewInt(0).SetString(order.Offset, 10); !ok {
					errs.AddForProperty(
						fmt.Sprintf("%v.%d.offset", shapeSideField, idx),
						ErrNotAValidInteger,
					)
				} else if offset.Cmp(zero) <= 0 {
					errs.AddForProperty(
						fmt.Sprintf("%v.%d.offset", shapeSideField, idx),
						ErrOrderInBuySideShapeOffsetInfEq0,
					)
				}
			}
		} else {
			switch order.Reference {
			case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
				if offset, ok := big.NewInt(0).SetString(order.Offset, 10); !ok {
					errs.AddForProperty(
						fmt.Sprintf("%v.%d.offset", shapeSideField, idx),
						ErrNotAValidInteger,
					)
				} else if offset.Cmp(zero) == -1 {
					errs.AddForProperty(
						fmt.Sprintf("%v.%d.offset", shapeSideField, idx),
						ErrOrderInSellSideShapeOffsetInf0,
					)
				}
			case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
				errs.AddForProperty(
					fmt.Sprintf("%v.%d.offset", shapeSideField, idx),
					ErrOrderInSellSideShapeWithBestBidPrice,
				)
			case types.PeggedReference_PEGGED_REFERENCE_MID:
				if offset, ok := big.NewInt(0).SetString(order.Offset, 10); !ok {
					errs.AddForProperty(
						fmt.Sprintf("%v.%d.offset", shapeSideField, idx),
						ErrNotAValidInteger,
					)
				} else if offset.Cmp(zero) <= 0 {
					errs.AddForProperty(
						fmt.Sprintf("%v.%d.offset", shapeSideField, idx),
						ErrOrderInSellSideShapeOffsetInfEq0,
					)
				}
			}
		}
	}
	return errs
}

func CheckLiquidityProvisionCancellation(cmd *commandspb.LiquidityProvisionCancellation) error {
	return checkLiquidityProvisionCancellation(cmd).ErrorOrNil()
}

func checkLiquidityProvisionCancellation(cmd *commandspb.LiquidityProvisionCancellation) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("liquidity_provision_cancellation", ErrIsRequired)
	}

	if len(cmd.MarketId) <= 0 {
		return errs.FinalAddForProperty("liquidity_provision_cancellation.market_id", ErrIsRequired)
	} else if !IsVegaPubkey(cmd.MarketId) {
		errs.AddForProperty("liquidity_provision_cancellation.market_id", ErrShouldBeAValidVegaID)
	}

	return errs
}

func CheckLiquidityProvisionAmendment(cmd *commandspb.LiquidityProvisionAmendment) error {
	return checkLiquidityProvisionAmendment(cmd).ErrorOrNil()
}

func checkLiquidityProvisionAmendment(cmd *commandspb.LiquidityProvisionAmendment) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("liquidity_provision_amendment", ErrIsRequired)
	}

	if len(cmd.MarketId) <= 0 {
		return errs.FinalAddForProperty("liquidity_provision_amendment.market_id", ErrIsRequired)
	}

	if len(cmd.CommitmentAmount) <= 0 &&
		len(cmd.Fee) <= 0 &&
		len(cmd.Sells) <= 0 &&
		len(cmd.Buys) <= 0 &&
		len(cmd.Reference) <= 0 {
		return errs.FinalAddForProperty("liquidity_provision_amendment", ErrIsRequired)
	}

	if len(cmd.Reference) > ReferenceMaxLen {
		errs.AddForProperty("liquidity_provision_amendment.reference", ErrReferenceTooLong)
	}

	errs.Merge(checkLiquidityProvisionShape(cmd.Buys, types.Side_SIDE_BUY, true))
	errs.Merge(checkLiquidityProvisionShape(cmd.Sells, types.Side_SIDE_SELL, true))

	return errs
}
