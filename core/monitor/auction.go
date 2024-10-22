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

package monitor

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
)

type AuctionState struct {
	mode               types.MarketTradingMode // current trading mode
	defMode            types.MarketTradingMode // default trading mode for market
	trigger            types.AuctionTrigger    // Set to the value indicating what started the auction
	begin              *time.Time              // optional setting auction start time (will be set if start flag is true)
	end                *types.AuctionDuration  // will be set when in auction, defines parameters that end an auction period
	start, stop        bool                    // flags to clarify whether we're entering or leaving auction
	m                  *types.Market           // keep market definition handy, useful to end auctions when default is FBA
	extension          *types.AuctionTrigger   // Set if the current auction was extended, reset after the event was created
	extensionEventSent bool
	maxDuration        *time.Duration
}

func NewAuctionState(mkt *types.Market, now time.Time) *AuctionState {
	s := AuctionState{
		mode:    types.MarketTradingModeOpeningAuction,
		defMode: types.MarketTradingModeContinuous,
		trigger: types.AuctionTriggerOpening,
		begin:   &now,
		start:   true,
		m:       mkt,
	}
	// no opening auction
	if mkt.OpeningAuction == nil {
		s.mode = s.defMode
		if s.mode == types.MarketTradingModeBatchAuction {
			// @TODO set end params here (FBA is not yet implemented)
			return &s
		}
		// no opening auction
		s.begin = nil
		s.start = false
		s.trigger = types.AuctionTriggerUnspecified
	} else {
		s.end = mkt.OpeningAuction.DeepClone()
	}
	return &s
}

func (a *AuctionState) GetAuctionBegin() *time.Time {
	if a.begin == nil {
		return nil
	}
	cpy := *a.begin
	return &cpy
}

func (a *AuctionState) GetAuctionEnd() *time.Time {
	if a.begin == nil || a.end == nil {
		return nil
	}
	cpy := *a.begin
	cpy = cpy.Add(time.Duration(a.end.Duration) * time.Second)
	return &cpy
}

func (a *AuctionState) StartLiquidityAuctionUnmetTarget(t time.Time, d *types.AuctionDuration) {
	a.startLiquidityAuction(t, d, types.AuctionTriggerLiquidityTargetNotMet)
}

// startLiquidityAuction - set the state to start a liquidity triggered auction
// @TODO these functions will be removed once the types are in proto.
func (a *AuctionState) startLiquidityAuction(t time.Time, d *types.AuctionDuration, tigger types.AuctionTrigger) {
	a.mode = types.MarketTradingModeMonitoringAuction
	a.trigger = tigger
	a.start = true
	a.stop = false
	a.begin = &t
	a.end = d
}

// StartPriceAuction - set the state to start a price triggered auction
// @TODO these functions will be removed once the types are in proto.
func (a *AuctionState) StartPriceAuction(t time.Time, d *types.AuctionDuration) {
	a.mode = types.MarketTradingModeMonitoringAuction
	a.trigger = types.AuctionTriggerPrice
	a.start = true
	a.stop = false
	a.begin = &t
	a.end = d
}

func (a *AuctionState) StartLongBlockAuction(t time.Time, d int64) {
	a.mode = types.MarketTradingModeLongBlockAuction
	a.trigger = types.AuctionTriggerLongBlock
	a.start = true
	a.stop = false
	a.begin = &t
	a.end = &types.AuctionDuration{Duration: d}
}

func (a *AuctionState) StartAutomatedPurchaseAuction(t time.Time, d int64) {
	a.mode = types.MarketTradingModeAutomatedPuchaseAuction
	a.trigger = types.AuctionTriggerAutomatedPurchase
	a.start = true
	a.stop = false
	a.begin = &t
	a.end = &types.AuctionDuration{Duration: d}
}

func (a *AuctionState) StartGovernanceSuspensionAuction(t time.Time) {
	a.mode = types.MarketTradingModeSuspendedViaGovernance
	a.trigger = types.AuctionTriggerGovernanceSuspension
	a.start = true
	a.stop = false
	a.begin = &t
	a.end = &types.AuctionDuration{Duration: 0}
}

func (a *AuctionState) EndGovernanceSuspensionAuction() {
	if a.trigger == types.AuctionTriggerGovernanceSuspension {
		// if there governance was the trigger and there is no extension, reset the state.
		if a.extension == nil {
			a.mode = types.MarketTradingModeContinuous
			a.trigger = types.AuctionTriggerUnspecified
			a.start = false
			a.stop = true
			a.begin = nil
			a.end = nil
		} else {
			// if we're leaving the governance auction which was the trigger but there was an extension trigger -
			// make the extension trigger the trigger and set the mode to monitoring auction.
			a.mode = types.MarketTradingModeMonitoringAuction
			a.trigger = *a.extension
			a.extension = nil
		}
	} else if a.ExtensionTrigger() == types.AuctionTriggerGovernanceSuspension {
		// if governance suspension was the extension trigger - just reset it.
		a.extension = nil
	}
}

// StartOpeningAuction - set the state to start an opening auction (used for testing)
// @TODO these functions will be removed once the types are in proto.
func (a *AuctionState) StartOpeningAuction(t time.Time, d *types.AuctionDuration) {
	a.mode = types.MarketTradingModeOpeningAuction
	a.trigger = types.AuctionTriggerOpening
	a.start = true
	a.stop = false
	a.begin = &t
	a.end = d
}

// ExtendAuctionPrice - call from price monitoring to extend the auction
// sets the extension trigger field accordingly.
func (a *AuctionState) ExtendAuctionPrice(delta types.AuctionDuration) {
	t := types.AuctionTriggerPrice
	a.extension = &t
	a.ExtendAuction(delta)
}

func (a *AuctionState) ExtendAuctionLongBlock(delta types.AuctionDuration) {
	t := types.AuctionTriggerLongBlock
	if a.trigger != t {
		a.extension = &t
	}
	a.ExtendAuction(delta)
}

func (a *AuctionState) ExtendAuctionAutomatedPurchase(delta types.AuctionDuration) {
	t := types.AuctionTriggerAutomatedPurchase
	if a.trigger != t {
		a.extension = &t
	}
	a.ExtendAuction(delta)
}

func (a *AuctionState) ExtendAuctionSuspension(delta types.AuctionDuration) {
	t := types.AuctionTriggerGovernanceSuspension
	a.extension = &t
	a.ExtendAuction(delta)
}

// ExtendAuction extends the current auction.
func (a *AuctionState) ExtendAuction(delta types.AuctionDuration) {
	a.end.Duration += delta.Duration
	a.end.Volume += delta.Volume
	a.stop = false // the auction was supposed to stop, but we've extended it
}

// SetReadyToLeave is called by monitoring engines to mark if an auction period has expired.
func (a *AuctionState) SetReadyToLeave() {
	// we can't leave the auction if it was triggered by governance suspension
	if a.trigger == types.AuctionTriggerGovernanceSuspension {
		return
	}
	if a.maxDuration != nil {
		a.maxDuration = nil
	}
	a.stop = true
}

// Duration returns a copy of the current auction duration object.
func (a AuctionState) Duration() types.AuctionDuration {
	if a.end == nil {
		return types.AuctionDuration{}
	}
	return *a.end
}

// Start - returns time pointer of the start of the auction (nil if not in auction).
func (a AuctionState) Start() time.Time {
	if a.begin == nil {
		return time.Time{} // zero time
	}
	return *a.begin
}

// ExpiresAt returns end as time -> if nil, the auction duration either isn't determined by time
// or we're simply not in an auction.
func (a AuctionState) ExpiresAt() *time.Time {
	if a.begin == nil { // no start time == no end time
		return nil
	}
	if a.end == nil || a.end.Duration == 0 { // not time limited
		return nil
	}
	// add duration to start time, return
	t := a.begin.Add(time.Duration(a.end.Duration) * time.Second)
	return &t
}

// Mode returns current trading mode.
func (a AuctionState) Mode() types.MarketTradingMode {
	return a.mode
}

// Trigger returns what triggered an auction.
func (a AuctionState) Trigger() types.AuctionTrigger {
	return a.trigger
}

// ExtensionTrigger returns what extended an auction.
func (a AuctionState) ExtensionTrigger() types.AuctionTrigger {
	if a.extension == nil {
		return types.AuctionTriggerUnspecified
	}
	return *a.extension
}

// InAuction returns bool if the market is in auction for any reason
// Returns false if auction is triggered, but not yet started by market (execution).
func (a AuctionState) InAuction() bool {
	return !a.start && a.trigger != types.AuctionTriggerUnspecified
}

func (a AuctionState) IsOpeningAuction() bool {
	return a.trigger == types.AuctionTriggerOpening
}

func (a AuctionState) IsPriceAuction() bool {
	return a.trigger == types.AuctionTriggerPrice
}

func (a AuctionState) IsPriceExtension() bool {
	return a.extension != nil && *a.extension == types.AuctionTriggerPrice
}

func (a AuctionState) IsFBA() bool {
	return a.trigger == types.AuctionTriggerBatch
}

// IsMonitorAuction - quick way to determine whether or not we're in an auction triggered by a monitoring engine.
func (a AuctionState) IsMonitorAuction() bool {
	// FIXME(jeremy): the second part of the condition is to support
	// the compatibility on 72 > 73 snapshots.

	return a.trigger == types.AuctionTriggerPrice || a.trigger == types.AuctionTriggerLiquidityTargetNotMet || a.trigger == types.AuctionTriggerUnableToDeployLPOrders
}

func (a AuctionState) IsPAPAuction() bool {
	return a.trigger == types.AuctionTriggerAutomatedPurchase
}

// CanLeave bool indicating whether auction should be closed or not, if true, we can still extend the auction
// but when the market takes over (after monitoring engines), the auction will be closed.
func (a AuctionState) CanLeave() bool {
	return a.stop
}

// AuctionStart bool indicates something has already triggered an auction to start, we can skip other monitoring potentially
// and we know to create an auction event.
func (a AuctionState) AuctionStart() bool {
	return a.start
}

// AuctionExtended - called to confirm we will not leave auction, returns the event to be sent
// or nil if the auction wasn't extended.
func (a *AuctionState) AuctionExtended(ctx context.Context, now time.Time) *events.Auction {
	if a.extension == nil || a.extensionEventSent {
		return nil
	}
	a.start = false
	end := int64(0)
	if a.begin == nil {
		a.begin = &now
	}
	if a.end != nil && a.end.Duration > 0 {
		end = a.begin.Add(time.Duration(a.end.Duration) * time.Second).UnixNano()
	}
	ext := *a.extension
	// set extension flag to nil
	a.extensionEventSent = true
	return events.NewAuctionEvent(ctx, a.m.ID, false, a.begin.UnixNano(), end, a.trigger, ext)
}

// AuctionStarted is called by the execution package to set flags indicating the market has started the auction.
func (a *AuctionState) AuctionStarted(ctx context.Context, now time.Time) *events.Auction {
	a.start = false
	end := int64(0)
	// Either an auction was just started, or a market in opening auction passed the vote, the real opening auction starts now.
	if a.begin == nil || (a.trigger == types.AuctionTriggerOpening && a.begin.Before(now)) {
		a.begin = &now
	}
	if a.end != nil && a.end.Duration > 0 {
		end = a.begin.Add(time.Duration(a.end.Duration) * time.Second).UnixNano()
	}
	return events.NewAuctionEvent(ctx, a.m.ID, false, a.begin.UnixNano(), end, a.trigger)
}

// Left is called by execution to update internal state indicating this auction was closed.
func (a *AuctionState) Left(ctx context.Context, now time.Time) *events.Auction {
	// the end-of-auction event
	var start int64
	if a.begin != nil {
		start = a.begin.UnixNano()
	}
	evt := events.NewAuctionEvent(ctx, a.m.ID, true, start, now.UnixNano(), a.trigger)
	a.start, a.stop = false, false
	a.begin, a.end = nil, nil
	a.trigger = types.AuctionTriggerUnspecified
	a.extension = nil
	a.mode = a.defMode
	// default mode is auction, this is an FBA market
	if a.mode == types.MarketTradingModeBatchAuction {
		a.trigger = types.AuctionTriggerBatch
	}
	return evt
}

// UpdateMinDuration - see if we need to update the end value for current auction duration (if any)
// if the auction duration increases, an auction event will be returned.
func (a *AuctionState) UpdateMinDuration(ctx context.Context, d time.Duration) *events.Auction {
	// oldExp is nil if we're not in auction
	if oldExp := a.ExpiresAt(); oldExp != nil {
		// calc new end for auction:
		newMin := a.begin.Add(d)
		// no need to check for nil, we already have
		if newMin.After(*oldExp) {
			a.end.Duration = int64(d / time.Second)
			// this would increase the duration by delta new - old, effectively setting duration == new min. Instead, we can just assign new min duration.
			// a.end.Duration += int64(newMin.Sub(*oldExp) / time.Second) // we have to divide by seconds as we're using seconds in AuctionDuration type
			return events.NewAuctionEvent(ctx, a.m.ID, false, a.begin.UnixNano(), newMin.UnixNano(), a.trigger)
		}
	}
	return nil
}

func (a *AuctionState) UpdateMaxDuration(_ context.Context, d time.Duration) {
	if a.trigger == types.AuctionTriggerOpening {
		a.maxDuration = &d
	}
}

func (a *AuctionState) ExceededMaxOpening(now time.Time) bool {
	if a.trigger != types.AuctionTriggerOpening || a.begin == nil || a.maxDuration == nil {
		return false
	}
	minTo := now
	if a.end != nil && a.end.Duration > 0 {
		minTo = a.begin.Add(time.Duration(a.end.Duration) * time.Second)
	}
	validTo := a.begin.Add(*a.maxDuration)
	// the market is invalid if it hasn't left auction before max duration
	// or if it cannot leave before max duration allows it
	return validTo.Before(now) || minTo.After(validTo)
}
