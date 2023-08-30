// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package teams

import (
	"context"
	"fmt"

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
		teamsSnapshot = append(teamsSnapshot, teamSnapshot)
	}

	slices.SortStableFunc(teamsSnapshot, func(a, b *snapshotpb.Team) bool {
		return a.Id < b.Id
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

	slices.SortStableFunc(teamSwitchesSnapshot, func(a, b *snapshotpb.TeamSwitch) bool {
		return a.PartyId < b.PartyId
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

func NewSnapshottedEngine(epochEngine EpochEngine, broker Broker, timeSvc TimeService) *SnapshottedEngine {
	se := &SnapshottedEngine{
		Engine:  NewEngine(epochEngine, broker, timeSvc),
		pl:      types.Payload{},
		stopped: false,
	}

	se.buildHashKeys()

	return se
}
