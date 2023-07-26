package teams

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

var ErrReferrerCannotJoinAnotherTeam = errors.New("a referrer cannot join another team")

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/teams EpochEngine

type EpochEngine interface {
	NotifyOnEpoch(func(context.Context, types.Epoch), func(context.Context, types.Epoch))
}

type Engine struct {
	// teams tracks all teams by team ID.
	teams map[types.TeamID]*types.Team

	// allTeamMembers tracks all the parties that belongs to a team, referrers and
	// referees, by their current team ID.
	allTeamMembers map[types.PartyID]types.TeamID

	// membersToMove tracks all the parties that switch teams. The switch only
	// happens by the end of the epoch.
	membersToMove map[types.PartyID]teamMove
}

func (e *Engine) CreateTeam(referrer types.PartyID, name string, teamURL string, avatarURL string, enableRewards bool) (types.TeamID, error) {
	newTeamID := e.newUniqueTeamID()

	if _, alreadyMember := e.allTeamMembers[referrer]; alreadyMember {
		return "", ErrPartyAlreadyBelongsToTeam(referrer)
	}

	e.teams[newTeamID] = &types.Team{
		ID:            newTeamID,
		Referrer:      referrer,
		Name:          name,
		TeamURL:       teamURL,
		AvatarURL:     avatarURL,
		EnableRewards: enableRewards,
	}

	e.allTeamMembers[referrer] = newTeamID

	return newTeamID, nil
}

func (e *Engine) UpdateTeam(id types.TeamID, name string, teamURL string, avatarURL string, enableRewards bool) error {
	_, exists := e.teams[id]
	if !exists {
		return ErrNoTeamMatchesID(id)
	}

	e.teams[id].Name = name
	e.teams[id].TeamURL = teamURL
	e.teams[id].AvatarURL = avatarURL
	e.teams[id].EnableRewards = enableRewards

	return nil
}

func (e *Engine) JoinTeam(id types.TeamID, party types.PartyID) error {
	for _, team := range e.teams {
		if team.Referrer == party {
			return ErrReferrerCannotJoinAnotherTeam
		}
	}

	teamToJoin, exists := e.teams[id]
	if !exists {
		return ErrNoTeamMatchesID(id)
	}

	teamJoined, alreadyMember := e.allTeamMembers[party]
	if alreadyMember {
		// This party is already member of a team, it will be moved at the end
		// of epoch.
		e.membersToMove[party] = teamMove{
			fromTeam: teamJoined,
			toTeam:   teamToJoin.ID,
		}
		return nil
	}

	// The party does not belong to a team, so he joins right away.
	teamToJoin.AddReferee(party)

	e.allTeamMembers[party] = teamToJoin.ID

	return nil
}

func (e *Engine) OnEpoch(_ context.Context, ep types.Epoch) {
	switch ep.Action {
	case proto.EpochAction_EPOCH_ACTION_END:
		e.moveMembers()
	}
}

func (e *Engine) OnEpochRestore(_ context.Context, _ types.Epoch) {}

func (e *Engine) moveMembers() {
	for party, move := range e.membersToMove {
		e.teams[move.fromTeam].RemoveReferee(party)
		e.teams[move.toTeam].AddReferee(party)
	}
}

func (e *Engine) newUniqueTeamID() types.TeamID {
	for {
		id := types.NewTeamID()
		if _, exists := e.teams[id]; !exists {
			return id
		}
	}
}

func (e *Engine) IsTeamMember(party types.PartyID) bool {
	_, isMember := e.allTeamMembers[party]
	return isMember
}

func NewEngine(epochEngine EpochEngine) *Engine {
	engine := &Engine{
		teams:          map[types.TeamID]*types.Team{},
		allTeamMembers: map[types.PartyID]types.TeamID{},
		membersToMove:  map[types.PartyID]teamMove{},
	}

	epochEngine.NotifyOnEpoch(engine.OnEpoch, engine.OnEpochRestore)

	return engine
}

type teamMove struct {
	fromTeam types.TeamID
	toTeam   types.TeamID
}
