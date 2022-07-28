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

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
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
