package commands

import (
	"errors"
	"math/big"
	"strconv"

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
	} else if !IsVegaID(cmd.MarketId) {
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
	} else if !IsVegaID(cmd.MarketId) {
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
	return errs
}
