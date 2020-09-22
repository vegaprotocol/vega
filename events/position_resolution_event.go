package events

import (
	"context"
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
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

func (p PosRes) Proto() types.PositionResolution {
	return types.PositionResolution{
		MarketID:   p.marketID,
		Closed:     int64(p.closed),
		Distressed: int64(p.distressed),
		MarkPrice:  p.markPrice,
	}
}

func (p PosRes) MarketProto() types.MarketEvent {
	return types.MarketEvent{
		MarketID: p.marketID,
		Payload:  p.MarketEvent(),
	}
}

func (p PosRes) StreamMessage() *types.BusEvent {
	pr := p.Proto()
	return &types.BusEvent{
		ID:   p.eventID(),
		Type: p.et.ToProto(),
		Event: &types.BusEvent_PositionResolution{
			PositionResolution: &pr,
		},
	}
}
