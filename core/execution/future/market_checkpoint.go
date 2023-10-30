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
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

func (m *Market) GetCPState() *types.CPMarketState {
	id := m.mkt.ID
	shares := m.equityShares.GetCPShares()
	// get all LP accounts, we don't have to sort this slice because we're fetching the balances
	// in the same order as we got the ELS shares (which is already a deterministically sorted slice).
	ipb, ok := m.collateral.GetInsurancePoolBalance(id, m.settlementAsset)
	if !ok {
		ipb = num.UintZero()
	}
	ms := types.CPMarketState{
		ID:               id,
		Shares:           shares,
		InsuranceBalance: ipb,
		LastTradeValue:   m.feeSplitter.TradeValue(),
		State:            m.mkt.State,
	}
	// if the market was closed/settled, include the last valid market definition in the checkpoint
	if m.mkt.State == types.MarketStateSettled || m.mkt.State == types.MarketStateClosed {
		ms.Market = m.mkt.DeepClone()
	}
	return &ms
}

func (m *Market) LoadCPState(state *types.CPMarketState) {
	m.mkt = state.Market
	m.feeSplitter.SetTradeValue(state.LastTradeValue)
	m.equityShares.SetCPShares(state.Shares)
	// @TODO bond account and insurance account
}

func (m *Market) SetSuccessorELS(state *types.CPMarketState) {
	// carry over traded value from predecessor
	m.feeSplitter.AddTradeValue(state.LastTradeValue)
	// load equity like shares
	m.equityShares.SetCPShares(state.Shares)
	// @TODO force a recalculation for the LP's who actually are present
}
