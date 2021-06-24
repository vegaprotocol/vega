package execution

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
)

func (m *Market) checkAuction(ctx context.Context, now time.Time) {
	// of course, if we're not in auction, there's nothing to do here
	if !m.as.InAuction() {
		return
	}
	wt, nt := m.matching.CanLeaveAuction()
	if !wt && m.as.IsOpeningAuction() {
		// we won't be able to leave opening auction anyway
		// in case the opening auction has expired, we might want to extend by 1s
		// but current behaviour is to leave the auction ASAP
		// m.extendAuctionIncompleteBook()
		return
	}
	// at this point, it doesn't matter what auction type we're in
	p, v, _ := m.matching.GetIndicativePriceAndVolume()
	// opening auction
	if m.as.IsOpeningAuction() {
		if endTS := m.as.ExpiresAt(); endTS == nil || !endTS.Before(now) {
			return
		}
		if err := m.pMonitor.CheckPrice(ctx, m.as, p.Clone(), v, now, true); err != nil {
			m.log.Panic("unable to run check price with price monitor",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		if evt := m.as.AuctionExtended(ctx); evt != nil {
			// this should never, ever happen
			m.log.Panic("Leaving opening auction somehow triggered price monitoring to extend the auction")
		}
		m.as.SetReadyToLeave()
		m.LeaveAuction(ctx, now)
		// the market is now in a ACTIVE state
		m.mkt.State = types.Market_STATE_ACTIVE
		m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

		m.equityShares.OpeningAuctionEnded()
		// start the market fee window
		m.feeSplitter.TimeWindowStart(now)
		return
	}
	// price and liquidity auctions
	if isPrice := m.as.IsPriceAuction(); isPrice || m.as.IsLiquidityAuction() {
		// hacky way to ensure the liquidity monitoring will calculate the target stake based on the target stake
		// SHOULD we leave the auction. Otherwise, we would leave a liquidity auction, and immediately enter a new one
		ft := []*types.Trade{
			{
				Size:  v,
				Price: p.Clone(),
			},
		}
		if !isPrice {
			m.checkLiquidity(ctx, ft)
		}
		if isPrice || m.as.CanLeave() {
			if err := m.pMonitor.CheckPrice(ctx, m.as, p.Clone(), v, now, true); err != nil {
				m.log.Panic("unable to run check price with price monitor",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
		}
		end := m.as.CanLeave()
		if isPrice && end {
			m.checkLiquidity(ctx, ft)
		}
		if evt := m.as.AuctionExtended(ctx); evt != nil {
			m.broker.Send(evt)
			end = false
		}
		// price monitoring engine and liquidity monitoring engine both indicated auction can end
		if end {
			// can we leave based on the book state?
			if !nt {
				m.extendAuctionIncompleteBook()
				return
			}
			m.LeaveAuction(ctx, now)
		}
	}
	// This is where FBA handling will go
}

func (m *Market) extendAuctionIncompleteBook() {
	if m.as.IsOpeningAuction() {
		// extend 1 second
		m.as.ExtendAuction(types.AuctionDuration{
			Duration: 1,
		})
		return
	}
	if m.as.IsPriceAuction() {
		m.as.ExtendAuctionPrice(types.AuctionDuration{
			Duration: 1,
		})
		return
	}
	m.as.ExtendAuctionLiquidity(types.AuctionDuration{
		Duration: 1,
	})
}
