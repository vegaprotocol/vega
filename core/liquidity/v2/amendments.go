// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package liquidity

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

var ErrPartyHaveNoLiquidityProvision = errors.New("party have no liquidity provision")

func (e *Engine) CanAmend(
	lps *types.LiquidityProvisionAmendment,
	party string,
) error {
	// does the party is an LP
	_, ok := e.provisions.Get(party)
	if !ok {
		return ErrPartyHaveNoLiquidityProvision
	}

	// is the new amendment valid?
	if err := e.ValidateLiquidityProvisionAmendment(lps); err != nil {
		return err
	}

	// yes
	return nil
}

func (e *Engine) getProposedCommitmentVariation(currentCommitment, newCommitment *num.Uint) *num.Uint {
	return num.UintZero().Sub(currentCommitment, newCommitment)
}

// TODO karel - this should be probably moved to market itself or some other layer there
// handles potential amendment fees and returns and error when amendment is not allowed
func (e *Engine) handleAmendmentFees(currentCommitment, newCommitment *num.Uint) error {
	zero := num.UintZero()
	pcv := e.getProposedCommitmentVariation(currentCommitment, newCommitment)

	// increase commitment
	if pcv.GTE(zero) {
		// check if they have sufficient collateral if yes then we are done
		// recalculate ELS
		sufficientCollateral := true // this should be a real function call
		if sufficientCollateral {
			// recalculate ELS
			return nil
		}
		return fmt.Errorf("not enough collateral to amend the commitment")

	}

	// decrease commitment
	maxPenaltyFreeReductionAmount := num.UintZero().Sub(e.CalculateSuppliedStake(), e.getTargetStake())
	mOneMulPcv := num.NewDecimalFromFloat(-1).Mul(pcv.ToDecimal())

	if mOneMulPcv.LessThanOrEqual(maxPenaltyFreeReductionAmount.ToDecimal()) {
		// done - just recalculate the ELS
		return nil
	}

	// fill this
	return nil
}

func (e *Engine) AmendLiquidityProvision(
	ctx context.Context,
	lpa *types.LiquidityProvisionAmendment,
	party string,
	idGen IDGen,
) error {
	if err := e.CanAmend(lpa, party); err != nil {
		return err
	}

	// LP exists, checked in the previous func
	lp, _ := e.provisions.Get(party)

	now := e.timeService.GetTimeNow().UnixNano()

	if lp.CommitmentAmount.EQ(lp.CommitmentAmount) {
		return nil
	}

	// TODO karel - this should be probably moved to market itself or some other layer there
	e.handleAmendmentFees(lp.CommitmentAmount, lp.CommitmentAmount)

	// update the LP
	lp.UpdatedAt = now
	lp.CommitmentAmount = lpa.CommitmentAmount.Clone()
	lp.Fee = lpa.Fee
	lp.Reference = lpa.Reference

	// update version
	lp.Version++

	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))
	e.provisions.Set(party, lp)
	return nil
}

func (e *Engine) ValidateLiquidityProvisionAmendment(lp *types.LiquidityProvisionAmendment) (err error) {
	if lp.Fee.IsZero() && !lp.ContainsOrders() && (lp.CommitmentAmount == nil || lp.CommitmentAmount.IsZero()) {
		return errors.New("empty liquidity provision amendment content")
	}

	// If orders fee is provided, we need it to be valid
	if lp.Fee.IsNegative() || lp.Fee.GreaterThan(e.maxFee) {
		return errors.New("invalid liquidity provision fee")
	}

	return nil
}
