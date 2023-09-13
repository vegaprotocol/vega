package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"github.com/pkg/errors"
)

type (
	TeamCreatedEvent interface {
		events.Event
		TeamCreated() eventspb.TeamCreated
	}

	TeamUpdateEvent interface {
		events.Event
		TeamUpdated() eventspb.TeamUpdated
	}

	RefereeJoinedTeam interface {
		events.Event
		RefereeJoinedTeam() eventspb.RefereeJoinedTeam
	}

	RefereeSwitchedTeam interface {
		events.Event
		RefereeSwitchedTeam() eventspb.RefereeSwitchedTeam
	}

	TeamStore interface {
		AddTeam(ctx context.Context, team *entities.Team) error
		UpdateTeam(ctx context.Context, team *entities.TeamUpdated) error
		RefereeJoinedTeam(ctx context.Context, referee *entities.TeamMember) error
		RefereeSwitchedTeam(ctx context.Context, referee *entities.RefereeTeamSwitch) error
	}

	Teams struct {
		subscriber
		store TeamStore
	}
)

func NewTeams(store TeamStore) *Teams {
	return &Teams{
		store: store,
	}
}

func (t *Teams) Types() []events.Type {
	return []events.Type{
		events.TeamCreatedEvent,
		events.TeamUpdatedEvent,
		events.RefereeJoinedTeamEvent,
		events.RefereeSwitchedTeamEvent,
	}
}

func (t *Teams) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case TeamCreatedEvent:
		return t.consumeTeamCreatedEvent(ctx, e)
	case TeamUpdateEvent:
		return t.consumeTeamUpdateEvent(ctx, e)
	case RefereeJoinedTeam:
		return t.consumeRefereeJoinedTeamEvent(ctx, e)
	case RefereeSwitchedTeam:
		return t.consumeRefereeSwitchedTeamEvent(ctx, e)
	default:
		return nil
	}
}

func (t *Teams) consumeTeamCreatedEvent(ctx context.Context, e TeamCreatedEvent) error {
	createdEvt := e.TeamCreated()
	created := entities.TeamCreatedFromProto(&createdEvt, t.vegaTime)
	return errors.Wrap(t.store.AddTeam(ctx, created), "adding team")
}

func (t *Teams) consumeTeamUpdateEvent(ctx context.Context, e TeamUpdateEvent) error {
	updatedEvt := e.TeamUpdated()
	updated := entities.TeamUpdatedFromProto(&updatedEvt, t.vegaTime)
	return errors.Wrap(t.store.UpdateTeam(ctx, updated), "updating team")
}

func (t *Teams) consumeRefereeJoinedTeamEvent(ctx context.Context, e RefereeJoinedTeam) error {
	joinedEvt := e.RefereeJoinedTeam()
	referee := entities.TeamRefereeFromProto(&joinedEvt, t.vegaTime)
	return errors.Wrap(t.store.RefereeJoinedTeam(ctx, referee), "adding referee to team")
}

func (t *Teams) consumeRefereeSwitchedTeamEvent(ctx context.Context, e RefereeSwitchedTeam) error {
	switchedEvt := e.RefereeSwitchedTeam()
	switched := entities.TeamRefereeHistoryFromProto(&switchedEvt, t.vegaTime)
	return errors.Wrap(t.store.RefereeSwitchedTeam(ctx, switched), "updating referee history")
}
