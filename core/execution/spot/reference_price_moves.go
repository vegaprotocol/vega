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

package spot

import (
	"context"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/types"
)

func (m *Market) checkForReferenceMoves(ctx context.Context, forceUpdate bool) {
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

	m.repriceAllSpecialOrders(ctx, changes)

	// Update the last price values
	// no need to clone the prices, they're not used in calculations anywhere in this function
	m.lastMidBuyPrice = newMidBuy
	m.lastMidSellPrice = newMidSell
	m.lastBestBidPrice = newBestBid
	m.lastBestAskPrice = newBestAsk
}
