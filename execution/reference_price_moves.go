package execution

import (
	"context"

	"code.vegaprotocol.io/vega/types"
)

const (
	// PriceMoveMid used to indicate that the mid price has moved
	PriceMoveMid = 1

	// PriceMoveBestBid used to indicate that the best bid price has moved
	PriceMoveBestBid = 2

	// PriceMoveBestAsk used to indicate that the best ask price has moved
	PriceMoveBestAsk = 4

	// PriceMoveAll used to indicate everything has moved
	PriceMoveAll = PriceMoveMid + PriceMoveBestBid + PriceMoveBestAsk
)

type OrderReferenceCheck types.Order

func (o OrderReferenceCheck) HasMoved(changes uint8) bool {
	return (o.PeggedOrder.Reference == types.PeggedReference_PEGGED_REFERENCE_MID &&
		changes&PriceMoveMid > 0) ||
		(o.PeggedOrder.Reference == types.PeggedReference_PEGGED_REFERENCE_BEST_BID &&
			changes&PriceMoveBestBid > 0) ||
		(o.PeggedOrder.Reference == types.PeggedReference_PEGGED_REFERENCE_BEST_ASK &&
			changes&PriceMoveBestAsk > 0)
}

func (m *Market) checkForReferenceMoves(
	ctx context.Context, orderUpdates []*types.Order, forceUpdate bool,
) {
	if m.as.InAuction() {
		return
	}

	newBestBid, _ := m.getBestStaticBidPrice()
	newBestAsk, _ := m.getBestStaticAskPrice()
	newMidBuy, _ := m.getStaticMidPrice(types.Side_SIDE_BUY)
	newMidSell, _ := m.getStaticMidPrice(types.Side_SIDE_SELL)

	// Look for a move
	var changes uint8
	if !forceUpdate {
		if newMidBuy != m.lastMidBuyPrice ||
			newMidSell != m.lastMidSellPrice {
			changes |= PriceMoveMid
		}
		if newBestBid != m.lastBestBidPrice {
			changes |= PriceMoveBestBid
		}
		if newBestAsk != m.lastBestAskPrice {
			changes |= PriceMoveBestAsk
		}
	} else {
		changes = PriceMoveAll
	}

	// now we can start all special order repricing...
	orderUpdates = m.repriceAllSpecialOrders(ctx, changes, orderUpdates)

	// 	// Update the last price values
	m.lastMidBuyPrice = newMidBuy
	m.lastMidSellPrice = newMidSell
	m.lastBestBidPrice = newBestBid
	m.lastBestAskPrice = newBestAsk

	// now we had new orderUpdates while processing those,
	// that would means someone got distressed, so some order
	// got uncrossed, so we need to check all these again.
	// we do not use the forceUpdate ffield here as it's
	// not required that prices moved though
	if len(orderUpdates) > 0 {
		m.checkForReferenceMoves(ctx, orderUpdates, false)
	}
}
