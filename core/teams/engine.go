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
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	proto "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"golang.org/x/exp/slices"
)

type Engine struct {
	broker      Broker
	timeService TimeService

	// minStakedVegaTokens limits referral code generation to parties staking at
	// least this number of tokens.
	minStakedVegaTokens num.Decimal

	// teams tracks all teams by team ID.
	teams map[types.TeamID]*types.Team

	// allTeamMembers tracks all the parties that belongs to a team, referrers and
	// referees, by their current team ID.
	allTeamMembers map[types.PartyID]types.TeamID

	// teamSwitches tracks all the parties that switch teams. The switch only
	// happens by the end of the epoch.
	teamSwitches map[types.PartyID]teamSwitch
}

func (e *Engine) OnReferralProgramMinStakedVegaTokensUpdate(_ context.Context, value num.Decimal) error {
	e.minStakedVegaTokens = value
	return nil
}

func (e *Engine) CreateTeam(ctx context.Context, referrer types.PartyID, deterministicTeamID types.TeamID, params *commandspb.CreateTeam) error {
	if err := e.ensureUniqueTeamID(deterministicTeamID); err != nil {
		return err
	}

	if _, alreadyMember := e.allTeamMembers[referrer]; alreadyMember {
		return ErrPartyAlreadyBelongsToTeam(referrer)
	}

	now := e.timeService.GetTimeNow()

	teamToAdd := &types.Team{
		ID: deterministicTeamID,
		Referrer: &types.Membership{
			PartyID:       referrer,
			JoinedAt:      now,
			NumberOfEpoch: 0,
		},
		Name:      ptr.UnBox(params.Name),
		TeamURL:   ptr.UnBox(params.TeamUrl),
		AvatarURL: ptr.UnBox(params.AvatarUrl),
		CreatedAt: now,
	}

	e.teams[deterministicTeamID] = teamToAdd

	e.allTeamMembers[referrer] = deterministicTeamID

	e.notifyTeamCreated(ctx, teamToAdd)

	return nil
}

func (e *Engine) UpdateTeam(ctx context.Context, referrer types.PartyID, params *commandspb.UpdateTeam) error {
	teamID := types.TeamID(params.TeamId)

	teamsToUpdate, exists := e.teams[teamID]
	if !exists {
		return ErrNoTeamMatchesID(teamID)
	}

	if teamsToUpdate.Referrer.PartyID != referrer {
		return ErrOnlyReferrerCanUpdateTeam
	}

	teamsToUpdate.Name = ptr.UnBox(params.Name)
	teamsToUpdate.TeamURL = ptr.UnBox(params.TeamUrl)
	teamsToUpdate.AvatarURL = ptr.UnBox(params.AvatarUrl)

	e.notifyTeamUpdated(ctx, teamsToUpdate)

	return nil
}

func (e *Engine) JoinTeam(ctx context.Context, referee types.PartyID, params *commandspb.JoinTeam) error {
	for _, team := range e.teams {
		if team.Referrer.PartyID == referee {
			return ErrReferrerCannotJoinAnotherTeam
		}
	}

	teamID := types.TeamID(params.TeamId)

	teamToJoin, exists := e.teams[teamID]
	if !exists {
		return ErrNoTeamMatchesID(teamID)
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
	teamToJoin.AddReferee(referee, e.timeService.GetTimeNow())

	e.allTeamMembers[referee] = teamToJoin.ID

	e.notifyRefereeJoinedTeam(ctx, teamToJoin, referee, time.Now())

	return nil
}

func (e *Engine) NumberOfEpochInTeamForParty(party types.PartyID) uint64 {
	teamID, isMember := e.allTeamMembers[party]
	if !isMember {
		return 0
	}

	team := e.teams[teamID]
	if team.Referrer.PartyID == party {
		return team.Referrer.NumberOfEpoch
	}

	for _, referee := range team.Referees {
		if referee.PartyID == party {
			return referee.NumberOfEpoch
		}
	}

	// This should never happen if the state is kept consistent in the engine between
	// fields `allTeamMembers` and `teams`. If it happens, this is a severe
	// programming error.
	panic(fmt.Sprintf("party %q is registered as a member of the team %q but the team does not reference his membership", party, teamID))
}

func (e *Engine) IsTeamMember(party types.PartyID) bool {
	_, isMember := e.allTeamMembers[party]
	return isMember
}

func (e *Engine) OnEpoch(ctx context.Context, ep types.Epoch) {
	switch ep.Action {
	case proto.EpochAction_EPOCH_ACTION_START:
		e.moveMembers(ctx, ep.StartTime)
	case proto.EpochAction_EPOCH_ACTION_END:
		e.updateMembershipStats()
	}
}

func (e *Engine) OnEpochRestore(_ context.Context, _ types.Epoch) {}

// moveMembers ensures members are moved in a deterministic order.
func (e *Engine) moveMembers(ctx context.Context, startEpochTime time.Time) {
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
		e.teams[move.toTeam].AddReferee(partyID, startEpochTime)
		e.allTeamMembers[partyID] = move.toTeam
		e.notifyRefereeSwitchedTeam(ctx, move, partyID, startEpochTime)
	}

	e.teamSwitches = map[types.PartyID]teamSwitch{}
}

func (e *Engine) updateMembershipStats() {
	for _, team := range e.teams {
		team.Referrer.NumberOfEpoch += 1
		for _, referee := range team.Referees {
			referee.NumberOfEpoch += 1
		}
	}
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

func (e *Engine) ensureUniqueTeamID(deterministicTeamID types.TeamID) error {
	if _, exists := e.teams[deterministicTeamID]; exists {
		return ErrComputedTeamIDIsAlreadyInUse
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
				PartyID:       refereeID,
				JoinedAt:      time.Unix(0, refereeSnapshot.JoinedAt),
				NumberOfEpoch: refereeSnapshot.NumberOfEpoch,
			})
		}

		e.teams[teamID] = &types.Team{
			ID: teamID,
			Referrer: &types.Membership{
				PartyID:       referrerID,
				JoinedAt:      time.Unix(0, teamSnapshot.Referrer.JoinedAt),
				NumberOfEpoch: teamSnapshot.Referrer.NumberOfEpoch,
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
