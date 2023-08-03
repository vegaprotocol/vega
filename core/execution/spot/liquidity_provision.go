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

package spot

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/core/types"
)

var ErrCommitmentAmountTooLow = errors.New("commitment amount is too low")

// SubmitLiquidityProvision forwards a LiquidityProvisionSubmission to the Liquidity Engine.
func (m *Market) SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, deterministicID string) error {
	if err := m.liquidity.SubmitLiquidityProvision(ctx, sub, party, deterministicID, m.mkt.State); err != nil {
		return err
	}

	// add the party to the list of all parties involved with
	// this market
	m.addParty(party)
	return nil
}

// AmendLiquidityProvision forwards a LiquidityProvisionAmendment to the Liquidity Engine.
func (m *Market) AmendLiquidityProvision(ctx context.Context, lpa *types.LiquidityProvisionAmendment, party string, deterministicID string) error {
	return m.liquidity.AmendLiquidityProvision(ctx, lpa, party, deterministicID, m.mkt.State)
}

// CancelLiquidityProvision forwards a LiquidityProvisionCancel to the Liquidity Engine.
func (m *Market) CancelLiquidityProvision(ctx context.Context, cancel *types.LiquidityProvisionCancellation, party string) error {
	err := m.liquidity.CancelLiquidityProvision(ctx, party)
	m.updateLiquidityFee(ctx)
	// and remove the party from the equity share like calculation
	m.equityShares.SetPartyStake(party, nil)
	// force update of shares so they are updated for all
	// _ = m.equityShares.SharesExcept(m.liquidity.GetInactiveParties())

	return err
}
