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

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
)

var ErrPartyHaveNoLiquidityProvision = errors.New("party have no liquidity provision")

func (e *Engine) AmendLiquidityProvision(ctx context.Context, lpa *types.LiquidityProvisionAmendment, party string) error {
	if err := e.CanAmend(lpa, party); err != nil {
		return err
	}

	// LP exists, checked in the previous func.
	lp, _ := e.provisions.Get(party)
	if lp == nil {
		lp, _ = e.pendingProvisions.Get(party)
	}
	updatedLp := e.createAmendedProvision(lp, lpa)

	// add to pending provision since the change in CommitmentAmount should be reflected at the beginning of next epoch.
	if lp.CommitmentAmount.NEQ(lpa.CommitmentAmount) {
		e.pendingProvisions.Set(updatedLp)
		e.broker.Send(events.NewLiquidityProvisionEvent(ctx, updatedLp))
		return nil
	}

	// we can update immediately since the commitment amount has not changed.
	updatedLp.Status = types.LiquidityProvisionStatusActive
	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, updatedLp))
	e.provisions.Set(party, updatedLp)
	return nil
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

func (e *Engine) CanAmend(lps *types.LiquidityProvisionAmendment, party string) error {
	if !e.IsLiquidityProvider(party) {
		return ErrPartyHaveNoLiquidityProvision
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
