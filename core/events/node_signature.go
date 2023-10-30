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

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

// NodeSignature ...
type NodeSignature struct {
	*Base
	e commandspb.NodeSignature
}

func NewNodeSignatureEvent(ctx context.Context, e commandspb.NodeSignature) *NodeSignature {
	cpy := e.DeepClone()
	return &NodeSignature{
		Base: newBase(ctx, NodeSignatureEvent),
		e:    *cpy,
	}
}

func (n NodeSignature) NodeSignature() commandspb.NodeSignature {
	return n.e
}

func (n NodeSignature) Proto() commandspb.NodeSignature {
	return n.e
}

func (n NodeSignature) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(n.Base)
	busEvent.Event = &eventspb.BusEvent_NodeSignature{
		NodeSignature: &n.e,
	}

	return busEvent
}

func NodeSignatureEventFromStream(ctx context.Context, be *eventspb.BusEvent) *NodeSignature {
	return &NodeSignature{
		Base: newBaseFromBusEvent(ctx, NodeSignatureEvent, be),
		e:    *be.GetNodeSignature(),
	}
}
