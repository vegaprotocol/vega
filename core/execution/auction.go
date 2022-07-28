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

package execution

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
)

func (m *Market) checkAuction(ctx context.Context, now time.Time) {
	// of course, if we're not in auction, there's nothing to do here
	if !m.as.InAuction() {
		return
	}

	// as soon as we have an indicative uncrossing price in opening auction it needs to be passed into the price monitoring engine so statevar calculation can start
	isOpening := m.as.IsOpeningAuction()
	if isOpening && !m.pMonitor.Initialised() {
		trades, err := m.matching.OrderBook.GetIndicativeTrades()
		if err != nil {
			m.log.Panic("Can't get indicative trades")
		}
		if len(trades) > 0 {
			// pass the first uncrossing trades to price engine so state variables depending on it can be initialised
			m.pMonitor.CheckPrice(ctx, m.as, trades, true)
			m.OnOpeningAuctionFirstUncrossingPrice()
		}
	}

	if endTS := m.as.ExpiresAt(); endTS == nil || !endTS.Before(now) {
		return
	}
	trades, err := m.matching.OrderBook.GetIndicativeTrades()
	if err != nil {
		m.log.Panic("Can't get indicative trades")
	}

	// opening auction
	if isOpening {
		if len(trades) == 0 {
			return
		}

		// first check liquidity - before we mark auction as ready to leave
		m.checkLiquidity(ctx, trades, true)
		if !m.as.CanLeave() {
			if e := m.as.AuctionExtended(ctx, now); e != nil {
				m.broker.Send(e)
			}
			return
		}
		// opening auction requirements satisfied at this point, other requirements still need to be checked downstream though
		m.as.SetReadyToLeave()
		m.pMonitor.CheckPrice(ctx, m.as, trades, true)
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
	}
	// price and liquidity auctions
	if endTS := m.as.ExpiresAt(); endTS == nil || !endTS.Before(now) {
		return
	}
	isPrice := m.as.IsPriceAuction() || m.as.IsPriceExtension()
	if !isPrice {
		m.checkLiquidity(ctx, trades, true)
	}
	if isPrice || m.as.CanLeave() {
		m.pMonitor.CheckPrice(ctx, m.as, trades, true)
	}
	end := m.as.CanLeave()
	if isPrice && end {
		m.checkLiquidity(ctx, trades, true)
	}
	if evt := m.as.AuctionExtended(ctx, m.timeService.GetTimeNow()); evt != nil {
		m.broker.Send(evt)
		end = false
	}
	// price monitoring engine and liquidity monitoring engine both indicated auction can end
	if end {
		// can we leave based on the book state?
		m.leaveAuction(ctx, now)
	}

	// This is where FBA handling will go
}
