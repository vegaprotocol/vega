package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

// StreamStart event emitted by a broker when a client first connects.
type StreamStart struct {
	*Base
	chainId string
}

func NewStreamStart(ctx context.Context, chainId string) *StreamStart {
	return &StreamStart{
		Base:    newBase(ctx, StreamStartEvent),
		chainId: chainId,
	}
}

func (t StreamStart) ChainId() string {
	return t.chainId
}

func (t StreamStart) Proto() eventspb.StreamStartEvent {
	return eventspb.StreamStartEvent{
		ChainId: t.chainId,
	}
}

func (t StreamStart) StreamMessage() *eventspb.BusEvent {
	p := t.Proto()
	return &eventspb.BusEvent{
		Version: eventspb.Version,
		Id:      t.eventID(),
		Block:   t.TraceID(),
		Type:    t.et.ToProto(),
		Event: &eventspb.BusEvent_StreamStart{
			StreamStart: &p,
		},
	}
}

func StreamStartFromStream(ctx context.Context, be *eventspb.BusEvent) *StreamStart {
	return &StreamStart{
		Base:    newBaseFromStream(ctx, StreamStartEvent, be),
		chainId: be.GetStreamStart().ChainId,
	}
}
