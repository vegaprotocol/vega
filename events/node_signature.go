package events

import (
	"context"

	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
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
	return &eventspb.BusEvent{
		Id:    n.eventID(),
		Block: n.TraceID(),
		Type:  n.et.ToProto(),
		Event: &eventspb.BusEvent_NodeSignature{
			NodeSignature: &n.e,
		},
	}
}
