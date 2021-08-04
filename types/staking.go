package types

import (
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type StakingEventType = eventspb.StakingEvent_Type

const (
	StakingEventTypeUnspecified = eventspb.StakingEvent_TYPE_UNSPECIFIED
	StakingEventTypeDeposited   = eventspb.StakingEvent_TYPE_DEPOSIT
	StakingEventTypeRemoved     = eventspb.StakingEvent_TYPE_REMOVE
)

type StakingEvent struct {
	ID     string
	Type   StakingEventType
	TS     int64
	Party  string
	Amount *num.Uint
}

func (s *StakingEvent) IntoProto() *eventspb.StakingEvent {
	return &eventspb.StakingEvent{
		Id:     s.ID,
		Type:   s.Type,
		Ts:     s.TS,
		Party:  s.Party,
		Amount: num.UintToString(s.Amount),
	}
}
