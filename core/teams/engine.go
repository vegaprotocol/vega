package teams

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

var ErrReferrerCannotJoinAnotherTeam = errors.New("a referrer cannot join another team")

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/teams EpochEngine,Broker

type EpochEngine interface {
	NotifyOnEpoch(func(context.Context, types.Epoch), func(context.Context, types.Epoch))
}

type Broker interface {
	Send(event events.Event)
}

type Engine struct {
	broker Broker

	// teams tracks all teams by team ID.
	teams map[types.TeamID]*types.Team

	// allTeamMembers tracks all the parties that belongs to a team, referrers and
	// referees, by their current team ID.
	allTeamMembers map[types.PartyID]types.TeamID
	// membersToMove tracks all the parties that switch teams. The switch only
	// happens by the end of the epoch.
	membersToMove map[types.PartyID]teamMove
}

func (e *Engine) CreateTeam(ctx context.Context, referrer types.PartyID, name string, teamURL string, avatarURL string) (types.TeamID, error) {
	newTeamID := e.newUniqueTeamID()

	if _, alreadyMember := e.allTeamMembers[referrer]; alreadyMember {
		return "", ErrPartyAlreadyBelongsToTeam(referrer)
	}

	teamToAdd := &types.Team{
		ID:        newTeamID,
		Referrer:  referrer,
		Name:      name,
		TeamURL:   teamURL,
		AvatarURL: avatarURL,
	}

	e.teams[newTeamID] = teamToAdd

	e.allTeamMembers[referrer] = newTeamID

	e.notifyTeamCreated(ctx, teamToAdd)

	return newTeamID, nil
}

func (e *Engine) UpdateTeam(ctx context.Context, id types.TeamID, name string, teamURL string, avatarURL string) error {
	teamsToUpdate, exists := e.teams[id]
	if !exists {
		return ErrNoTeamMatchesID(id)
	}

	teamsToUpdate.Name = name
	teamsToUpdate.TeamURL = teamURL
	teamsToUpdate.AvatarURL = avatarURL

	e.notifyTeamUpdated(ctx, teamsToUpdate)

	return nil
}

func (e *Engine) JoinTeam(ctx context.Context, id types.TeamID, party types.PartyID) error {
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

	e.notifyRefereeJoinedTeam(ctx, teamToJoin, party)

	return nil
}

func (e *Engine) IsTeamMember(party types.PartyID) bool {
	_, isMember := e.allTeamMembers[party]
	return isMember
}

func (e *Engine) OnEpoch(ctx context.Context, ep types.Epoch) {
	switch ep.Action {
	case proto.EpochAction_EPOCH_ACTION_END:
		e.moveMembers(ctx)
	}
}

func (e *Engine) OnEpochRestore(_ context.Context, _ types.Epoch) {}

func (e *Engine) moveMembers(ctx context.Context) {
	for party, move := range e.membersToMove {
		e.teams[move.fromTeam].RemoveReferee(party)
		e.teams[move.toTeam].AddReferee(party)
		e.allTeamMembers[party] = move.toTeam
		e.notifyRefereeSwitchedTeam(ctx, move, party)
	}

	e.membersToMove = map[types.PartyID]teamMove{}
}

func (e *Engine) notifyTeamCreated(ctx context.Context, teamToAdd *types.Team) {
	e.broker.Send(events.NewTeamCreatedEvent(ctx, teamToAdd))
}

func (e *Engine) notifyTeamUpdated(ctx context.Context, teamsToUpdate *types.Team) {
	e.broker.Send(events.NewTeamUpdatedEvent(ctx, teamsToUpdate))
}

func (e *Engine) notifyRefereeSwitchedTeam(ctx context.Context, move teamMove, party types.PartyID) {
	e.broker.Send(events.NewRefereeSwitchedTeamEvent(ctx, move.fromTeam, move.toTeam, party))
}

func (e *Engine) notifyRefereeJoinedTeam(ctx context.Context, teamID *types.Team, party types.PartyID) {
	e.broker.Send(events.NewRefereeJoinedTeamEvent(ctx, teamID.ID, party))
}

func (e *Engine) newUniqueTeamID() types.TeamID {
	for {
		id := types.NewTeamID()
		if _, exists := e.teams[id]; !exists {
			return id
		}
	}
}

func NewEngine(epochEngine EpochEngine, broker Broker) *Engine {
	engine := &Engine{
		broker: broker,

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
