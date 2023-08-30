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
	"errors"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"golang.org/x/exp/slices"
)

type Engine struct {
	broker      Broker
	timeService TimeService

	currentEpoch uint64

	// minStakedVegaTokens limits referral code generation to parties staking at
	// least this number of tokens.
	minStakedVegaTokens num.Decimal

	// teams tracks all teams by team ID.
	teams map[types.TeamID]*types.Team

	// allTeamMembers maps a party to the team they are members of.
	allTeamMembers map[types.PartyID]types.TeamID

	// teamSwitches tracks all the parties that switch teams. The switch only
	// happens by the end of the epoch.
	teamSwitches map[types.PartyID]teamSwitch
}

func (e *Engine) OnReferralProgramMinStakedVegaTokensUpdate(_ context.Context, value num.Decimal) error {
	e.minStakedVegaTokens = value
	return nil
}

func (e *Engine) TeamExists(team types.TeamID) bool {
	_, ok := e.teams[team]
	return ok
}

func (e *Engine) CreateTeam(ctx context.Context, referrer types.PartyID, deterministicTeamID types.TeamID, params *commandspb.CreateReferralSet_Team) error {
	if err := e.ensureUniqueTeamID(deterministicTeamID); err != nil {
		return err
	}

	if err := e.ensureUniqueTeamName(params.Name); err != nil {
		return err
	}

	if _, alreadyMember := e.allTeamMembers[referrer]; alreadyMember {
		return ErrPartyAlreadyBelongsToTeam(referrer)
	}

	if len(params.Name) <= 0 {
		return errors.New("missing required team name parameter")
	}

	now := e.timeService.GetTimeNow()

	teamToAdd := &types.Team{
		ID: deterministicTeamID,
		Referrer: &types.Membership{
			PartyID:        referrer,
			JoinedAt:       now,
			StartedAtEpoch: e.currentEpoch,
		},
		Name:      params.Name,
		TeamURL:   ptr.UnBox(params.TeamUrl),
		AvatarURL: ptr.UnBox(params.AvatarUrl),
		CreatedAt: now,
		Closed:    params.Closed,
	}

	e.teams[deterministicTeamID] = teamToAdd

	e.allTeamMembers[referrer] = deterministicTeamID

	e.notifyTeamCreated(ctx, teamToAdd)

	return nil
}

func (e *Engine) UpdateTeam(ctx context.Context, referrer types.PartyID, teamID types.TeamID, params *commandspb.UpdateReferralSet_Team) error {
	teamsToUpdate, exists := e.teams[teamID]
	if !exists {
		return ErrNoTeamMatchesID(teamID)
	}

	if teamsToUpdate.Referrer.PartyID != referrer {
		return ErrOnlyReferrerCanUpdateTeam
	}

	// can't update if empty and nil as it's a mandatory field
	if params.Name != nil && len(*params.Name) > 0 {
		teamsToUpdate.Name = ptr.UnBox(params.Name)
	}

	// those apply change if not nil only?
	// to be sure to not erase things by mistake?
	if params.TeamUrl != nil {
		teamsToUpdate.TeamURL = ptr.UnBox(params.TeamUrl)
	}

	if params.AvatarUrl != nil {
		teamsToUpdate.AvatarURL = ptr.UnBox(params.AvatarUrl)
	}

	if params.Closed != nil {
		teamsToUpdate.Closed = ptr.UnBox(params.Closed)
	}

	e.notifyTeamUpdated(ctx, teamsToUpdate)

	return nil
}

func (e *Engine) JoinTeam(ctx context.Context, referee types.PartyID, params *commandspb.ApplyReferralCode) error {
	for _, team := range e.teams {
		if team.Referrer.PartyID == referee {
			return ErrReferrerCannotJoinAnotherTeam
		}
	}

	teamID := types.TeamID(params.Id)

	teamToJoin, exists := e.teams[teamID]
	if !exists {
		return ErrNoTeamMatchesID(teamID)
	}

	if teamToJoin.Closed {
		return ErrTeamIsClosed(teamID)
	}

	teamJoined, alreadyMember := e.allTeamMembers[referee]
	if alreadyMember {
		// This party is already member of a team, it will be moved at the end
		// of epoch.
		e.teamSwitches[referee] = teamSwitch{
			fromTeam: teamJoined,
			toTeam:   teamToJoin.ID,
		}
		return nil
	}

	// The party does not belong to a team, so he joins right away.
	membership := &types.Membership{
		PartyID:        referee,
		JoinedAt:       e.timeService.GetTimeNow(),
		StartedAtEpoch: e.currentEpoch,
	}
	teamToJoin.Referees = append(teamToJoin.Referees, membership)

	e.allTeamMembers[referee] = teamToJoin.ID

	e.notifyRefereeJoinedTeam(ctx, teamToJoin, membership)

	return nil
}

func (e *Engine) GetAllPartiesInTeams(minEpochsInTeam uint64) []string {
	parties := make([]string, 0, len(e.allTeamMembers))

	for t := range e.teams {
		members := e.GetTeamMembers(string(t), minEpochsInTeam)
		if len(members) > 0 {
			parties = append(parties, members...)
		}
	}
	sort.Strings(parties)
	return parties
}

func (e *Engine) GetTeamMembers(team string, minEpochsInTeam uint64) []string {
	t := e.teams[(types.TeamID(team))]
	teamMembers := make([]string, 0, len(t.Referees)+1)
	for _, m := range t.Referees {
		if m.StartedAtEpoch-e.currentEpoch >= minEpochsInTeam {
			teamMembers = append(teamMembers, string(m.PartyID))
		}
	}
	if t.Referrer.StartedAtEpoch-e.currentEpoch >= minEpochsInTeam {
		teamMembers = append(teamMembers, string(t.Referrer.PartyID))
	}
	sort.Strings(teamMembers)
	return teamMembers
}

func (e *Engine) IsTeamMember(party types.PartyID) bool {
	_, isMember := e.allTeamMembers[party]
	return isMember
}

func (e *Engine) OnEpoch(ctx context.Context, ep types.Epoch) {
	if ep.Action == vegapb.EpochAction_EPOCH_ACTION_START {
		e.currentEpoch = ep.Seq
		e.moveMembers(ctx, ep.StartTime, ep.Seq)
	}
}

func (e *Engine) OnEpochRestore(_ context.Context, ep types.Epoch) {
	if ep.Action == vegapb.EpochAction_EPOCH_ACTION_START {
		e.currentEpoch = ep.Seq
	}
}

// moveMembers ensures members are moved in a deterministic order.
func (e *Engine) moveMembers(ctx context.Context, startEpochTime time.Time, epoch uint64) {
	sortedPartyID := make([]types.PartyID, 0, len(e.teamSwitches))
	for partyID := range e.teamSwitches {
		sortedPartyID = append(sortedPartyID, partyID)
	}
	slices.SortStableFunc(sortedPartyID, func(a, b types.PartyID) bool {
		return a < b
	})

	for _, partyID := range sortedPartyID {
		move := e.teamSwitches[partyID]
		e.teams[move.fromTeam].RemoveReferee(partyID)
		membership := &types.Membership{
			PartyID:        partyID,
			JoinedAt:       startEpochTime,
			StartedAtEpoch: epoch,
		}
		toTeam := e.teams[move.toTeam]
		toTeam.Referees = append(toTeam.Referees, membership)

		e.allTeamMembers[partyID] = toTeam.ID
		e.notifyRefereeSwitchedTeam(ctx, move, membership)
	}

	e.teamSwitches = map[types.PartyID]teamSwitch{}
}

func (e *Engine) notifyTeamCreated(ctx context.Context, teamToAdd *types.Team) {
	e.broker.Send(events.NewTeamCreatedEvent(ctx, teamToAdd))
}

func (e *Engine) notifyTeamUpdated(ctx context.Context, teamsToUpdate *types.Team) {
	e.broker.Send(events.NewTeamUpdatedEvent(ctx, teamsToUpdate))
}

func (e *Engine) notifyRefereeSwitchedTeam(ctx context.Context, move teamSwitch, membership *types.Membership) {
	e.broker.Send(events.NewRefereeSwitchedTeamEvent(ctx, move.fromTeam, move.toTeam, membership))
}

func (e *Engine) notifyRefereeJoinedTeam(ctx context.Context, teamID *types.Team, membership *types.Membership) {
	e.broker.Send(events.NewRefereeJoinedTeamEvent(ctx, teamID.ID, membership))
}

func (e *Engine) ensureUniqueTeamID(deterministicTeamID types.TeamID) error {
	if _, exists := e.teams[deterministicTeamID]; exists {
		return ErrComputedTeamIDIsAlreadyInUse
	}
	return nil
}

func (e *Engine) ensureUniqueTeamName(name string) error {
	for _, team := range e.teams {
		if team.Name == name {
			return ErrTeamNameIsAlreadyInUse
		}
	}

	return nil
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
				PartyID:        refereeID,
				JoinedAt:       time.Unix(0, refereeSnapshot.JoinedAt),
				StartedAtEpoch: refereeSnapshot.StartedAtEpoch,
			})
		}

		e.teams[teamID] = &types.Team{
			ID: teamID,
			Referrer: &types.Membership{
				PartyID:        referrerID,
				JoinedAt:       time.Unix(0, teamSnapshot.Referrer.JoinedAt),
				StartedAtEpoch: teamSnapshot.Referrer.StartedAtEpoch,
			},
			Referees:  referees,
			Name:      teamSnapshot.Name,
			TeamURL:   teamSnapshot.TeamUrl,
			AvatarURL: teamSnapshot.AvatarUrl,
			CreatedAt: time.Unix(0, teamSnapshot.CreatedAt),
			Closed:    teamSnapshot.Closed,
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
