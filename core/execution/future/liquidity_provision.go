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

	// TODO karel - implement here

	return nil
}
