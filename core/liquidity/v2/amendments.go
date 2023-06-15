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

func (e *Engine) AmendLiquidityProvision(
	ctx context.Context,
	lpa *types.LiquidityProvisionAmendment,
	party string,
) error {
	if err := e.CanAmend(lpa, party); err != nil {
		return err
	}

	// LP exists, checked in the previous func
	lp, _ := e.provisions.Get(party)

	// update the LP
	lp.UpdatedAt = e.timeService.GetTimeNow().UnixNano()
	lp.CommitmentAmount = lpa.CommitmentAmount.Clone()
	lp.Fee = lpa.Fee
	lp.Reference = lpa.Reference

	// update version
	lp.Version++

	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))
	e.provisions.Set(party, lp)
	return nil
}

func (e *Engine) CanAmend(
	lps *types.LiquidityProvisionAmendment,
	party string,
) error {
	if !e.IsLiquidityProvider(party) {
		return ErrPartyHaveNoLiquidityProvision
	}

	if err := e.ValidateLiquidityProvisionAmendment(lps); err != nil {
		return err
	}

	return nil
}

func (e *Engine) ValidateLiquidityProvisionAmendment(lp *types.LiquidityProvisionAmendment) (err error) {
	if lp.Fee.IsZero() && (lp.CommitmentAmount == nil || lp.CommitmentAmount.IsZero()) {
		return errors.New("empty liquidity provision amendment content")
	}

	// If orders fee is provided, we need it to be valid
	if lp.Fee.IsNegative() || lp.Fee.GreaterThan(e.maxFee) {
		return errors.New("invalid liquidity provision fee")
	}

	return nil
}
