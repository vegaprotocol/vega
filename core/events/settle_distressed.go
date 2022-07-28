// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/core/types/num"
)

type SettleDistressed struct {
	*Base
	partyID  string
	marketID string
	margin   *num.Uint
	price    *num.Uint
	ts       int64
}

func NewSettleDistressed(ctx context.Context, partyID, marketID string, price, margin *num.Uint, ts int64) *SettleDistressed {
	return &SettleDistressed{
		Base:     newBase(ctx, SettleDistressedEvent),
		partyID:  partyID,
		marketID: marketID,
		margin:   margin.Clone(),
		price:    price.Clone(),
		ts:       ts,
	}
}

func (s SettleDistressed) IsParty(id string) bool {
	return s.partyID == id
}

func (s SettleDistressed) PartyID() string {
	return s.partyID
}

func (s SettleDistressed) MarketID() string {
	return s.marketID
}

func (s SettleDistressed) Margin() *num.Uint {
	return s.margin.Clone()
}

func (s SettleDistressed) Price() *num.Uint {
	return s.price.Clone()
}

func (s SettleDistressed) Timestamp() int64 {
	return s.ts
}

func (s SettleDistressed) Proto() eventspb.SettleDistressed {
	return eventspb.SettleDistressed{
		MarketId: s.marketID,
		PartyId:  s.partyID,
		Margin:   s.margin.String(),
		Price:    s.price.String(),
	}
}

func (s SettleDistressed) StreamMessage() *eventspb.BusEvent {
	p := s.Proto()

	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_SettleDistressed{
		SettleDistressed: &p,
	}

	return busEvent
}

func SettleDistressedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *SettleDistressed {
	sd := be.GetSettleDistressed()
	sdMargin, marginOverflow := num.UintFromString(sd.Margin, 10)
	sdPrice, priceOverflow := num.UintFromString(sd.Price, 10)

	if marginOverflow || priceOverflow {
		return nil
	}

	return &SettleDistressed{
		Base:     newBaseFromBusEvent(ctx, SettleDistressedEvent, be),
		partyID:  sd.PartyId,
		marketID: sd.MarketId,
		margin:   sdMargin,
		price:    sdPrice,
	}
}
