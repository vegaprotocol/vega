package events

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/data-node/types"
)

// MarginLevels - the margin levels event
type MarginLevels struct {
	*Base
	l proto.MarginLevels
}

func NewMarginLevelsEvent(ctx context.Context, l types.MarginLevels) *MarginLevels {
	return &MarginLevels{
		Base: newBase(ctx, MarginLevelsEvent),
		l:    *l.IntoProto(),
	}
}

func (m MarginLevels) MarginLevels() proto.MarginLevels {
	return m.l
}

func (m MarginLevels) IsParty(id string) bool {
	return m.l.PartyId == id
}

func (m MarginLevels) PartyID() string {
	return m.l.PartyId
}

func (m MarginLevels) MarketID() string {
	return m.l.MarketId
}

func (m MarginLevels) Asset() string {
	return m.l.Asset
}

func (m MarginLevels) Proto() proto.MarginLevels {
	return m.l
}

func (m MarginLevels) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    m.eventID(),
		Block: m.TraceID(),
		Type:  m.et.ToProto(),
		Event: &eventspb.BusEvent_MarginLevels{
			MarginLevels: &m.l,
		},
	}
}
