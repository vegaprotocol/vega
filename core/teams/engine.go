package teams

import (
	"context"
	"errors"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"golang.org/x/exp/slices"
)

var ErrReferrerCannotJoinAnotherTeam = errors.New("a referrer cannot join another team")

type Engine struct {
	broker      Broker
	timeService TimeService

	// teams tracks all teams by team ID.
	teams map[types.TeamID]*types.Team

	// allTeamMembers tracks all the parties that belongs to a team, referrers and
	// referees, by their current team ID.
	allTeamMembers map[types.PartyID]types.TeamID

	// teamSwitches tracks all the parties that switch teams. The switch only
	// happens by the end of the epoch.
	teamSwitches map[types.PartyID]teamSwitch
}

func (e *Engine) CreateTeam(ctx context.Context, referrer types.PartyID, name string, teamURL string, avatarURL string) (types.TeamID, error) {
	newTeamID := e.newUniqueTeamID()

	if _, alreadyMember := e.allTeamMembers[referrer]; alreadyMember {
		return "", ErrPartyAlreadyBelongsToTeam(referrer)
	}

	now := e.timeService.GetTimeNow()

	teamToAdd := &types.Team{
		ID: newTeamID,
		Referrer: &types.Membership{
			PartyID:  referrer,
			JoinedAt: now,
		},
		Name:      name,
		TeamURL:   teamURL,
		AvatarURL: avatarURL,
		CreatedAt: now,
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

func (e *Engine) JoinTeam(ctx context.Context, teamID types.TeamID, partyID types.PartyID) error {
	for _, team := range e.teams {
		if team.Referrer.PartyID == partyID {
			return ErrReferrerCannotJoinAnotherTeam
		}
	}

	teamToJoin, exists := e.teams[teamID]
	if !exists {
		return ErrNoTeamMatchesID(teamID)
	}

	teamJoined, alreadyMember := e.allTeamMembers[partyID]
	if alreadyMember {
		// This party is already member of a team, it will be moved at the end
		// of epoch.
		e.teamSwitches[partyID] = teamSwitch{
			fromTeam: teamJoined,
			toTeam:   teamToJoin.ID,
		}
		return nil
	}

	// The party does not belong to a team, so he joins right away.
	teamToJoin.AddReferee(partyID, e.timeService.GetTimeNow())

	e.allTeamMembers[partyID] = teamToJoin.ID

	e.notifyRefereeJoinedTeam(ctx, teamToJoin, partyID, time.Now())

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

// moveMembers ensures members are moved in a deterministic order.
func (e *Engine) moveMembers(ctx context.Context) {
	sortedPartyID := make([]types.PartyID, 0, len(e.teamSwitches))
	for partyID := range e.teamSwitches {
		sortedPartyID = append(sortedPartyID, partyID)
	}
	slices.SortStableFunc(sortedPartyID, func(a, b types.PartyID) bool {
		return a < b
	})

	now := e.timeService.GetTimeNow()

	for _, partyID := range sortedPartyID {
		move := e.teamSwitches[partyID]
		e.teams[move.fromTeam].RemoveReferee(partyID)
		e.teams[move.toTeam].AddReferee(partyID, now)
		e.allTeamMembers[partyID] = move.toTeam
		e.notifyRefereeSwitchedTeam(ctx, move, partyID, now)
	}

	e.teamSwitches = map[types.PartyID]teamSwitch{}
}

func (e *Engine) notifyTeamCreated(ctx context.Context, teamToAdd *types.Team) {
	e.broker.Send(events.NewTeamCreatedEvent(ctx, teamToAdd))
}

func (e *Engine) notifyTeamUpdated(ctx context.Context, teamsToUpdate *types.Team) {
	e.broker.Send(events.NewTeamUpdatedEvent(ctx, teamsToUpdate))
}

func (e *Engine) notifyRefereeSwitchedTeam(ctx context.Context, move teamSwitch, party types.PartyID, switchedAt time.Time) {
	e.broker.Send(events.NewRefereeSwitchedTeamEvent(ctx, move.fromTeam, move.toTeam, party, switchedAt))
}

func (e *Engine) notifyRefereeJoinedTeam(ctx context.Context, teamID *types.Team, party types.PartyID, joinedAt time.Time) {
	e.broker.Send(events.NewRefereeJoinedTeamEvent(ctx, teamID.ID, party, joinedAt))
}

func (e *Engine) newUniqueTeamID() types.TeamID {
	for {
		id := types.NewTeamID()
		if _, exists := e.teams[id]; !exists {
			return id
		}
	}
}

func (e *Engine) loadTeamsFromSnapshot(teamsSnapshot []*snapshotpb.Team) {
	for _, teamSnapshot := range teamsSnapshot {
		teamID := types.TeamID(teamSnapshot.Id)

		referrerID := types.PartyID(teamSnapshot.Referrer.PartyId)
		e.allTeamMembers[referrerID] = teamID

		referees := make([]*types.Membership, 0, len(teamSnapshot.Referees))
		for _, refereeSnapshot := range teamSnapshot.Referees {
			refereeID := types.PartyID(refereeSnapshot.PartyId)
			e.allTeamMembers[refereeID] = teamID
			referees = append(referees, &types.Membership{
				PartyID:  refereeID,
				JoinedAt: time.Unix(0, refereeSnapshot.JoinedAt),
			})
		}

		e.teams[teamID] = &types.Team{
			ID: teamID,
			Referrer: &types.Membership{
				PartyID:  referrerID,
				JoinedAt: time.Unix(0, teamSnapshot.Referrer.JoinedAt),
			},
			Referees:  referees,
			Name:      teamSnapshot.Name,
			TeamURL:   teamSnapshot.TeamUrl,
			AvatarURL: teamSnapshot.AvatarUrl,
			CreatedAt: time.Unix(0, teamSnapshot.CreatedAt),
		}
	}
}

func (e *Engine) loadTeamSwitchesFromSnapshot(teamSwitchesSnapshot []*snapshotpb.TeamSwitch) {
	for _, teamSwitchSnapshot := range teamSwitchesSnapshot {
		partyID := types.PartyID(teamSwitchSnapshot.PartyId)
		e.teamSwitches[partyID] = teamSwitch{
			fromTeam: types.TeamID(teamSwitchSnapshot.FromTeamId),
			toTeam:   types.TeamID(teamSwitchSnapshot.ToTeamId),
		}
	}
}

func NewEngine(epochEngine EpochEngine, broker Broker, timeSvc TimeService) *Engine {
	engine := &Engine{
		broker:      broker,
		timeService: timeSvc,

		teams:          map[types.TeamID]*types.Team{},
		allTeamMembers: map[types.PartyID]types.TeamID{},
		teamSwitches:   map[types.PartyID]teamSwitch{},
	}

	epochEngine.NotifyOnEpoch(engine.OnEpoch, engine.OnEpochRestore)

	return engine
}

type teamSwitch struct {
	fromTeam types.TeamID
	toTeam   types.TeamID
}
