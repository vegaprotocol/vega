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
	"time"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

func (m *Market) checkAuction(ctx context.Context, now time.Time, idgen common.IDGenerator) {
	if !m.as.InAuction() {
		if m.as.AuctionStart() {
			m.enterAuction(ctx)
		}
		return
	}

	if m.mkt.State == types.MarketStateSuspendedViaGovernance {
		if endTS := m.as.ExpiresAt(); endTS != nil && endTS.Before(now) {
			m.as.ExtendAuctionSuspension(types.AuctionDuration{Duration: int64(m.minDuration.Seconds())})
		}
	}

	// here we are in auction, we'll want to check
	// the triggers if we are leaving
	defer func() {
		m.triggerStopOrders(ctx, idgen)
	}()

	indicativeUncrossingPrice := num.UintZero()

	checkExceeded := m.mkt.State == types.MarketStatePending
	// as soon as we have an indicative uncrossing price in opening auction it needs to be passed into the price monitoring engine so statevar calculation can start
	isOpening := m.as.IsOpeningAuction()
	if isOpening && !m.pMonitor.Initialised() {
		indicativeUncrossingPrice = m.matching.OrderBook.GetIndicativePrice()

		if !indicativeUncrossingPrice.IsZero() {
			// pass the first uncrossing price to price monitoring engine so state variables depending on it can be initialised
			m.pMonitor.ResetPriceHistory(indicativeUncrossingPrice)
			m.OnOpeningAuctionFirstUncrossingPrice()
		}
		if checkExceeded && m.as.ExceededMaxOpening(now) {
			m.closeSpotMarket(ctx)
			return
		}
	}

	if endTS := m.as.ExpiresAt(); endTS == nil || !endTS.Before(now) {
		if checkExceeded && isOpening && m.as.ExceededMaxOpening(now) {
			m.closeSpotMarket(ctx)
		}
		return
	}
	if indicativeUncrossingPrice.IsZero() {
		indicativeUncrossingPrice = m.matching.OrderBook.GetIndicativePrice()
	}

	// opening auction
	if isOpening {
		if indicativeUncrossingPrice.IsZero() {
			if checkExceeded && m.as.ExceededMaxOpening(now) {
				m.closeSpotMarket(ctx)
			}
			return
		}

		// opening auction period has expired, and we have trades, we should be ready to leave
		// other requirements still need to be checked downstream though
		m.as.SetReadyToLeave()

		// if we don't have yet consensus for the floating point parameters, stay in the opening auction
		if !m.CanLeaveOpeningAuction() {
			m.log.Info("cannot leave opening auction - waiting for floating point to complete the first round")
			return
		}
		m.log.Info("leaving opening auction for market", logging.String("market-id", m.mkt.ID))
		m.leaveAuction(ctx, now)

		m.equityShares.OpeningAuctionEnded()
		// start the market fee window
		m.feeSplitter.TimeWindowStart(now)

		// reset SLA epoch
		m.liquidity.OnEpochStart(ctx,
			m.timeService.GetTimeNow(),
			m.getCurrentMarkPrice(),
			m.midPrice(),
			m.getTargetStake(),
			m.positionFactor,
		)

		return
	}

	if m.as.Trigger() == types.AuctionTriggerLongBlock || m.as.ExtensionTrigger() == types.AuctionTriggerLongBlock ||
		m.as.Trigger() == types.AuctionTriggerAutomatedPurchase || m.as.ExtensionTrigger() == types.AuctionTriggerAutomatedPurchase {
		if endTS := m.as.ExpiresAt(); endTS != nil && endTS.Before(now) {
			m.as.SetReadyToLeave()
		}
	}

	isPrice := m.as.IsPriceAuction() || m.as.IsPriceExtension()
	if isPrice || m.as.CanLeave() {
		m.pMonitor.CheckPrice(ctx, m.as, indicativeUncrossingPrice, true, true)
	}
	end := m.as.CanLeave()
	if evt := m.as.AuctionExtended(ctx, m.timeService.GetTimeNow()); evt != nil {
		m.broker.Send(evt)
		end = false
	}
	// price monitoring engine and liquidity monitoring engine both indicated auction can end
	if end {
		// can we leave based on the book state?
		m.leaveAuction(ctx, now)
	}
}
