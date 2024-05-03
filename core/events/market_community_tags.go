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

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type MarketCommunityTags struct {
	*Base
	a eventspb.MarketCommunityTags
}

func NewMarketCommunityTagsEvent(ctx context.Context, e eventspb.MarketCommunityTags) *MarketCommunityTags {
	return &MarketCommunityTags{
		Base: newBase(ctx, MarketCommunityTagsEvent),
		a:    e,
	}
}

func (a *MarketCommunityTags) MarketCommunityTags() eventspb.MarketCommunityTags {
	return a.a
}

func (a MarketCommunityTags) Proto() eventspb.MarketCommunityTags {
	return a.a
}

func (a MarketCommunityTags) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(a.Base)
	busEvent.Event = &eventspb.BusEvent_MarketCommunityTags{
		MarketCommunityTags: &a.a,
	}
	return busEvent
}

func MarketCommunityTagsEventFromStream(ctx context.Context, be *eventspb.BusEvent) *MarketCommunityTags {
	return &MarketCommunityTags{
		Base: newBaseFromBusEvent(ctx, MarketCommunityTagsEvent, be),
		a:    *be.GetMarketCommunityTags(),
	}
}
