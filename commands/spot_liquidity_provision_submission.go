package commands

import (
	"math/big"
	"strconv"

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
		return errs.FinalAddForProperty("spot_liquidity_provision_cancellation.market_id", ErrIsRequired)
	} else if !IsVegaPubkey(cmd.MarketId) {
		errs.AddForProperty("spot_liquidity_provision_cancellation.market_id", ErrShouldBeAValidVegaID)
	}

	return errs
}

func CheckSpotLiquidityProvisionAmendment(cmd *commandspb.SpotLiquidityProvisionAmendment) error {
	return checkSpotLiquidityProvisionAmendment(cmd).ErrorOrNil()
}

func checkSpotLiquidityProvisionAmendment(cmd *commandspb.SpotLiquidityProvisionAmendment) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("spot_liquidity_provision_amendment", ErrIsRequired)
	}

	if len(cmd.MarketId) <= 0 {
		return errs.FinalAddForProperty("spot_liquidity_provision_amendment.market_id", ErrIsRequired)
	}

	if cmd.BuyCommitmentAmount == nil && cmd.SellCommitmentAmount == nil && cmd.Fee == nil {
		return errs.FinalAddForProperty("spot_liquidity_provision_amendment", ErrIsRequired)
	}

	return errs
}
