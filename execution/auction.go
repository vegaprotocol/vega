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
	// at this point, it doesn't matter what auction type we're in
	p, v, _ := m.matching.GetIndicativePriceAndVolume()
	// opening auction
	if m.as.IsOpeningAuction() {
		if wt {
			//only do this once
			if !m.sawIndicativePrice {
				//pass the uncrossing price to liquidity engine
				if err := m.pMonitor.CheckPrice(ctx, m.as, p.Clone(), v, now, true); err != nil {
					m.log.Panic("unable to run check price with price monitor",
						logging.String("market-id", m.GetID()),
						logging.Error(err))
				}
				m.OnOpeningAuctionFirstUncrossingPrice()
				m.sawIndicativePrice = true
			}

		}
		if endTS := m.as.ExpiresAt(); endTS == nil || !endTS.Before(now) {
			return
		}
		trades, err := m.matching.GetIndicativeTrades()
		if err != nil {
			m.log.Panic("Can't get indicative trades")
		}
		if len(trades) == 0 {
			return
		}
		//opening auction requirements satisfied at this point
		m.as.SetReadyToLeave()

		m.checkLiquidity(ctx, trades, true)
		if !m.as.CanLeave() {
			return
		}
		if err := m.pMonitor.CheckPrice(ctx, m.as, p.Clone(), v, true); err != nil {
			m.log.Panic("unable to run check price with price monitor",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		if m.as.ExtensionTrigger() == types.AuctionTriggerPrice {
			// this should never, ever happen
			m.log.Panic("Leaving opening auction somehow triggered price monitoring to extend the auction")
		}

		// if we don't have yet consensus for the floating point parameters, stay in the opening auction
		if !m.CanLeaveOpeningAuction() {
			m.log.Info("cannot leave opening auction - waiting for floating point to complete the first round")
			return
		}
		m.log.Info("leaving opening auction for market", logging.String("market-id", m.mkt.ID))
		m.leaveAuction(ctx, now)
		// the market is now in a ACTIVE state
		m.mkt.State = types.MarketStateActive
		// the market is now properly open, so set the timestamp to when the opening auction actually ended
		m.mkt.MarketTimestamps.Open = now.UnixNano()
		m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

		m.equityShares.OpeningAuctionEnded()
		// start the market fee window
		m.feeSplitter.TimeWindowStart(now)
		return
	} else
	// price and liquidity auctions
	{
		if endTS := m.as.ExpiresAt(); endTS == nil || !endTS.Before(now) {
			return
		}
		isPrice := m.as.IsPriceAuction()
		trades, err := m.matching.GetIndicativeTrades()
		if err != nil {
			m.log.Panic("Can't get indicative trades")
		}
		if !isPrice {
			m.checkLiquidity(ctx, trades, true)
		}
		if isPrice || m.as.CanLeave() {
			if err := m.pMonitor.CheckPrice(ctx, m.as, p.Clone(), v, true); err != nil {
				m.log.Panic("unable to run check price with price monitor",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
		}
		end := m.as.CanLeave()
		if isPrice && end {
			m.checkLiquidity(ctx, trades, true)
		}
		if evt := m.as.AuctionExtended(ctx, m.currentTime); evt != nil {
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
			m.leaveAuction(ctx, now)
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
