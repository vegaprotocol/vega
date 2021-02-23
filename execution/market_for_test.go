package execution

import (
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

// GetPeggedOrderCount returns the number of pegged orders in the market
func (m *Market) GetPeggedOrderCount() int {
	return len(m.peggedOrders)
}

// GetParkedOrderCount returns hte number of parked orders in the market
func (m *Market) GetParkedOrderCount() int {
	var count int
	for _, order := range m.peggedOrders {
		if order.Status == types.Order_STATUS_PARKED {
			count++
		}
	}
	return count
}

// GetPeggedExpiryOrderCount returns the number of pegged order that can expire
func (m *Market) GetPeggedExpiryOrderCount() int {
	return m.expiringOrders.GetExpiryingOrderCount()
}

// GetOrdersOnBookCount returns the number of orders on the live book
func (m *Market) GetOrdersOnBookCount() int64 {
	return m.matching.GetTotalNumberOfOrders()
}

// StartPriceAuction initialises the market to handle a price auction
func (m *Market) StartPriceAuction(now time.Time) {
	end := types.AuctionDuration{
		Duration: 1000,
	}
	// setup auction
	m.as.StartPriceAuction(now, &end)
}

// TSCalc returns the local tsCalc instance
func (m *Market) TSCalc() TargetStakeCalculator {
	return m.tsCalc
}

func (m *Market) State() types.Market_State {
	return m.mkt.State
}

// Return the number if liquidity provisions in the market
func (m *Market) GetLPSCount() int {
	return len(m.equityShares.lps)
}

// Get the commitment of a LP provider given their partyID
func (m *Market) GetLPCommitment(partyID string) int {
	lps := m.equityShares.lps[partyID]
	if lps != nil {
		return int(lps.stake)
	}
	return 0
}

// Get the LP Provision object for the given partyID
func (m *Market) GetLPProvision(partyID string) *types.LiquidityProvision {
	return m.liquidity.LiquidityProvisionByPartyID(partyID)
}

// Are we currently in an auction
func (m *Market) InAuction() bool {
	return m.as.InAuction()
}
