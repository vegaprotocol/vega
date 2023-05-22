package execution

import (
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

func (m *Market) GetCPState() *types.CPMarketState {
	shares := m.equityShares.getCPShares()
	id := m.mkt.ID
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
		LastTradeValue:   m.feeSplitter.tradeValue.Clone(),
	}
	// if the market was closed/settled, include the last valid market definition in the checkpoint
	if m.mkt.State == types.MarketStateSettled || m.mkt.State == types.MarketStateClosed {
		ms.Market = m.mkt.DeepClone()
	}
	return &ms
}

func (m *Market) LoadCPState(state *types.CPMarketState) {
	m.mkt = state.Market
	m.feeSplitter.tradeValue = state.LastTradeValue.Clone()
	m.equityShares.setCPShares(state.Shares)
	// @TODO bond account and insurance account
}

func (m *Market) SetSuccessorELS(state *types.CPMarketState) {
	// carry over traded value from predecessor
	m.feeSplitter.tradeValue.AddSum(state.LastTradeValue)
	// load equity like shares
	m.equityShares.setCPShares(state.Shares)
	// @TODO force a recalculation for the LP's who actually are present
}
