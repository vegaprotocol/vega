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

	proto "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type NetworkLimits struct {
	*Base
	nl *proto.NetworkLimits
}

func NewNetworkLimitsEvent(ctx context.Context, limits *proto.NetworkLimits) *NetworkLimits {
	return &NetworkLimits{
		Base: newBase(ctx, NetworkLimitsEvent),
		nl:   limits,
	}
}

func (n *NetworkLimits) NetworkLimits() *proto.NetworkLimits {
	return n.nl
}

func (n NetworkLimits) Proto() *proto.NetworkLimits {
	return n.nl
}

func (n NetworkLimits) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(n.Base)
	busEvent.Event = &eventspb.BusEvent_NetworkLimits{
		NetworkLimits: n.nl,
	}

	return busEvent
}

func NetworkLimitsEventFromStream(ctx context.Context, be *eventspb.BusEvent) *NetworkLimits {
	return &NetworkLimits{
		Base: newBaseFromBusEvent(ctx, NetworkLimitsEvent, be),
		nl:   be.GetNetworkLimits(),
	}
}
