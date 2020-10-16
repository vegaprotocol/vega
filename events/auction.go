package events

import (
	"context"
	"fmt"
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

type Auction struct {
	*Base
	// marketID for the market creating the auction event
	marketID string
	// start time in nanoseconds since 1/1/1970 of the current/last auction
	auctionStart int64
	// stop time in nanoseconds since 1/1/1970 of the current/last auction
	auctionStop int64
	// is/was this an opening auction
	openingAuction bool
	// are we leaving the auction (=true) or entering an auction (=false)
	leave bool
	// what precisely triggered the auction
	trigger types.AuctionTrigger
}

// NewAuctionEvent creates a new auction event object
func NewAuctionEvent(ctx context.Context, marketID string, leave bool, start, stop int64, trigger types.AuctionTrigger) *Auction {
	opening := (trigger == types.AuctionTrigger_AUCTION_TRIGGER_OPENING)
	return &Auction{
		Base:           newBase(ctx, AuctionEvent),
		marketID:       marketID,
		auctionStart:   start,
		auctionStop:    stop,
		openingAuction: opening,
		leave:          leave,
		trigger:        trigger,
	}
}

func (a Auction) MarketID() string {
	return a.marketID
}

// Auction returns the action performed (either true=leave auction, or false=entering auction)
func (a Auction) Auction() bool {
	return a.leave
}

// MarketEvent - implement market event interface so we can log this event
func (a Auction) MarketEvent() string {
	// is in auction
	start := time.Unix(0, a.auctionStart).Format(time.RFC3339Nano)
	stopT := time.Unix(0, a.auctionStop)
	if a.leave {
		if a.auctionStop == 0 {
			stopT = time.Now()
		}
		stop := stopT.Format(time.RFC3339Nano)
		if a.openingAuction {
			return fmt.Sprintf("Market %s left opening auction started at %s at %s (trigger: %s)", a.marketID, start, stop, a.trigger)
		}
		return fmt.Sprintf("Market %s left auction started at %s at %s (trigger: %s)", a.marketID, start, stop, a.trigger)
	}
	if a.openingAuction {
		// an opening auction will always have a STOP time
		stop := stopT.Format(time.RFC3339Nano)
		return fmt.Sprintf("Market %s entered opening auction at %s, will close at %s (trigger: %s)", a.marketID, start, stop, a.trigger)
	}
	if a.auctionStop == 0 {
		return fmt.Sprintf("Market %s entered auction mode at %s (trigger: %s)", a.marketID, start, a.trigger)
	}
	return fmt.Sprintf("Market %s entered auction mode at %s, auction closes at %s (trigger: %s)", a.marketID, start, stopT.Format(time.RFC3339Nano), a.trigger)
}

// Proto wrap event data in a proto message
func (a Auction) Proto() types.AuctionEvent {
	return types.AuctionEvent{
		MarketID:       a.marketID,
		OpeningAuction: a.openingAuction,
		Leave:          a.leave,
		Start:          a.auctionStart,
		End:            a.auctionStop,
		Trigger:        a.trigger,
	}
}

// StreamMessage returns the BusEvent message for the event stream API
func (a Auction) StreamMessage() *types.BusEvent {
	p := a.Proto()
	return &types.BusEvent{
		ID:    a.eventID(),
		Block: a.TraceID(),
		Type:  a.et.ToProto(),
		Event: &types.BusEvent_Auction{
			Auction: &p,
		},
	}
}

// StreamMarketMessage - allows for this event to be streamed as just a market event
// containing just market ID and a string akin to a log message
func (a Auction) StreamMarketMessage() *types.BusEvent {
	return &types.BusEvent{
		ID:    a.eventID(),
		Block: a.TraceID(),
		Type:  types.BusEventType_BUS_EVENT_TYPE_MARKET,
		Event: &types.BusEvent_Market{
			Market: &types.MarketEvent{
				MarketID: a.marketID,
				Payload:  a.MarketEvent(),
			},
		},
	}
}
