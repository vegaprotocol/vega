package events

import (
	"context"

	"code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	"code.vegaprotocol.io/vega/types"
)

type MarketData struct {
	*Base
	md types.MarketData
}

func NewMarketDataEvent(ctx context.Context, md types.MarketData) *MarketData {
	cpy := md.DeepClone()
	return &MarketData{
		Base: newBase(ctx, MarketDataEvent),
		md:   *cpy,
	}
}

func (m MarketData) MarketID() string {
	return m.md.Market
}

func (m MarketData) MarketData() types.MarketData {
	return m.md
}

func (m MarketData) Proto() proto.MarketData {
	md := m.md.IntoProto()
	return *md
}

func (m MarketData) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    m.eventID(),
		Block: m.TraceID(),
		Type:  m.et.ToProto(),
		Event: &eventspb.BusEvent_MarketData{
			MarketData: m.md.IntoProto(),
		},
	}
}
