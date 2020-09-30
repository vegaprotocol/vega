package execution

import (
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

// Trigger - placeholder type for proto enum
type Trigger int

const (
	Trigger_None Trigger = iota // none -> continuous trading
	Trigger_OpeningAuction
	Trigger_FBA
	Trigger_PriceMonitoring
	Trigger_LiquidityMonitoring
)

type auctionState struct {
	mode        types.MarketState      // current trading mode
	defMode     types.MarketState      // default trading mode for market
	trigger     Trigger                // Set to the value indicating what started the auction
	begin       *time.Time             // optional setting auction start time (will be set if start flag is true)
	end         *types.AuctionDuration // will be set when in auction, defines parameters that end an auction period
	start, stop bool                   // flags to clarify whether we're entering or leaving auction
	m           *types.Market          // keep market definition handy, useful to end auctions when default is FBA
}

func newAuctionState(mkt *types.Market, now time.Time) *auctionState {
	s := auctionState{
		mode:    types.MarketState_MARKET_STATE_AUCTION,
		defMode: types.MarketState_MARKET_STATE_CONTINUOUS,
		trigger: Trigger_OpeningAuction,
		begin:   &now,
		end:     mkt.OpeningAuction,
		start:   true,
	}
	if mkt.GetContinuous() == nil {
		s.defMode = types.MarketState_MARKET_STATE_AUCTION
	}
	// no opening auction
	if mkt.OpeningAuction == nil {
		s.mode = s.defMode
		s.begin = nil
		s.start = false
		s.trigger = Trigger_None
	}
	return &s
}

// StartLiquidityAuction - set the state to start a liquidity triggered auction
// @TODO these functions will be removed once the types are in proto
func (a *auctionState) StartLiquidityAuction(t time.Time, d *types.AuctionDuration) {
	a.mode = types.MarketState_MARKET_STATE_AUCTION // auction mode
	a.trigger = Trigger_LiquidityMonitoring
	a.start = true
	a.stop = false
	a.begin = &t
	a.end = d
}

// StartPriceAuction - set the state to start a price triggered auction
// @TODO these functions will be removed once the types are in proto
func (a *auctionState) StartPriceAuction(t time.Time, d *types.AuctionDuration) {
	a.mode = types.MarketState_MARKET_STATE_AUCTION // auction mode
	a.trigger = Trigger_PriceMonitoring
	a.start = true
	a.stop = false
	a.begin = &t
	a.end = d
}

// ExtendDuration - extend current auction, leaving trigger etc... in tact
func (a *auctionState) ExtendAuction(delta types.AuctionDuration) {
	a.end.Duration += delta.Duration
	a.end.Volume += delta.Volume
	a.stop = false // the auction was supposed to stop, but we've extended it
}

// EndAuction is called by monitoring engines to mark if an auction period has expired
func (a *auctionState) EndAuction() {
	a.stop = true
}

// Duration returns a copy of the current auction duration object
func (a auctionState) Duration() types.AuctionDuration {
	if a.end == nil {
		return types.AuctionDuration{}
	}
	return *a.end
}

// Start - returns time pointer of the start of the auction (nil if not in auction)
func (a auctionState) Start() time.Time {
	if a.begin == nil {
		return time.Time{} // zero time
	}
	return *a.begin
}

// Mode returns current trading mode
func (a auctionState) Mode() types.MarketState {
	return a.mode
}

// InAuction returns bool if the market is in auction for any reason
func (a auctionState) InAuction() bool {
	return (a.trigger != Trigger_None)
}

func (a auctionState) IsOpeningAuction() bool {
	return (a.trigger == Trigger_OpeningAuction)
}

func (a auctionState) IsLiquidityAuction() bool {
	return (a.trigger == Trigger_LiquidityMonitoring)
}

func (a auctionState) IsPriceAuction() bool {
	return (a.trigger == Trigger_PriceMonitoring)
}

func (a auctionState) IsFBA() bool {
	return (a.trigger == Trigger_FBA)
}

// AuctionEnd bool indicating whether auction should be closed or not, if true, we can still extend the auction
// but when the market takes over (after monitoring engines), the auction will be closed
func (a auctionState) AuctionEnd() bool {
	return a.stop
}

// AuctionStart bool indicates something has already triggered an auction to start, we can skip other monitoring potentially
// and we know to create an auction event
func (a auctionState) AuctionStart() bool {
	return a.start
}

// AuctionStarted is called by the execution package to set flags indicating the market has started the auction
func (a *auctionState) AuctionStarted() {
	a.start = false
}

// AuctionEnded is called by execution to update internal state indicating this auction was closed
func (a *auctionState) AuctionEnded() {
	a.start, a.stop = false, false
	a.begin, a.end = nil, nil
	a.trigger = Trigger_None
	a.mode = a.defMode
	// default mode is auction, this is an FBA market
	if a.mode == types.MarketState_MARKET_STATE_AUCTION {
		a.trigger = Trigger_FBA
	}
}
