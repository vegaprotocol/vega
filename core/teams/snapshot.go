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

package teams

import (
	"context"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"golang.org/x/exp/slices"
)

type SnapshottedEngine struct {
	*Engine

	pl types.Payload

	stopped bool

	// Keys need to be computed when the engine is instantiated as they are dynamic.
	hashKeys        []string
	teamsKey        string
	teamSwitchesKey string
}

func (e *SnapshottedEngine) Namespace() types.SnapshotNamespace {
	return types.TeamsSnapshot
}

func (e *SnapshottedEngine) Keys() []string {
	return e.hashKeys
}

func (e *SnapshottedEngine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise(k)
	return state, nil, err
}

func (e *SnapshottedEngine) LoadState(_ context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch data := p.Data.(type) {
	case *types.PayloadTeams:
		e.Engine.loadTeamsFromSnapshot(data.Teams)
		return nil, nil
	case *types.PayloadTeamSwitches:
		e.Engine.loadTeamSwitchesFromSnapshot(data.TeamSwitches)
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *SnapshottedEngine) Stopped() bool {
	return e.stopped
}

func (e *SnapshottedEngine) StopSnapshots() {
	e.stopped = true
}

func (e *SnapshottedEngine) serialise(k string) ([]byte, error) {
	if e.stopped {
		return nil, nil
	}

	switch k {
	case e.teamsKey:
		return e.serialiseTeams()
	case e.teamSwitchesKey:
		return e.serialiseTeamSwitches()
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *SnapshottedEngine) serialiseTeams() ([]byte, error) {
	teams := e.Engine.teams
	teamsSnapshot := make([]*snapshotpb.Team, 0, len(teams))
	for _, team := range teams {
		refereesSnapshot := make([]*snapshotpb.Membership, 0, len(team.Referees))
		for _, referee := range team.Referees {
			refereesSnapshot = append(refereesSnapshot, &snapshotpb.Membership{
				PartyId:        string(referee.PartyID),
				JoinedAt:       referee.JoinedAt.UnixNano(),
				StartedAtEpoch: referee.StartedAtEpoch,
			})
		}

		teamSnapshot := &snapshotpb.Team{
			Id: string(team.ID),
			Referrer: &snapshotpb.Membership{
				PartyId:        string(team.Referrer.PartyID),
				JoinedAt:       team.Referrer.JoinedAt.UnixNano(),
				StartedAtEpoch: team.Referrer.StartedAtEpoch,
			},
			Referees:  refereesSnapshot,
			Name:      team.Name,
			TeamUrl:   team.TeamURL,
			AvatarUrl: team.AvatarURL,
			CreatedAt: team.CreatedAt.UnixNano(),
			Closed:    team.Closed,
		}

		if len(team.AllowList) > 0 {
			teamSnapshot.AllowList = make([]string, 0, len(team.AllowList))
			for _, partyID := range team.AllowList {
				teamSnapshot.AllowList = append(teamSnapshot.AllowList, partyID.String())
			}
		}

		teamsSnapshot = append(teamsSnapshot, teamSnapshot)
	}

	slices.SortStableFunc(teamsSnapshot, func(a, b *snapshotpb.Team) int {
		return strings.Compare(a.Id, b.Id)
	})

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_Teams{
			Teams: &snapshotpb.Teams{
				Teams: teamsSnapshot,
			},
		},
	}

	serialisedTeams, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not serialize teams payload: %w", err)
	}

	return serialisedTeams, nil
}

func (e *SnapshottedEngine) serialiseTeamSwitches() ([]byte, error) {
	teamSwitches := e.Engine.teamSwitches
	teamSwitchesSnapshot := make([]*snapshotpb.TeamSwitch, 0, len(teamSwitches))

	for partyID, teamSwitch := range teamSwitches {
		teamSwitchSnapshot := &snapshotpb.TeamSwitch{
			FromTeamId: string(teamSwitch.fromTeam),
			ToTeamId:   string(teamSwitch.toTeam),
			PartyId:    string(partyID),
		}
		teamSwitchesSnapshot = append(teamSwitchesSnapshot, teamSwitchSnapshot)
	}

	slices.SortStableFunc(teamSwitchesSnapshot, func(a, b *snapshotpb.TeamSwitch) int {
		return strings.Compare(a.PartyId, b.PartyId)
	})

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_TeamSwitches{
			TeamSwitches: &snapshotpb.TeamSwitches{
				TeamSwitches: teamSwitchesSnapshot,
			},
		},
	}

	serialisedTeamSwitches, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not serialize team switches payload: %w", err)
	}

	return serialisedTeamSwitches, nil
}

func (e *SnapshottedEngine) buildHashKeys() {
	e.teamsKey = (&types.PayloadTeams{}).Key()
	e.teamSwitchesKey = (&types.PayloadTeamSwitches{}).Key()

	e.hashKeys = append([]string{}, e.teamsKey, e.teamSwitchesKey)
}

func NewSnapshottedEngine(broker Broker, timeSvc TimeService) *SnapshottedEngine {
	se := &SnapshottedEngine{
		Engine:  NewEngine(broker, timeSvc),
		pl:      types.Payload{},
		stopped: false,
	}

	se.buildHashKeys()

	return se
}
