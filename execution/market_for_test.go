package execution

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/events"
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

// GetVolumeOnBook returns the volume of orders on one side of the book
func (m *Market) GetVolumeOnBook() int64 {
	return m.matching.GetTotalVolume()
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

// Returns the amount of assets in the bond account
func (m *Market) GetBondAccountBalance(ctx context.Context, partyID, marketID, asset string) uint64 {
	bondAccount, err := m.collateral.GetOrCreatePartyBondAccount(ctx, partyID, marketID, asset)
	if err == nil {
		return bondAccount.Balance
	}
	return 0
}

// Returns the amount of assets in the general account
func (m *Market) GetGeneralAccountBalance(partyID, asset string) uint64 {
	generalAccount, err := m.collateral.GetPartyGeneralAccount(partyID, asset)
	if err == nil {
		return generalAccount.Balance
	}
	return 0
}

// Returns the amount of assets in the margin account
func (m *Market) GetMarginAccountBalance(partyID, marketID, asset string) uint64 {
	marginAccount, err := m.collateral.GetPartyMarginAccount(marketID, partyID, asset)
	if err == nil {
		return marginAccount.Balance
	}
	return 0
}

// Get the total assets for a party
func (m *Market) GetTotalAccountBalance(ctx context.Context, partyID, marketID, asset string) uint64 {
	return m.GetGeneralAccountBalance(partyID, asset) +
		m.GetMarginAccountBalance(partyID, marketID, asset) +
		m.GetBondAccountBalance(ctx, partyID, marketID, asset)
}

// Return the current liquidity fee value for a market
func (m *Market) GetLiquidityFee() float64 {
	return m.fee.GetLiquidityFee()
}

// Log out orders that don't match
func (m *Market) ValidateOrder(order *types.Order) bool {
	order2, err := m.matching.GetOrderByID(order.Id)
	if err != nil {
		return false
	}
	if order.Price != order2.Price ||
		order.Size != order2.Size ||
		order.Remaining != order2.Remaining ||
		order.Status != order2.Status {
		fmt.Println("Orders do not match")
		fmt.Println("OrderBook  :", order2)
		fmt.Println("MarketDepth:", order)
		return false
	}
	return true
}

func (m *Market) SendEvents(ctx context.Context, orderCount int) {
	order := &types.Order{
		Price:     1111,
		Size:      2222,
		Remaining: 3333,
		Id:        "wiqoweqoiwopqiweuoiwe",
	}

	for i := 0; i < orderCount; i++ {
		order.UpdatedAt = int64(i)
		m.broker.Send(events.NewOrderEvent(ctx, order))
	}
}
