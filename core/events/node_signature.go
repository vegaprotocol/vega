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

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
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
