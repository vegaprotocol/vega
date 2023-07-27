package events

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/ptr"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type TeamCreated struct {
	*Base
	e eventspb.TeamCreated
}

func (t TeamCreated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_TeamCreated{
		TeamCreated: &t.e,
	}

	return busEvent
}

func NewTeamCreatedEvent(ctx context.Context, t *types.Team) *TeamCreated {
	return &TeamCreated{
		Base: newBase(ctx, TeamCreatedEvent),
		e: eventspb.TeamCreated{
			TeamId:    string(t.ID),
			Referrer:  string(t.Referrer),
			Name:      ptr.From(t.Name),
			TeamUrl:   ptr.From(t.TeamURL),
			AvatarUrl: ptr.From(t.AvatarURL),
		},
	}
}

func TeamCreatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TeamCreated {
	return &TeamCreated{
		Base: newBaseFromBusEvent(ctx, TeamCreatedEvent, be),
		e:    *be.GetTeamCreated(),
	}
}

type TeamUpdated struct {
	*Base
	e eventspb.TeamUpdated
}

func (t TeamUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_TeamUpdated{
		TeamUpdated: &t.e,
	}

	return busEvent
}

func NewTeamUpdatedEvent(ctx context.Context, t *types.Team) *TeamUpdated {
	return &TeamUpdated{
		Base: newBase(ctx, TeamUpdatedEvent),
		e: eventspb.TeamUpdated{
			TeamId:    string(t.ID),
			Name:      ptr.From(t.Name),
			TeamUrl:   ptr.From(t.TeamURL),
			AvatarUrl: ptr.From(t.AvatarURL),
		},
	}
}

func TeamUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TeamUpdated {
	return &TeamUpdated{
		Base: newBaseFromBusEvent(ctx, TeamUpdatedEvent, be),
		e:    *be.GetTeamUpdated(),
	}
}

type RefereeSwitchedTeam struct {
	*Base
	e eventspb.RefereeSwitchedTeam
}

func (t RefereeSwitchedTeam) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_RefereeSwitchedTeam{
		RefereeSwitchedTeam: &t.e,
	}

	return busEvent
}

func NewRefereeSwitchedTeamEvent(ctx context.Context, from, to types.TeamID, referee types.PartyID) *RefereeSwitchedTeam {
	return &RefereeSwitchedTeam{
		Base: newBase(ctx, RefereeSwitchedTeamEvent),
		e: eventspb.RefereeSwitchedTeam{
			FromTeamId: string(from),
			ToTeamId:   string(to),
			Referee:    string(referee),
		},
	}
}

func RefereeSwitchedTeamEventFromStream(ctx context.Context, be *eventspb.BusEvent) *RefereeSwitchedTeam {
	return &RefereeSwitchedTeam{
		Base: newBaseFromBusEvent(ctx, RefereeSwitchedTeamEvent, be),
		e:    *be.GetRefereeSwitchedTeam(),
	}
}

type RefereeJoinedTeam struct {
	*Base
	e eventspb.RefereeJoinedTeam
}

func (t RefereeJoinedTeam) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_RefereeJoinedTeam{
		RefereeJoinedTeam: &t.e,
	}

	return busEvent
}

func NewRefereeJoinedTeamEvent(ctx context.Context, teamID types.TeamID, referee types.PartyID) *RefereeJoinedTeam {
	return &RefereeJoinedTeam{
		Base: newBase(ctx, RefereeJoinedTeamEvent),
		e: eventspb.RefereeJoinedTeam{
			TeamId:  string(teamID),
			Referee: string(referee),
		},
	}
}

func RefereeJoinedTeamEventFromStream(ctx context.Context, be *eventspb.BusEvent) *RefereeJoinedTeam {
	return &RefereeJoinedTeam{
		Base: newBaseFromBusEvent(ctx, RefereeJoinedTeamEvent, be),
		e:    *be.GetRefereeJoinedTeam(),
	}
}
