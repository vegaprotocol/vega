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
