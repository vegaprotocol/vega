// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package events

import (
	"context"
	"slices"
	"strings"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
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

func (t TeamCreated) TeamCreated() *eventspb.TeamCreated {
	return &t.e
}

func NewTeamCreatedEvent(ctx context.Context, epoch uint64, t *types.Team) *TeamCreated {
	e := eventspb.TeamCreated{
		TeamId:    string(t.ID),
		Referrer:  string(t.Referrer.PartyID),
		Name:      t.Name,
		TeamUrl:   ptr.From(t.TeamURL),
		AvatarUrl: ptr.From(t.AvatarURL),
		CreatedAt: t.CreatedAt.UnixNano(),
		AtEpoch:   epoch,
		Closed:    t.Closed,
	}

	e.AllowList = make([]string, 0, len(t.AllowList))
	for _, id := range t.AllowList {
		e.AllowList = append(e.AllowList, id.String())
	}

	return &TeamCreated{
		Base: newBase(ctx, TeamCreatedEvent),
		e:    e,
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

func (t TeamUpdated) TeamUpdated() *eventspb.TeamUpdated {
	return &t.e
}

func NewTeamUpdatedEvent(ctx context.Context, t *types.Team) *TeamUpdated {
	e := eventspb.TeamUpdated{
		TeamId:    string(t.ID),
		Name:      t.Name,
		TeamUrl:   ptr.From(t.TeamURL),
		AvatarUrl: ptr.From(t.AvatarURL),
		Closed:    t.Closed,
	}

	e.AllowList = make([]string, 0, len(t.AllowList))
	for _, id := range t.AllowList {
		e.AllowList = append(e.AllowList, id.String())
	}

	return &TeamUpdated{
		Base: newBase(ctx, TeamUpdatedEvent),
		e:    e,
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

func (t RefereeSwitchedTeam) RefereeSwitchedTeam() *eventspb.RefereeSwitchedTeam {
	return &t.e
}

func NewRefereeSwitchedTeamEvent(ctx context.Context, from, to types.TeamID, membership *types.Membership) *RefereeSwitchedTeam {
	return &RefereeSwitchedTeam{
		Base: newBase(ctx, RefereeSwitchedTeamEvent),
		e: eventspb.RefereeSwitchedTeam{
			FromTeamId: string(from),
			ToTeamId:   string(to),
			Referee:    string(membership.PartyID),
			SwitchedAt: membership.JoinedAt.UnixNano(),
			AtEpoch:    membership.StartedAtEpoch,
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

func (t RefereeJoinedTeam) RefereeJoinedTeam() *eventspb.RefereeJoinedTeam {
	return &t.e
}

func NewRefereeJoinedTeamEvent(ctx context.Context, teamID types.TeamID, membership *types.Membership) *RefereeJoinedTeam {
	return &RefereeJoinedTeam{
		Base: newBase(ctx, RefereeJoinedTeamEvent),
		e: eventspb.RefereeJoinedTeam{
			TeamId:   string(teamID),
			Referee:  string(membership.PartyID),
			JoinedAt: membership.JoinedAt.UnixNano(),
			AtEpoch:  membership.StartedAtEpoch,
		},
	}
}

func RefereeJoinedTeamEventFromStream(ctx context.Context, be *eventspb.BusEvent) *RefereeJoinedTeam {
	return &RefereeJoinedTeam{
		Base: newBaseFromBusEvent(ctx, RefereeJoinedTeamEvent, be),
		e:    *be.GetRefereeJoinedTeam(),
	}
}

type TeamsStatsUpdated struct {
	*Base
	e eventspb.TeamsStatsUpdated
}

func (t TeamsStatsUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_TeamsStatsUpdated{
		TeamsStatsUpdated: &t.e,
	}

	return busEvent
}

func (t TeamsStatsUpdated) TeamsStatsUpdated() *eventspb.TeamsStatsUpdated {
	return &t.e
}

func NewTeamsStatsUpdatedEvent(ctx context.Context, seq uint64, rawTeamsStats map[string]map[string]*num.Uint) *TeamsStatsUpdated {
	teamsStats := make([]*eventspb.TeamStats, 0, len(rawTeamsStats))
	for teamID, rawTeamStats := range rawTeamsStats {
		ts := make([]*eventspb.TeamMemberStats, 0, len(rawTeamStats))
		for partyID, notionalVolume := range rawTeamStats {
			ts = append(ts, &eventspb.TeamMemberStats{
				PartyId:        partyID,
				NotionalVolume: notionalVolume.String(),
			})
		}

		slices.SortStableFunc(ts, func(a, b *eventspb.TeamMemberStats) int {
			return strings.Compare(a.PartyId, b.PartyId)
		})

		teamsStats = append(teamsStats, &eventspb.TeamStats{
			TeamId:       teamID,
			MembersStats: ts,
		})
	}

	slices.SortStableFunc(teamsStats, func(a, b *eventspb.TeamStats) int {
		return strings.Compare(a.TeamId, b.TeamId)
	})

	return &TeamsStatsUpdated{
		Base: newBase(ctx, TeamsStatsUpdatedEvent),
		e: eventspb.TeamsStatsUpdated{
			Stats:   teamsStats,
			AtEpoch: seq,
		},
	}
}

func TeamsStatsUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TeamsStatsUpdated {
	return &TeamsStatsUpdated{
		Base: newBaseFromBusEvent(ctx, TeamsStatsUpdatedEvent, be),
		e:    *be.GetTeamsStatsUpdated(),
	}
}
