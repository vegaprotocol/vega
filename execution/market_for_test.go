package execution

import (
	"context"
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

// Return the state of the LP submission for the given partyID
func (m *Market) GetLPSState(partyID string) types.LiquidityProvision_Status {
	lps := m.liquidity.LiquidityProvisionByPartyID(partyID)

	if lps != nil {
		return lps.Status
	}
	return types.LiquidityProvision_STATUS_UNSPECIFIED
}

// Returns all the pegged orders for a given party
func (m *Market) GetPeggedOrders(partyID string) []*types.Order {
	orders := m.matching.GetOrdersPerParty(partyID)

	peggedOrders := []*types.Order{}
	for _, order := range orders {
		if order.PeggedOrder != nil {
			peggedOrders = append(peggedOrders, order)
		}
	}
	return peggedOrders
}

func (m *Market) GetBondAccountBalance(ctx context.Context, partyID, marketID, asset string) uint64 {
	bondAccount, err := m.collateral.GetOrCreatePartyBondAccount(ctx, partyID, marketID, asset)
	if err == nil {
		return bondAccount.Balance
	}
	return 0
}
