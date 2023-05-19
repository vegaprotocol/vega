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

package execution

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

func (m *Market) checkBondBalance(ctx context.Context) {
	lps := m.liquidity.ProvisionsPerParty().Slice()
	mID := m.GetID()
	transfers := make([]*types.LedgerMovement, 0, len(lps))
	for _, lp := range lps {
		party := lp.Party
		bondAcc, err := m.collateral.GetPartyBondAccount(mID, party, m.settlementAsset)
		if err != nil || bondAcc == nil {
			continue
		}
		// commitment is covered by bond balance already
		if bondAcc.Balance.GTE(lp.CommitmentAmount) {
			continue
		}
		gen, err := m.collateral.GetPartyGeneralAccount(party, m.settlementAsset)
		// no balance in general account
		if err != nil || gen.Balance.IsZero() {
			continue
		}
		bondShort := num.UintZero().Sub(lp.CommitmentAmount, bondAcc.Balance)
		// Min clones
		amt := num.Min(bondShort, gen.Balance)
		t := &types.Transfer{
			Owner: party,
			Type:  types.TransferTypeBondLow,
			Amount: &types.FinancialAmount{
				Asset:  m.settlementAsset,
				Amount: amt,
			},
			MinAmount: amt.Clone(),
		}
		resp, err := m.collateral.BondUpdate(ctx, mID, t)
		if err != nil {
			m.log.Panic("Failed to top up bond balance",
				logging.String("market-id", mID),
				logging.String("party", party),
				logging.Error(err))
		}
		if len(resp.Entries) > 0 {
			transfers = append(transfers, resp)
		}
	}
	if len(transfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, transfers))
	}
}
