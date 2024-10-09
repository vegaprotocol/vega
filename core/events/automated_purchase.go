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

package events

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type AutomatedPurchaseAnnounced struct {
	*Base
	From            string
	FromAccountType types.AccountType
	ToAccountType   types.AccountType
	MarketID        string
	Amount          *num.Uint
}

func NewProtocolAutomatedPurchaseAnnouncedEvent(ctx context.Context, from string, fromAccountType types.AccountType, toAccountType types.AccountType, marketID string, amount *num.Uint) *AutomatedPurchaseAnnounced {
	return &AutomatedPurchaseAnnounced{
		Base:            newBase(ctx, AutomatedPurchaseAnnouncedEvent),
		From:            from,
		FromAccountType: fromAccountType,
		ToAccountType:   toAccountType,
		MarketID:        marketID,
		Amount:          amount,
	}
}

func (ap AutomatedPurchaseAnnounced) Proto() eventspb.AutomatedPurchaseAnnounced {
	return eventspb.AutomatedPurchaseAnnounced{
		From:            ap.From,
		FromAccountType: ap.FromAccountType,
		ToAccountType:   ap.ToAccountType,
		MarketId:        ap.MarketID,
		Amount:          ap.Amount.String(),
	}
}

func (ap AutomatedPurchaseAnnounced) AutomatedPurchaseAnnouncedEvent() eventspb.AutomatedPurchaseAnnounced {
	return ap.Proto()
}

func (ap AutomatedPurchaseAnnounced) StreamMessage() *eventspb.BusEvent {
	p := ap.Proto()
	busEvent := newBusEventFromBase(ap.Base)
	busEvent.Event = &eventspb.BusEvent_AutomatedPurchaseAnnounced{
		AutomatedPurchaseAnnounced: &p,
	}

	return busEvent
}

func AutomatedPurchaseAnnouncedFromStream(ctx context.Context, be *eventspb.BusEvent) *AutomatedPurchaseAnnounced {
	event := be.GetAutomatedPurchaseAnnounced()
	if event == nil {
		return nil
	}

	return &AutomatedPurchaseAnnounced{
		Base:            newBaseFromBusEvent(ctx, AutomatedPurchaseAnnouncedEvent, be),
		From:            event.From,
		FromAccountType: event.FromAccountType,
		ToAccountType:   event.ToAccountType,
		MarketID:        event.MarketId,
		Amount:          num.MustUintFromString(event.Amount, 10),
	}
}
