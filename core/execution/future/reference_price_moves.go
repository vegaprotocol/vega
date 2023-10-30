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

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/types"
)

func (m *Market) checkForReferenceMoves(
	ctx context.Context, orderUpdates []*types.Order, forceUpdate bool,
) {
	if m.as.InAuction() {
		return
	}

	newBestBid, _ := m.getBestStaticBidPrice()
	newBestAsk, _ := m.getBestStaticAskPrice()
	newMidBuy, _ := m.getStaticMidPrice(types.SideBuy)
	newMidSell, _ := m.getStaticMidPrice(types.SideSell)

	// Look for a move
	var changes uint8
	if !forceUpdate {
		if newMidBuy.NEQ(m.lastMidBuyPrice) || newMidSell.NEQ(m.lastMidSellPrice) {
			changes |= common.PriceMoveMid
		}
		if newBestBid.NEQ(m.lastBestBidPrice) {
			changes |= common.PriceMoveBestBid
		}
		if newBestAsk.NEQ(m.lastBestAskPrice) {
			changes |= common.PriceMoveBestAsk
		}
	} else {
		changes = common.PriceMoveAll
	}

	// now we can start all special order repricing...
	orderUpdates = m.repriceAllSpecialOrders(ctx, changes, orderUpdates)

	// Update the last price values
	// no need to clone the prices, they're not used in calculations anywhere in this function
	m.lastMidBuyPrice = newMidBuy
	m.lastMidSellPrice = newMidSell
	m.lastBestBidPrice = newBestBid
	m.lastBestAskPrice = newBestAsk

	// now we had new orderUpdates while processing those,
	// that would means someone got distressed, so some order
	// got uncrossed, so we need to check all these again.
	// we do not use the forceUpdate field here as it's
	// not required that prices moved though
	if len(orderUpdates) > 0 {
		m.checkForReferenceMoves(ctx, orderUpdates, false)
	}
}
