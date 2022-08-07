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

package events

import (
	"context"
	"fmt"
	"time"

	types "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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
	// what component extended the ongoing auction (if any)
	extension types.AuctionTrigger
}

// NewAuctionEvent creates a new auction event object.
func NewAuctionEvent(ctx context.Context, marketID string, leave bool, start, stop int64, triggers ...types.AuctionTrigger) *Auction {
	if len(triggers) == 0 {
		return nil
	}
	trigger := triggers[0]
	opening := trigger == types.AuctionTrigger_AUCTION_TRIGGER_OPENING
	e := &Auction{
		Base:           newBase(ctx, AuctionEvent),
		marketID:       marketID,
		auctionStart:   start,
		auctionStop:    stop,
		openingAuction: opening,
		leave:          leave,
		trigger:        trigger,
	}
	if len(triggers) == 2 {
		e.extension = triggers[1]
	}

	return e
}

func (a Auction) MarketID() string {
	return a.marketID
}

// Auction returns the action performed (either true=leave auction, or false=entering auction).
func (a Auction) Auction() bool {
	return a.leave
}

// MarketEvent - implement market event interface so we can log this event.
func (a Auction) MarketEvent() string {
	// is in auction
	start := time.Unix(0, a.auctionStart).Format(time.RFC3339Nano)
	if a.extension != types.AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED {
		return fmt.Sprintf("Market %s in auction mode (%s) started at %s (extension reason: %s)", a.marketID, a.trigger, start, a.extension)
	}
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

// Proto wrap event data in a proto message.
func (a Auction) Proto() eventspb.AuctionEvent {
	return eventspb.AuctionEvent{
		MarketId:         a.marketID,
		OpeningAuction:   a.openingAuction,
		Leave:            a.leave,
		Start:            a.auctionStart,
		End:              a.auctionStop,
		Trigger:          a.trigger,
		ExtensionTrigger: a.extension,
	}
}

// StreamMessage returns the BusEvent message for the event stream API.
func (a Auction) StreamMessage() *eventspb.BusEvent {
	p := a.Proto()

	busEvent := newBusEventFromBase(a.Base)
	busEvent.Event = &eventspb.BusEvent_Auction{
		Auction: &p,
	}

	return busEvent
}

// StreamMarketMessage - allows for this event to be streamed as just a market event
// containing just market ID and a string akin to a log message.
func (a Auction) StreamMarketMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(a.Base)
	busEvent.Type = eventspb.BusEventType_BUS_EVENT_TYPE_MARKET
	busEvent.Event = &eventspb.BusEvent_Market{
		Market: &eventspb.MarketEvent{
			MarketId: a.marketID,
			Payload:  a.MarketEvent(),
		},
	}

	return busEvent
}

func AuctionEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Auction {
	e := &Auction{
		Base:           newBaseFromBusEvent(ctx, AuctionEvent, be),
		marketID:       be.GetAuction().MarketId,
		auctionStart:   be.GetAuction().Start,
		auctionStop:    be.GetAuction().End,
		openingAuction: be.GetAuction().OpeningAuction,
		leave:          be.GetAuction().Leave,
		trigger:        be.GetAuction().Trigger,
		extension:      be.GetAuction().ExtensionTrigger,
	}

	return e
}
