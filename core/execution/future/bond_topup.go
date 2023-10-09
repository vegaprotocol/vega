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

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

func (m *Market) checkBondBalance(ctx context.Context) {
	lps := m.liquidityEngine.ProvisionsPerParty().Slice()
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
