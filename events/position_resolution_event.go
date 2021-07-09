package events

import (
	"context"
	"fmt"

	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

type PosRes struct {
	*Base
	distressed, closed int
	marketID           string
	markPrice          uint64
}

func NewPositionResolution(ctx context.Context, distressed, closed int, markPrice uint64, marketID string) *PosRes {
	base := newBase(ctx, PositionResolution)
	return &PosRes{
		Base:       base,
		distressed: distressed,
		closed:     closed,
		markPrice:  markPrice,
		marketID:   marketID,
	}
}

// MarketEvent implement the MarketEvent interface
func (p PosRes) MarketEvent() string {
	return fmt.Sprintf("Market %s entered position resolution, %d traders were distressed, %d of which were closed out at mark price %d", p.marketID, p.distressed, p.closed, p.markPrice)
}

func (p PosRes) MarketID() string {
	return p.marketID
}

func (p PosRes) MarkPrice() uint64 {
	return p.markPrice
}

func (p PosRes) Distressed() int {
	return p.distressed
}

func (p PosRes) Closed() int {
	return p.closed
}

func (p PosRes) Proto() eventspb.PositionResolution {
	return eventspb.PositionResolution{
		MarketId:   p.marketID,
		Closed:     int64(p.closed),
		Distressed: int64(p.distressed),
		MarkPrice:  p.markPrice,
	}
}

func (p PosRes) MarketProto() eventspb.MarketEvent {
	return eventspb.MarketEvent{
		MarketId: p.marketID,
		Payload:  p.MarketEvent(),
	}
}

func (p PosRes) StreamMessage() *eventspb.BusEvent {
	pr := p.Proto()
	return &eventspb.BusEvent{
		Id:    p.eventID(),
		Block: p.TraceID(),
		Type:  p.et.ToProto(),
		Event: &eventspb.BusEvent_PositionResolution{
			PositionResolution: &pr,
		},
	}
}

func (p PosRes) StreamMarketMessage() *eventspb.BusEvent {
	msg := p.MarketProto()
	return &eventspb.BusEvent{
		Id:   p.eventID(),
		Type: eventspb.BusEventType_BUS_EVENT_TYPE_MARKET,
		Event: &eventspb.BusEvent_Market{
			Market: &msg,
		},
	}
}
