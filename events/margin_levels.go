package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
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
	return (m.l.PartyID == id)
}

func (m MarginLevels) PartyID() string {
	return m.l.PartyID
}

func (m MarginLevels) MarketID() string {
	return m.l.MarketID
}

func (m MarginLevels) Asset() string {
	return m.l.Asset
}

func (m MarginLevels) Proto() types.MarginLevels {
	return m.l
}

func (m MarginLevels) StreamMessage() *types.BusEvent {
	return &types.BusEvent{
		ID:   m.traceID,
		Type: m.et.ToProto(),
		Event: &types.BusEvent_MarginLevels{
			MarginLevels: &m.l,
		},
	}
}
