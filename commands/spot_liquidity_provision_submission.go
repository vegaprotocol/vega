package commands

import (
	"math/big"
	"strconv"

	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckSpotLiquidityProvisionSubmission(cmd *commandspb.SpotLiquidityProvisionSubmission) error {
	return checkSpotLiquidityProvisionSubmission(cmd).ErrorOrNil()
}

func checkSpotLiquidityProvisionSubmission(cmd *commandspb.SpotLiquidityProvisionSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("spot_liquidity_provision_submission", ErrIsRequired)
	}

	if len(cmd.MarketId) <= 0 {
		errs.AddForProperty("spot_liquidity_provision_submission.market_id", ErrIsRequired)
	} else if !IsVegaPubkey(cmd.MarketId) {
		errs.AddForProperty("spot_liquidity_provision_submission.market_id", ErrShouldBeAValidVegaID)
	}

	if len(cmd.Reference) > ReferenceMaxLen {
		errs.AddForProperty("spot_liquidity_provision_submission.reference", ErrReferenceTooLong)
	}

	// if the commitment amount is 0, then the command should be interpreted as
	// a cancellation of the liquidity provision. As a result, the validation
	// shouldn't be made on the rest of the field.
	// However, since the user might by sending an blank command to probe the
	// validation, we want to return as many error message as possible.
	// A cancellation is only valid if a market is specified, and the commitment is
	// 0. In any case the core will consider that as a cancellation, so we return
	// the error that we go from the market id check.

	if len(cmd.BuyCommitmentAmount) > 0 {
		if commitment, ok := big.NewInt(0).SetString(cmd.BuyCommitmentAmount, 10); !ok {
			errs.AddForProperty("spot_liquidity_provision_submission.buy_commitment_amount", ErrNotAValidInteger)
		} else if commitment.Cmp(big.NewInt(0)) <= 0 {
			return errs.FinalAddForProperty("spot_liquidity_provision_submission.buy_commitment_amount", ErrIsNotValidNumber)
		}
	} else {
		errs.AddForProperty("spot_liquidity_provision_submission.buy_commitment_amount", ErrIsRequired)
	}

	if len(cmd.SellCommitmentAmount) > 0 {
		if commitment, ok := big.NewInt(0).SetString(cmd.SellCommitmentAmount, 10); !ok {
			errs.AddForProperty("spot_liquidity_provision_submission.sell_commitment_amount", ErrNotAValidInteger)
		} else if commitment.Cmp(big.NewInt(0)) <= 0 {
			return errs.FinalAddForProperty("spot_liquidity_provision_submission.sell_commitment_amount", ErrIsNotValidNumber)
		}
	} else {
		errs.AddForProperty("spot_liquidity_provision_submission.sell_commitment_amount", ErrIsRequired)
	}

	if len(cmd.Fee) <= 0 {
		errs.AddForProperty("spot_liquidity_provision_submission.fee", ErrIsRequired)
	} else {
		if fee, err := strconv.ParseFloat(cmd.Fee, 64); err != nil {
			errs.AddForProperty(
				"spot_liquidity_provision_submission.fee",
				ErrIsNotValid,
			)
		} else if fee < 0 {
			errs.AddForProperty("spot_liquidity_provision_submission.fee", ErrMustBePositive)
		}
	}

	errs.Merge(checkLiquidityProvisionShape(cmd.Buys, types.Side_SIDE_BUY, false))
	errs.Merge(checkLiquidityProvisionShape(cmd.Sells, types.Side_SIDE_SELL, false))

	return errs
}

func CheckSpotLiquidityProvisionCancellation(cmd *commandspb.SpotLiquidityProvisionCancellation) error {
	return checkSpotLiquidityProvisionCancellation(cmd).ErrorOrNil()
}

func checkSpotLiquidityProvisionCancellation(cmd *commandspb.SpotLiquidityProvisionCancellation) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("spot_liquidity_provision_cancellation", ErrIsRequired)
	}

	if len(cmd.MarketId) <= 0 {
		return errs.FinalAddForProperty("liquidity_provision_cancellation.market_id", ErrIsRequired)
	} else if !IsVegaPubkey(cmd.MarketId) {
		errs.AddForProperty("liquidity_provision_cancellation.market_id", ErrShouldBeAValidVegaID)
	}

	return errs
}

func CheckSpotLiquidityProvisionAmendment(cmd *commandspb.SpotLiquidityProvisionAmendment) error {
	return checkSpotLiquidityProvisionAmendment(cmd).ErrorOrNil()
}

func checkSpotLiquidityProvisionAmendment(cmd *commandspb.SpotLiquidityProvisionAmendment) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("liquidity_provision_amendment", ErrIsRequired)
	}

	if len(cmd.MarketId) <= 0 {
		return errs.FinalAddForProperty("liquidity_provision_amendment.market_id", ErrIsRequired)
	}

	if (len(cmd.BuyCommitmentAmount) <= 0 || len(cmd.SellCommitmentAmount) <= 0) &&
		len(cmd.Fee) <= 0 &&
		len(cmd.Sells) <= 0 &&
		len(cmd.Buys) <= 0 &&
		len(cmd.Reference) <= 0 {
		return errs.FinalAddForProperty("spot_liquidity_provision_amendment", ErrIsRequired)
	}

	if len(cmd.Reference) > ReferenceMaxLen {
		errs.AddForProperty("liquidity_provision_amendment.reference", ErrReferenceTooLong)
	}

	errs.Merge(checkLiquidityProvisionShape(cmd.Buys, types.Side_SIDE_BUY, true))
	errs.Merge(checkLiquidityProvisionShape(cmd.Sells, types.Side_SIDE_SELL, true))

	return errs
}
