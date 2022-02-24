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
