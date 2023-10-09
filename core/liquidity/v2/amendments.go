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

package liquidity

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
)

var ErrPartyHaveNoLiquidityProvision = errors.New("party have no liquidity provision")

func (e *Engine) AmendLiquidityProvision(
	ctx context.Context,
	lpa *types.LiquidityProvisionAmendment,
	party string,
	isCancel bool,
) (bool, error) {
	if err := e.CanAmend(lpa, party, !isCancel); err != nil {
		return false, err
	}

	// LP exists, checked in the previous func.
	lp, _ := e.provisions.Get(party)
	if lp == nil {
		lp, _ = e.pendingProvisions.Get(party)
	}

	// If we are cancelling the LP, preserve the reference field
	if lpa.CommitmentAmount.IsZero() {
		lpa.Reference = lp.Reference
	}

	updatedLp := e.createAmendedProvision(lp, lpa)

	// add to pending provision since the change in CommitmentAmount should be reflected at the beginning of next epoch
	// if it's not opening auction.
	if lp.CommitmentAmount.NEQ(lpa.CommitmentAmount) && !e.auctionState.IsOpeningAuction() {
		e.pendingProvisions.Set(updatedLp)
		e.broker.Send(events.NewLiquidityProvisionEvent(ctx, updatedLp))
		return false, nil
	}

	// cancel immediately during opening auction.
	if e.auctionState.IsOpeningAuction() && isCancel {
		if err := e.CancelLiquidityProvision(ctx, party); err != nil {
			return false, err
		}

		return true, nil
	}

	// update immediately since either the commitment amount has not changed.
	updatedLp.Status = types.LiquidityProvisionStatusActive
	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, updatedLp))
	e.provisions.Set(party, updatedLp)
	return true, nil
}

func (e *Engine) createAmendedProvision(
	currentProvision *types.LiquidityProvision,
	amendment *types.LiquidityProvisionAmendment,
) *types.LiquidityProvision {
	return &types.LiquidityProvision{
		ID:               currentProvision.ID,
		MarketID:         currentProvision.MarketID,
		Party:            currentProvision.Party,
		CreatedAt:        currentProvision.CreatedAt,
		Status:           types.LiquidityProvisionStatusPending,
		Fee:              amendment.Fee,
		Reference:        amendment.Reference,
		Version:          currentProvision.Version + 1,
		CommitmentAmount: amendment.CommitmentAmount.Clone(),
		UpdatedAt:        e.timeService.GetTimeNow().UnixNano(),
	}
}

func (e *Engine) CanAmend(lps *types.LiquidityProvisionAmendment, party string, shouldValidate bool) error {
	if !e.IsLiquidityProvider(party) {
		return ErrPartyHaveNoLiquidityProvision
	}

	if !shouldValidate {
		return nil
	}

	if err := e.ValidateLiquidityProvisionAmendment(lps); err != nil {
		return err
	}

	return nil
}

func (e *Engine) ValidateLiquidityProvisionAmendment(lp *types.LiquidityProvisionAmendment) error {
	if lp.Fee.IsZero() && (lp.CommitmentAmount == nil || lp.CommitmentAmount.IsZero()) {
		return errors.New("empty liquidity provision amendment content")
	}

	// If orders fee is provided, we need it to be valid
	if lp.Fee.IsNegative() || lp.Fee.GreaterThan(e.maxFee) {
		return errors.New("invalid liquidity provision fee")
	}

	return nil
}
