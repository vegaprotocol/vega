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
