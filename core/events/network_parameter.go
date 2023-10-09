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

type NetworkParameter struct {
	*Base
	np proto.NetworkParameter
}

func NewNetworkParameterEvent(ctx context.Context, key, value string) *NetworkParameter {
	return &NetworkParameter{
		Base: newBase(ctx, NetworkParameterEvent),
		np:   proto.NetworkParameter{Key: key, Value: value},
	}
}

func (n *NetworkParameter) NetworkParameter() proto.NetworkParameter {
	return n.np
}

func (n NetworkParameter) Proto() proto.NetworkParameter {
	return n.np
}

func (n NetworkParameter) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(n.Base)
	busEvent.Event = &eventspb.BusEvent_NetworkParameter{
		NetworkParameter: &n.np,
	}

	return busEvent
}

func NetworkParameterEventFromStream(ctx context.Context, be *eventspb.BusEvent) *NetworkParameter {
	return &NetworkParameter{
		Base: newBaseFromBusEvent(ctx, NetworkParameterEvent, be),
		np:   proto.NetworkParameter{Key: be.GetNetworkParameter().Key, Value: be.GetNetworkParameter().Value},
	}
}
