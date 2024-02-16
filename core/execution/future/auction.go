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
	"time"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
)

func (m *Market) checkAuction(ctx context.Context, now time.Time, idgen common.IDGenerator) {
	if !m.as.InAuction() {
		// new block, check bond balance, top up if needed
		m.checkBondBalance(ctx)
		return
	}

	if m.mkt.State == types.MarketStateSuspendedViaGovernance {
		if endTS := m.as.ExpiresAt(); endTS != nil && endTS.Before(now) {
			m.as.ExtendAuctionSuspension(types.AuctionDuration{Duration: int64(m.minDuration)})
		}
	}

	// here we are in auction, we'll want to check
	// the triggers if we are leaving
	defer func() {
		m.triggerStopOrders(ctx, idgen)
	}()
	var (
		trades []*types.Trade
		err    error
	)

	checkExceeded := m.mkt.State == types.MarketStatePending

	// as soon as we have an indicative uncrossing price in opening auction it needs to be passed into the price monitoring engine so statevar calculation can start
	isOpening := m.as.IsOpeningAuction()
	if isOpening && !m.pMonitor.Initialised() {
		if trades, err = m.matching.GetIndicativeTrades(); err != nil {
			m.log.Panic("Can't get indicative trades")
		}
		if len(trades) > 0 {
			// pass the first uncrossing trades to price engine so state variables depending on it can be initialised
			m.pMonitor.CheckPrice(ctx, m.as, trades, true)
			m.OnOpeningAuctionFirstUncrossingPrice()
		}
	}

	if endTS := m.as.ExpiresAt(); endTS == nil || !endTS.Before(now) {
		if isOpening && checkExceeded && m.as.ExceededMaxOpening(now) {
			// cancel the market, exceeded opening auction
			m.log.Debug("Market was cancelled because it failed to leave opening auction in time", logging.MarketID(m.GetID()))
			m.terminateMarket(ctx, types.MarketStateCancelled, nil)
		}
		return
	}
	if len(trades) == 0 {
		if trades, err = m.matching.GetIndicativeTrades(); err != nil {
			m.log.Panic("Can't get indicative trades")
		}
	}

	// opening auction
	if isOpening {
		if len(trades) == 0 {
			if checkExceeded && m.as.ExceededMaxOpening(now) {
				m.log.Debug("Market was cancelled because it failed to leave opening auction in time", logging.MarketID(m.GetID()))
				m.terminateMarket(ctx, types.MarketStateCancelled, nil)
			}
			return
		}

		// first check liquidity - before we mark auction as ready to leave
		m.checkBondBalance(ctx)
		if checkExceeded && m.as.ExceededMaxOpening(now) {
			// cancel the market, exceeded opening auction
			m.log.Debug("Market was cancelled because it failed to leave opening auction in time", logging.MarketID(m.GetID()))
			m.terminateMarket(ctx, types.MarketStateCancelled, nil)
			return
		}
		if e := m.as.AuctionExtended(ctx, now); e != nil {
			m.broker.Send(e)
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
	// NOTE: This is a fix for the snapshot restores in case we're restoring a liquidity auction
	// from a snapshot (or protocol upgrade) with state form before the liquidity monitoring was still
	// in place. This can be removed once we've deployed the version without liquidity monitoring.
	// Liquidity auctions are no longer a thing, we know we're not in opening auction here
	// if we're not in price auction, we should just let the liquidity auction expire
	if m.as.Trigger() == types.AuctionTriggerLiquidityTargetNotMet || m.as.Trigger() == types.AuctionTriggerUnableToDeployLPOrders {
		m.as.SetReadyToLeave()
	}
	// price and liquidity auctions
	isPrice := m.as.IsPriceAuction() || m.as.IsPriceExtension()
	if !isPrice {
		m.checkBondBalance(ctx)
	}
	if isPrice || m.as.CanLeave() {
		m.pMonitor.CheckPrice(ctx, m.as, trades, true)
	}
	end := m.as.CanLeave()
	if isPrice && end {
		m.checkBondBalance(ctx)
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
