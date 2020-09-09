package events

import (
	"context"
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
	// are we entering or leaving the auction
	leave bool
}

// NewAuctionEvent creates a new auction event object
func NewAuctionEvent(ctx context.Context, marketID string, leave bool, start, stop int64, opening bool) *Auction {
	return &Auction{
		Base:           newBase(ctx, AuctionEvent),
		marketID:       marketID,
		auctionStart:   start,
		auctionStop:    stop,
		openingAuction: opening,
		leave:          leave,
	}
}

// Auction returns the action performed (either true=leave auction, or false=enter)
func (a *Auction) Auction() bool {
	return a.leave
}
