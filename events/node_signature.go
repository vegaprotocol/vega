package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

// NodeSignature ...
type NodeSignature struct {
	*Base
	e types.NodeSignature
}

func NewNodeSignatureEvent(ctx context.Context, e types.NodeSignature) *NodeSignature {
	cpy := e.DeepClone()
	return &NodeSignature{
		Base: newBase(ctx, NodeSignatureEvent),
		e:    *cpy,
	}
}

func (n NodeSignature) NodeSignature() types.NodeSignature {
	return n.e
}

func (n NodeSignature) Proto() types.NodeSignature {
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
