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

	stateChanged bool
}

func NewAuctionState(mkt *types.Market, now time.Time) *AuctionState {
	s := AuctionState{
		mode:         types.MarketTradingModeOpeningAuction,
		defMode:      types.MarketTradingModeContinuous,
		trigger:      types.AuctionTriggerOpening,
		begin:        &now,
		start:        true,
		m:            mkt,
		stateChanged: true,
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

func (a *AuctionState) StartLiquidityAuctionNoOrders(t time.Time, d *types.AuctionDuration) {
	a.startLiquidityAuction(t, d, types.AuctionTriggerUnableToDeployLPOrders)
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
	a.stateChanged = true
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
	a.stateChanged = true
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
	a.stateChanged = true
}

// ExtendAuctionPrice - call from price monitoring to extend the auction
// sets the extension trigger field accordingly.
func (a *AuctionState) ExtendAuctionPrice(delta types.AuctionDuration) {
	t := types.AuctionTriggerPrice
	a.extension = &t
	a.ExtendAuction(delta)
}

func (a *AuctionState) ExtendAuctionLiquidityNoOrders(delta types.AuctionDuration) {
	a.extendAuctionLiquidity(delta, types.AuctionTriggerUnableToDeployLPOrders)
}

func (a *AuctionState) ExtendAuctionLiquidityUnmetTarget(delta types.AuctionDuration) {
	a.extendAuctionLiquidity(delta, types.AuctionTriggerLiquidityTargetNotMet)
}

// extendAuctionLiquidity - call from liquidity monitoring to extend the auction
// sets the extension trigger field accordingly.
func (a *AuctionState) extendAuctionLiquidity(delta types.AuctionDuration, trigger types.AuctionTrigger) {
	t := trigger
	a.extension = &t
	a.extensionEventSent = false
	a.ExtendAuction(delta)
}

// ExtendAuction extends the current auction.
func (a *AuctionState) ExtendAuction(delta types.AuctionDuration) {
	a.end.Duration += delta.Duration
	a.end.Volume += delta.Volume
	a.stop = false // the auction was supposed to stop, but we've extended it
	a.stateChanged = true
}

// SetReadyToLeave is called by monitoring engines to mark if an auction period has expired.
func (a *AuctionState) SetReadyToLeave() {
	a.stop = true
	a.stateChanged = true
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

func (a AuctionState) IsLiquidityAuction() bool {
	return a.trigger == types.AuctionTriggerLiquidityTargetNotMet || a.trigger == types.AuctionTriggerUnableToDeployLPOrders
}

func (a AuctionState) IsPriceAuction() bool {
	return a.trigger == types.AuctionTriggerPrice
}

func (a AuctionState) IsLiquidityExtension() bool {
	return a.extension != nil && (*a.extension == types.AuctionTriggerLiquidityTargetNotMet || *a.extension == types.AuctionTriggerUnableToDeployLPOrders)
}

func (a AuctionState) IsPriceExtension() bool {
	return a.extension != nil && *a.extension == types.AuctionTriggerPrice
}

func (a AuctionState) IsFBA() bool {
	return a.trigger == types.AuctionTriggerBatch
}

// IsMonitorAuction - quick way to determine whether or not we're in an auction triggered by a monitoring engine.
func (a AuctionState) IsMonitorAuction() bool {
	return a.trigger == types.AuctionTriggerPrice || a.trigger == types.AuctionTriggerLiquidityTargetNotMet || a.trigger == types.AuctionTriggerUnableToDeployLPOrders
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
	a.stateChanged = true
	return events.NewAuctionEvent(ctx, a.m.ID, false, a.begin.UnixNano(), end, a.trigger, ext)
}

// AuctionStarted is called by the execution package to set flags indicating the market has started the auction.
func (a *AuctionState) AuctionStarted(ctx context.Context, now time.Time) *events.Auction {
	a.start = false
	end := int64(0)
	if a.begin == nil {
		a.begin = &now
	}
	if a.end != nil && a.end.Duration > 0 {
		end = a.begin.Add(time.Duration(a.end.Duration) * time.Second).UnixNano()
	}
	a.stateChanged = true
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
	a.stateChanged = true
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
			a.stateChanged = true
			a.end.Duration = int64(d / time.Second)
			// this would increase the duration by delta new - old, effectively setting duration == new min. Instead, we can just assign new min duraiton.
			// a.end.Duration += int64(newMin.Sub(*oldExp) / time.Second) // we have to divide by seconds as we're using seconds in AuctionDuration type
			return events.NewAuctionEvent(ctx, a.m.ID, false, a.begin.UnixNano(), newMin.UnixNano(), a.trigger)
		}
	}
	return nil
}
