package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

type NetworkParameter struct {
	*Base
	np types.NetworkParameter
}

func NewNetworkParameterEvent(ctx context.Context, key, value string) *NetworkParameter {
	return &NetworkParameter{
		Base: newBase(ctx, NetworkParameterEvent),
		np:   types.NetworkParameter{Key: key, Value: value},
	}
}

func (n *NetworkParameter) NetworkParameter() types.NetworkParameter {
	return n.np
}

func (n NetworkParameter) Proto() types.NetworkParameter {
	return n.np
}

func (n NetworkParameter) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    n.eventID(),
		Block: n.TraceID(),
		Type:  n.et.ToProto(),
		Event: &eventspb.BusEvent_NetworkParameter{
			NetworkParameter: &n.np,
		},
	}
}
