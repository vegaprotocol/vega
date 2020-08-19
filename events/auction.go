package events

import (
	"context"
)

type Auction struct {
	*Base
	leave bool
}

// NewAuctionEvent creates a new auction event object
func NewAuctionEvent(ctx context.Context, leave bool) *Auction {
	return &Auction{
		Base:  newBase(ctx, AuctionEvent),
		leave: leave,
	}
}

// Auction returns the action performed (either true=leave auction, or false=enter)
func (a *Auction) Auction() bool {
	return a.leave
}
