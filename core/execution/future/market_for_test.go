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
	"time"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

// UpdateRiskFactorsForTest is a hack for setting the risk factors for tests directly rather than through the consensus engine.
// Never use this for anything functional.
func (m *Market) UpdateRiskFactorsForTest() {
	m.risk.CalculateRiskFactorsForTest()
}

func (m *Market) EnterAuction(ctx context.Context) {
	m.enterAuction(ctx)
}

func (m *Market) LeaveAuctionWithIDGen(ctx context.Context, now time.Time, generator common.IDGenerator) {
	m.idgen = generator
	defer func() { m.idgen = nil }()
	m.leaveAuction(ctx, now)
}

// GetPeggedOrderCount returns the number of pegged orders in the market.
func (m *Market) GetPeggedOrderCount() int {
	return len(m.matching.GetActivePeggedOrderIDs()) + len(m.peggedOrders.GetParkedIDs())
}

// GetParkedOrderCount returns hte number of parked orders in the market.
func (m *Market) GetParkedOrderCount() int {
	return len(m.peggedOrders.Parked())
}

// GetPeggedExpiryOrderCount returns the number of pegged order that can expire.
func (m *Market) GetPeggedExpiryOrderCount() int {
	return m.expiringOrders.GetExpiryingOrderCount()
}

// GetOrdersOnBookCount returns the number of orders on the live book.
func (m *Market) GetOrdersOnBookCount() int64 {
	return m.matching.GetTotalNumberOfOrders()
}

// GetVolumeOnBook returns the volume of orders on one side of the book.
func (m *Market) GetVolumeOnBook() int64 {
	return m.matching.GetTotalVolume()
}

// StartPriceAuction initialises the market to handle a price auction.
func (m *Market) StartPriceAuction(now time.Time) {
	end := types.AuctionDuration{
		Duration: 1000,
	}
	// setup auction
	m.as.StartPriceAuction(now, &end)
}

// TSCalc returns the local tsCalc instance.
func (m *Market) TSCalc() TargetStakeCalculator {
	return m.tsCalc
}

func (m *Market) State() types.MarketState {
	return m.mkt.State
}

// Return the number if liquidity provisions in the market.
func (m *Market) GetLPSCount() int {
	return len(m.equityShares.lps)
}

// Return the state of the LP submission for the given partyID.
func (m *Market) GetLPSState(partyID string) types.LiquidityProvisionStatus {
	lps := m.liquidity.LiquidityProvisionByPartyID(partyID)

	if lps != nil {
		return lps.Status
	}
	return types.LiquidityProvisionUnspecified
}

// Returns all the pegged orders for a given party.
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

// Returns the amount of assets in the bond account (no need to clone in these functions).
func (m *Market) GetBondAccountBalance(ctx context.Context, partyID, marketID, asset string) *num.Uint {
	bondAccount, err := m.collateral.GetOrCreatePartyBondAccount(ctx, partyID, marketID, asset)
	if err == nil {
		return bondAccount.Balance
	}
	return num.UintZero()
}

// Returns the amount of assets in the general account.
func (m *Market) GetGeneralAccountBalance(partyID, asset string) *num.Uint {
	generalAccount, err := m.collateral.GetPartyGeneralAccount(partyID, asset)
	if err == nil {
		return generalAccount.Balance
	}
	return num.UintZero()
}

// Returns the amount of assets in the margin account.
func (m *Market) GetMarginAccountBalance(partyID, marketID, asset string) *num.Uint {
	marginAccount, err := m.collateral.GetPartyMarginAccount(marketID, partyID, asset)
	if err == nil {
		return marginAccount.Balance
	}
	return num.UintZero()
}

// Get the total assets for a party.
func (m *Market) GetTotalAccountBalance(ctx context.Context, partyID, marketID, asset string) *num.Uint {
	return num.Sum(
		m.GetGeneralAccountBalance(partyID, asset),
		m.GetMarginAccountBalance(partyID, marketID, asset),
		m.GetBondAccountBalance(ctx, partyID, marketID, asset),
	)
}

// Return the current liquidity fee value for a market.
func (m *Market) GetLiquidityFee() num.Decimal {
	return m.fee.GetLiquidityFee()
}

// Log out orders that don't match.
func (m *Market) ValidateOrder(order *types.Order) bool {
	order2, err := m.matching.GetOrderByID(order.ID)
	if err != nil {
		return false
	}
	return (order.Price.EQ(order2.Price) && order.Size == order2.Size &&
		order.Remaining == order2.Remaining && order.Status == order2.Status)
}

func (m *Market) DumpBook() {
	m.matching.PrintState("test")
}
