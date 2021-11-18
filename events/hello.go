package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

// Hello event emitted by a broker when a client first connects
type Hello struct {
	*Base
	chainId string
}

func NewHello(ctx context.Context, chainId string) *Hello {
	return &Hello{
		Base:    newBase(ctx, HelloEvent),
		chainId: chainId,
	}
}

func (t Hello) ChainId() string {
	return t.chainId
}

func (t Hello) Proto() eventspb.HelloEvent {
	return eventspb.HelloEvent{
		ChainId: t.chainId,
	}
}

func (t Hello) StreamMessage() *eventspb.BusEvent {
	p := t.Proto()
	return &eventspb.BusEvent{
		Version: eventspb.Version,
		Id:      t.eventID(),
		Block:   t.TraceID(),
		Type:    t.et.ToProto(),
		Event: &eventspb.BusEvent_HelloEvent{
			HelloEvent: &p,
		},
	}
}

func HelloEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Hello {
	return &Hello{
		Base:    newBaseFromStream(ctx, HelloEvent, be),
		chainId: be.GetHelloEvent().ChainId,
	}
}
