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

package spot

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

	// will be set to non-nil if a peg is missing
	_, _, err := m.getBestStaticPricesDecimal()

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
	if err == nil {
		m.repriceAllSpecialOrders(ctx, changes, orderUpdates)
	} else {
		// we won't be able to reprice here
		m.parkAllPeggedOrders(ctx)
	}

	// Update the last price values
	// no need to clone the prices, they're not used in calculations anywhere in this function
	m.lastMidBuyPrice = newMidBuy
	m.lastMidSellPrice = newMidSell
	m.lastBestBidPrice = newBestBid
	m.lastBestAskPrice = newBestAsk
}
