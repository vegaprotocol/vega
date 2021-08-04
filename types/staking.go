package types

import (
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type StakingEventKind = eventspb.StakingEvent_Kind

const (
	StakingEventKindUnspecified = eventspb.StakingEvent_KIND_UNSPECIFIED
	StakingEventKindDeposited   = eventspb.StakingEvent_KIND_DEPOSIT
	StakingEventKindRemoved     = eventspb.StakingEvent_KIND_REMOVE
)

type StakingEvent struct {
	ID     string
	Kind   StakingEventKind
	TS     int64
	Party  string
	Amount *num.Uint
}

func (s *StakingEvent) IntoProto() *eventspb.StakingEvent {
	return &eventspb.StakingEvent{
		Id:     s.ID,
		Kind:   s.Kind,
		Ts:     s.TS,
		Party:  s.Party,
		Amount: num.UintToString(s.Amount),
	}
}
