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

package future

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/core/types"
)

var ErrCommitmentAmountTooLow = errors.New("commitment amount is too low")

// SubmitLiquidityProvision forwards a LiquidityProvisionSubmission to the Liquidity Engine.
func (m *Market) SubmitLiquidityProvision(
	ctx context.Context,
	sub *types.LiquidityProvisionSubmission,
	party, deterministicID string,
) error {
	defer m.onTxProcessed()

	// add the party to the list of all parties involved with
	// this market
	m.addParty(party)

	_, err := m.collateral.CreatePartyMarginAccount(ctx, party, m.GetID(), m.settlementAsset)
	if err != nil {
		return err
	}

	return m.liquidity.SubmitLiquidityProvision(ctx, sub, party, deterministicID, m.GetMarketState())
}

// AmendLiquidityProvision forwards a LiquidityProvisionAmendment to the Liquidity Engine.
func (m *Market) AmendLiquidityProvision(ctx context.Context, lpa *types.LiquidityProvisionAmendment, party string, deterministicID string) (err error) {
	defer m.onTxProcessed()

	return m.liquidity.AmendLiquidityProvision(ctx, lpa, party, deterministicID, m.GetMarketState())
}

// CancelLiquidityProvision forwards a LiquidityProvisionCancel to the Liquidity Engine.
func (m *Market) CancelLiquidityProvision(ctx context.Context, cancel *types.LiquidityProvisionCancellation, party string) (err error) {
	defer m.onTxProcessed()

	return m.liquidity.CancelLiquidityProvision(ctx, party)
}
