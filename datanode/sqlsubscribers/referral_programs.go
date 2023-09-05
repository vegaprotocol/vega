package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	ReferralProgramStartedEvent interface {
		events.Event
		GetReferralProgramStarted() *eventspb.ReferralProgramStarted
	}

	ReferralProgramUpdatedEvent interface {
		events.Event
		GetReferralProgramUpdated() *eventspb.ReferralProgramUpdated
	}

	ReferralProgramEndedEvent interface {
		events.Event
		GetReferralProgramEnded() *eventspb.ReferralProgramEnded
	}

	ReferralStore interface {
		AddReferralProgram(ctx context.Context, referral *entities.ReferralProgram) error
		UpdateReferralProgram(ctx context.Context, referral *entities.ReferralProgram) error
		EndReferralProgram(ctx context.Context, referralID entities.ReferralProgramID, version uint64, vegaTime time.Time) error
	}

	ReferralProgram struct {
		subscriber
		store ReferralStore
	}
)

func NewReferralProgram(store ReferralStore) *ReferralProgram {
	return &ReferralProgram{
		store: store,
	}
}

func (rp *ReferralProgram) Types() []events.Type {
	return []events.Type{
		events.ReferralProgramStartedEvent,
		events.ReferralProgramUpdatedEvent,
		events.ReferralProgramEndedEvent,
	}
}

func (rp *ReferralProgram) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case ReferralProgramStartedEvent:
		return rp.consumeReferralProgramStartedEvent(ctx, e)
	case ReferralProgramUpdatedEvent:
		return rp.consumeReferralProgramUpdatedEvent(ctx, e)
	case ReferralProgramEndedEvent:
		return rp.consumeReferralProgramEndedEvent(ctx, e)
	default:
		return nil
	}
}

func (rp *ReferralProgram) consumeReferralProgramStartedEvent(ctx context.Context, e ReferralProgramStartedEvent) error {
	program := entities.ReferralProgramFromProto(e.GetReferralProgramStarted().GetProgram(), rp.vegaTime)
	return rp.store.AddReferralProgram(ctx, program)
}

func (rp *ReferralProgram) consumeReferralProgramUpdatedEvent(ctx context.Context, e ReferralProgramUpdatedEvent) error {
	program := entities.ReferralProgramFromProto(e.GetReferralProgramUpdated().GetProgram(), rp.vegaTime)
	return rp.store.UpdateReferralProgram(ctx, program)
}

func (rp *ReferralProgram) consumeReferralProgramEndedEvent(ctx context.Context, e ReferralProgramEndedEvent) error {
	referralID := entities.ReferralProgramID(e.GetReferralProgramEnded().GetId())
	return rp.store.EndReferralProgram(ctx, referralID, e.GetReferralProgramEnded().GetVersion(), rp.vegaTime)
}
