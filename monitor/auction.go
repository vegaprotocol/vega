package monitor

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

type AuctionState struct {
	mode        types.Market_TradingMode // current trading mode
	defMode     types.Market_TradingMode // default trading mode for market
	trigger     types.AuctionTrigger     // Set to the value indicating what started the auction
	begin       *time.Time               // optional setting auction start time (will be set if start flag is true)
	end         *types.AuctionDuration   // will be set when in auction, defines parameters that end an auction period
	start, stop bool                     // flags to clarify whether we're entering or leaving auction
	m           *types.Market            // keep market definition handy, useful to end auctions when default is FBA
	extension   *types.AuctionTrigger    // Set if the current auction was extended, reset after the event was created
	// openingAuctionTimer tracks the elapsed time spend in opening auction.
	openingAuctionTimer *metrics.TimeCounter
}

func NewAuctionState(mkt *types.Market, now time.Time) *AuctionState {
	s := AuctionState{
		mode:    types.Market_TRADING_MODE_OPENING_AUCTION,
		defMode: types.Market_TRADING_MODE_CONTINUOUS,
		trigger: types.AuctionTrigger_AUCTION_TRIGGER_OPENING,
		begin:   &now,
		end:     mkt.OpeningAuction,
		start:   true,
		m:       mkt,
	}
	if mkt.GetContinuous() == nil {
		s.defMode = types.Market_TRADING_MODE_BATCH_AUCTION
	}
	// no opening auction
	if mkt.OpeningAuction == nil {
		s.mode = s.defMode
		if s.mode == types.Market_TRADING_MODE_BATCH_AUCTION {
			// @TODO set end params here (FBA is not yet implemented)
			return &s
		}
		// no opening auction
		s.begin = nil
		s.start = false
		s.trigger = types.AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED
	}
	return &s
}

// StartLiquidityAuction - set the state to start a liquidity triggered auction
// @TODO these functions will be removed once the types are in proto
func (a *AuctionState) StartLiquidityAuction(t time.Time, d *types.AuctionDuration) {
	a.mode = types.Market_TRADING_MODE_MONITORING_AUCTION
	a.trigger = types.AuctionTrigger_AUCTION_TRIGGER_LIQUIDITY
	a.start = true
	a.stop = false
	a.begin = &t
	a.end = d
}

// StartPriceAuction - set the state to start a price triggered auction
// @TODO these functions will be removed once the types are in proto
func (a *AuctionState) StartPriceAuction(t time.Time, d *types.AuctionDuration) {
	a.mode = types.Market_TRADING_MODE_MONITORING_AUCTION
	a.trigger = types.AuctionTrigger_AUCTION_TRIGGER_PRICE
	a.start = true
	a.stop = false
	a.begin = &t
	a.end = d
}

// StartOpeningAuction - set the state to start an opening auction (used for testing)
// @TODO these functions will be removed once the types are in proto
func (a *AuctionState) StartOpeningAuction(t time.Time, d *types.AuctionDuration) {
	a.mode = types.Market_TRADING_MODE_OPENING_AUCTION
	a.trigger = types.AuctionTrigger_AUCTION_TRIGGER_OPENING
	a.start = true
	a.stop = false
	a.begin = &t
	a.end = d
}

// ExtendAuctionPrice - call from price monitoring to extend the auction
// sets the extension trigger field accordingly
func (a *AuctionState) ExtendAuctionPrice(delta types.AuctionDuration) {
	t := types.AuctionTrigger_AUCTION_TRIGGER_PRICE
	a.extension = &t
	a.ExtendAuction(delta)
}

// ExtendAuctionLiquidity - call from liquidity monitoring to extend the auction
// sets the extension trigger field accordingly
func (a *AuctionState) ExtendAuctionLiquidity(delta types.AuctionDuration) {
	t := types.AuctionTrigger_AUCTION_TRIGGER_LIQUIDITY
	a.extension = &t
	a.ExtendAuction(delta)
}

// ExtendAuction extends the current auction, leaving trigger etc... in tact
// @TODO private this function once we´ve ensured it´s no longer used elsewhere
func (a *AuctionState) ExtendAuction(delta types.AuctionDuration) {
	a.end.Duration += delta.Duration
	a.end.Volume += delta.Volume
	a.stop = false // the auction was supposed to stop, but we've extended it
}

// EndAuction is called by monitoring engines to mark if an auction period has expired
func (a *AuctionState) EndAuction() {
	a.stop = true
}

// Duration returns a copy of the current auction duration object
func (a AuctionState) Duration() types.AuctionDuration {
	if a.end == nil {
		return types.AuctionDuration{}
	}
	return *a.end
}

// Start - returns time pointer of the start of the auction (nil if not in auction)
func (a AuctionState) Start() time.Time {
	if a.begin == nil {
		return time.Time{} // zero time
	}
	return *a.begin
}

// ExpiresAt returns end as time -> if nil, the auction duration either isn't determined by time
// or we're simply not in an auction
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

// Mode returns current trading mode
func (a AuctionState) Mode() types.Market_TradingMode {
	return a.mode
}

// Trigger returns what triggered an auction
func (a AuctionState) Trigger() types.AuctionTrigger {
	return a.trigger
}

// InAuction returns bool if the market is in auction for any reason
// Returns false if auction is triggered, but not yet started by market (execution)
func (a AuctionState) InAuction() bool {
	return !a.start && a.trigger != types.AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED
}

func (a AuctionState) IsOpeningAuction() bool {
	return a.trigger == types.AuctionTrigger_AUCTION_TRIGGER_OPENING
}

func (a AuctionState) IsLiquidityAuction() bool {
	return a.trigger == types.AuctionTrigger_AUCTION_TRIGGER_LIQUIDITY
}

func (a AuctionState) IsPriceAuction() bool {
	return a.trigger == types.AuctionTrigger_AUCTION_TRIGGER_PRICE
}

func (a AuctionState) IsFBA() bool {
	return a.trigger == types.AuctionTrigger_AUCTION_TRIGGER_BATCH
}

// IsMonitorAuction - quick way to determine whether or not we're in an auction triggered by a monitoring engine
func (a AuctionState) IsMonitorAuction() bool {
	return a.trigger == types.AuctionTrigger_AUCTION_TRIGGER_PRICE || a.trigger == types.AuctionTrigger_AUCTION_TRIGGER_LIQUIDITY
}

// AuctionEnd bool indicating whether auction should be closed or not, if true, we can still extend the auction
// but when the market takes over (after monitoring engines), the auction will be closed
func (a AuctionState) AuctionEnd() bool {
	return a.stop
}

// AuctionStart bool indicates something has already triggered an auction to start, we can skip other monitoring potentially
// and we know to create an auction event
func (a AuctionState) AuctionStart() bool {
	return a.start
}

// AuctionExtended - called to confirm we will not leave auction, returns the event to be sent
// or nil if the auction wasn't extended
func (a *AuctionState) AuctionExtended(ctx context.Context) *events.Auction {
	if a.extension == nil {
		return nil
	}
	a.start = false
	end := int64(0)
	if a.begin == nil {
		b := vegatime.Now()
		a.begin = &b
	}
	if a.end != nil && a.end.Duration > 0 {
		end = a.begin.Add(time.Duration(a.end.Duration) * time.Second).UnixNano()
	}
	ext := *a.extension
	// set extension flag to nil
	a.extension = nil
	return events.NewAuctionEvent(ctx, a.m.Id, false, a.begin.UnixNano(), end, a.trigger, ext)
}

// AuctionStarted is called by the execution package to set flags indicating the market has started the auction
func (a *AuctionState) AuctionStarted(ctx context.Context) *events.Auction {
	a.openingAuctionTimer = metrics.NewTimeCounter(a.m.Id, "auction", "Auction duration")
	a.start = false
	end := int64(0)
	if a.begin == nil {
		b := vegatime.Now()
		a.begin = &b
	}
	if a.end != nil && a.end.Duration > 0 {
		end = a.begin.Add(time.Duration(a.end.Duration) * time.Second).UnixNano()
	}
	return events.NewAuctionEvent(ctx, a.m.Id, false, a.begin.UnixNano(), end, a.trigger)
}

// AuctionEnded is called by execution to update internal state indicating this auction was closed
func (a *AuctionState) AuctionEnded(ctx context.Context, now time.Time) *events.Auction {
	a.openingAuctionTimer.EngineTimeCounterAdd()

	// the end-of-auction event
	var start int64 = 0
	if a.begin != nil {
		start = a.begin.UnixNano()
	}
	evt := events.NewAuctionEvent(ctx, a.m.Id, true, start, now.UnixNano(), a.trigger)
	a.start, a.stop = false, false
	a.begin, a.end = nil, nil
	a.trigger = types.AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED
	a.mode = a.defMode
	// default mode is auction, this is an FBA market
	if a.mode == types.Market_TRADING_MODE_BATCH_AUCTION {
		a.trigger = types.AuctionTrigger_AUCTION_TRIGGER_BATCH
	}
	return evt
}

// UpdateMinDuration - see if we need to update the end value for current auction duration (if any)
// if the auction duration increases, an auction event will be returned
func (a *AuctionState) UpdateMinDuration(ctx context.Context, d time.Duration) *events.Auction {
	oldExp := a.ExpiresAt()
	// oldExp is nil if we're not in auction
	if oldExp == nil {
		return nil
	}
	// calc new end for auction:
	newMin := a.begin.Add(d)
	// no need to check for nil, we already have
	if newMin.After(*oldExp) {
		a.end.Duration += int64(newMin.Sub(*oldExp) / time.Second) // we have to divide by seconds as we're using secondws in AuctionDuration type
		return events.NewAuctionEvent(ctx, a.m.Id, false, a.begin.UnixNano(), newMin.UnixNano(), a.trigger)
	}
	return nil
}
