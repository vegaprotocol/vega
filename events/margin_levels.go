package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

// MarginLevels - the margin levels event
type MarginLevels struct {
	*Base
	l types.MarginLevels
}

func NewMarginLevelsEvent(ctx context.Context, l types.MarginLevels) *MarginLevels {
	return &MarginLevels{
		Base: newBase(ctx, MarginLevelsEvent),
		l:    l,
	}
}

func (m MarginLevels) MarginLevels() types.MarginLevels {
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

func (m MarginLevels) Proto() types.MarginLevels {
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
