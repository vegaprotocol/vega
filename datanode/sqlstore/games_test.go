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

package sqlstore_test

import (
	"context"
	"math/rand"
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/libs/slice"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type gameStores struct {
	blocks    *sqlstore.Blocks
	assets    *sqlstore.Assets
	accounts  *sqlstore.Accounts
	transfers *sqlstore.Transfers
	rewards   *sqlstore.Rewards
	parties   *sqlstore.Parties
	games     *sqlstore.Games
	teams     *sqlstore.Teams
}

func TestListGames(t *testing.T) {
	ctx := tempTransaction(t)
	stores := setupGamesTest(t, ctx)
	startingBlock := addTestBlockForTime(t, ctx, stores.blocks, time.Now())
	gamesData, gameIDs, _, teams, individuals := setupGamesData(ctx, t, stores, startingBlock, 50)
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	t.Run("Should list all games data if no filter is given", func(t *testing.T) {
		t.Run("and return all data for the most recent epoch if no epoch is given", func(t *testing.T) {
			want := filterForEpochs(50, 50, gamesData)
			t.Run("if no pagination is given", func(t *testing.T) {
				got, _, err := stores.games.ListGames(ctx, nil, nil, nil, nil, nil, nil, entities.CursorPagination{})
				assert.NoError(t, err)
				assert.Equal(t, want, got)
			})

			t.Run("if first page is requested", func(t *testing.T) {
				first := int32(2)
				pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
				require.NoError(t, err)
				got, pageInfo, err := stores.games.ListGames(ctx, nil, nil, nil, nil, nil, nil, pagination)
				assert.NoError(t, err)
				want := want[:2]
				assert.Equal(t, want, got)
				assert.Equal(t, entities.PageInfo{
					HasNextPage:     true,
					HasPreviousPage: false,
					StartCursor:     want[0].Cursor().Encode(),
					EndCursor:       want[1].Cursor().Encode(),
				}, pageInfo)
			})

			t.Run("if first page after cursor is requested", func(t *testing.T) {
				first := int32(2)
				after := gamesData[1].Cursor().Encode()
				pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
				require.NoError(t, err)
				got, pageInfo, err := stores.games.ListGames(ctx, nil, nil, nil, nil, nil, nil, pagination)
				assert.NoError(t, err)
				want := want[2:4]
				assert.Equal(t, want, got)
				assert.Equal(t, entities.PageInfo{
					HasNextPage:     true,
					HasPreviousPage: true,
					StartCursor:     want[0].Cursor().Encode(),
					EndCursor:       want[1].Cursor().Encode(),
				}, pageInfo)
			})

			t.Run("if last page is requested", func(t *testing.T) {
				last := int32(2)
				pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
				require.NoError(t, err)
				_, _, err = stores.games.ListGames(ctx, nil, nil, nil, nil, nil, nil, pagination)
				assert.Error(t, err)
			})

			t.Run("if last page before cursor is requested", func(t *testing.T) {
				last := int32(2)
				before := gamesData[2].Cursor().Encode()
				pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
				require.NoError(t, err)
				_, _, err = stores.games.ListGames(ctx, nil, nil, nil, nil, nil, nil, pagination)
				assert.Error(t, err)
			})
		})
		t.Run("and return data from the start to most recent epoch if no end epoch is given", func(t *testing.T) {
			t.Run("when start is less than 30 epochs before most recent", func(t *testing.T) {
				epochFrom := uint64(1)
				epochTo := uint64(20)
				got, _, err := stores.games.ListGames(ctx, nil, nil, ptr.From(epochFrom), ptr.From(epochTo), nil, nil, entities.CursorPagination{})
				require.NoError(t, err)
				want := filterForEpochs(1, 20, gamesData)
				require.Equal(t, len(want), len(got))
				assert.Equal(t, want, got)
			})
			t.Run("when start is more than 30 epochs before most recent", func(t *testing.T) {
				want := filterForEpochs(1, 30, gamesData)
				epochFrom := uint64(1)
				got, _, err := stores.games.ListGames(ctx, nil, nil, ptr.From(epochFrom), nil, nil, nil, entities.CursorPagination{})
				require.NoError(t, err)
				require.Equal(t, len(want), len(got))
				assert.Equal(t, want, got)
			})
		})
		t.Run("and return all data from 30 previous epochs to given end epoch", func(t *testing.T) {
			t.Run("if no start epoch given", func(t *testing.T) {
				epochTo := uint64(40)
				got, _, err := stores.games.ListGames(ctx, nil, nil, nil, ptr.From(epochTo), nil, nil, entities.CursorPagination{})
				require.NoError(t, err)
				want := filterForEpochs(11, 40, gamesData)
				require.Equal(t, len(want), len(got))
				assert.Equal(t, want, got)
			})
			t.Run("if start is more than 30 epochs before end", func(t *testing.T) {
				epochFrom := uint64(1)
				epochTo := uint64(40)
				got, _, err := stores.games.ListGames(ctx, nil, nil, ptr.From(epochFrom), ptr.From(epochTo), nil, nil, entities.CursorPagination{})
				require.NoError(t, err)
				want := filterForEpochs(11, 40, gamesData)
				require.Equal(t, len(want), len(got))
				assert.Equal(t, want, got)
			})
		})
	})
	t.Run("Should list a game's stats if gameID is provided", func(t *testing.T) {
		t.Run("and return data from the most recent epoch if no epoch is given", func(t *testing.T) {
			i := r.Intn(len(gameIDs))
			gameID := gameIDs[i]
			want := filterForGameID(filterForEpochs(50, 50, gamesData), gameID)
			got, _, err := stores.games.ListGames(ctx, ptr.From(gameID), nil, nil, nil, nil, nil, entities.CursorPagination{})
			require.NoError(t, err)
			require.Equal(t, len(want), len(got))
			assert.Equal(t, want, got)
		})
		t.Run("and return data for the 30 epochs up to the given end epoch", func(t *testing.T) {
			i := r.Intn(len(gameIDs))
			gameID := gameIDs[i]
			want := filterForGameID(filterForEpochs(11, 40, gamesData), gameID)
			epochTo := uint64(40)
			got, _, err := stores.games.ListGames(ctx, ptr.From(gameID), nil, nil, ptr.From(epochTo), nil, nil, entities.CursorPagination{})
			require.NoError(t, err)
			require.Equal(t, len(want), len(got))
			assert.Equal(t, want, got)
		})
		t.Run("and return data between the given start and end epochs", func(t *testing.T) {
			i := r.Intn(len(gameIDs))
			gameID := gameIDs[i]
			want := filterForGameID(filterForEpochs(21, 40, gamesData), gameID)
			epochFrom := uint64(21)
			epochTo := uint64(40)
			got, _, err := stores.games.ListGames(ctx, ptr.From(gameID), nil, ptr.From(epochFrom), ptr.From(epochTo), nil, nil, entities.CursorPagination{})
			require.NoError(t, err)
			require.Equal(t, len(want), len(got))
			assert.Equal(t, want, got)
		})
	})
	t.Run("Should list games for a specific entity type if specified", func(t *testing.T) {
		t.Run("and return data for the most recent epoch if no epoch is given", func(t *testing.T) {
			t.Run("when entity scope is teams", func(t *testing.T) {
				entityScope := vega.EntityScope_ENTITY_SCOPE_TEAMS
				want := filterForEntityScope(filterForEpochs(50, 50, gamesData), entityScope)
				got, _, err := stores.games.ListGames(ctx, nil, ptr.From(entityScope), nil, nil, nil, nil, entities.CursorPagination{})
				require.NoError(t, err)
				require.Equal(t, len(want), len(got))
				assert.Equal(t, want, got)
			})
			t.Run("when entity scope is individuals", func(t *testing.T) {
				entityScope := vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS
				want := filterForEntityScope(filterForEpochs(50, 50, gamesData), entityScope)
				got, _, err := stores.games.ListGames(ctx, nil, ptr.From(entityScope), nil, nil, nil, nil, entities.CursorPagination{})
				require.NoError(t, err)
				require.Equal(t, len(want), len(got))
				assert.Equal(t, want, got)
			})
		})
		t.Run("and return data for the 30 epochs up to the given end epoch", func(t *testing.T) {
			t.Run("when entity scope is teams", func(t *testing.T) {
				entityScope := vega.EntityScope_ENTITY_SCOPE_TEAMS
				want := filterForEntityScope(filterForEpochs(11, 40, gamesData), entityScope)
				epochTo := uint64(40)
				got, _, err := stores.games.ListGames(ctx, nil, ptr.From(entityScope), nil, ptr.From(epochTo), nil, nil, entities.CursorPagination{})
				require.NoError(t, err)
				require.Equal(t, len(want), len(got))
				for i, w := range want {
					for j, e := range w.Entities {
						wt := e.(*entities.TeamGameEntity)
						gt := got[i].Entities[j].(*entities.TeamGameEntity)
						assert.Equalf(t, wt.Team.TeamID, gt.Team.TeamID, "TeamID mismatch, game index: %d, entity index: %d", i, j)
						for k, m := range wt.Team.MembersParticipating {
							assert.Equalf(t, m.Individual, gt.Team.MembersParticipating[k].Individual, "Individual mismatch, game index: %d, entity index: %d, member index: %d", i, j, k)
							assert.Equal(t, m.Rank, gt.Team.MembersParticipating[k].Rank, "Rank mismatch, game index: %d, entity index: %d, member index: %d", i, j, k)
						}
						assert.Equal(t, wt.Rank, gt.Rank, "Rank mismatch, game index: %d, entity index: %d", i, j)
					}
				}
				assert.Equal(t, want, got)
			})
			t.Run("when entity scope is individuals", func(t *testing.T) {
				entityScope := vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS
				want := filterForEntityScope(filterForEpochs(11, 40, gamesData), entityScope)
				epochTo := uint64(40)
				got, _, err := stores.games.ListGames(ctx, nil, ptr.From(entityScope), nil, ptr.From(epochTo), nil, nil, entities.CursorPagination{})
				require.NoError(t, err)
				require.Equal(t, len(want), len(got))
				assert.Equal(t, want, got)
			})
		})
		t.Run("and return data between the given start and end epochs", func(t *testing.T) {
			t.Run("when entity scope is teams", func(t *testing.T) {
				entityScope := vega.EntityScope_ENTITY_SCOPE_TEAMS
				want := filterForEntityScope(filterForEpochs(21, 40, gamesData), entityScope)
				epochFrom := uint64(21)
				epochTo := uint64(40)
				got, _, err := stores.games.ListGames(ctx, nil, ptr.From(entityScope), ptr.From(epochFrom), ptr.From(epochTo), nil, nil, entities.CursorPagination{})
				require.NoError(t, err)
				require.Equal(t, len(want), len(got))
				assert.Equal(t, want, got)
			})
			t.Run("when entity scope is individuals", func(t *testing.T) {
				entityScope := vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS
				want := filterForEntityScope(filterForEpochs(21, 40, gamesData), entityScope)
				epochFrom := uint64(21)
				epochTo := uint64(40)
				got, _, err := stores.games.ListGames(ctx, nil, ptr.From(entityScope), ptr.From(epochFrom), ptr.From(epochTo), nil, nil, entities.CursorPagination{})
				require.NoError(t, err)
				require.Equal(t, len(want), len(got))
				assert.Equal(t, want, got)
			})
		})
	})
	t.Run("Should list game stats for a team if entity scope is not set and team ID is provided", func(t *testing.T) {
		t.Run("and return data from the most recent epoch if no epoch is given", func(t *testing.T) {
			// Randomly choose a team
			teamID := pickRandomTeam(r, teams)
			want := filterForTeamID(filterForEpochs(50, 50, gamesData), teamID.String())
			got, _, err := stores.games.ListGames(ctx, nil, nil, nil, nil, ptr.From(teamID), nil, entities.CursorPagination{})
			require.NoError(t, err)
			require.Equal(t, len(want), len(got))
			assert.Equal(t, want, got)
		})
	})
	t.Run("Should list games stats for an individual", func(t *testing.T) {
		t.Run("And the entity scope is not set and individual ID is provided", func(t *testing.T) {
			i := r.Intn(100)
			var partyID entities.PartyID

			if i%2 == 0 {
				// choose a random team member
				teamID := pickRandomTeam(r, teams)
				members := teams[teamID.String()]
				j := r.Intn(len(members))
				partyID = members[j].ID
			} else {
				// choose a random individual
				j := r.Intn(len(individuals))
				partyID = individuals[j].ID
			}
			want := filterForPartyID(filterForEpochs(50, 50, gamesData), partyID.String())
			got, _, err := stores.games.ListGames(ctx, nil, nil, nil, nil, nil, ptr.From(partyID), entities.CursorPagination{})
			require.NoError(t, err)
			require.Equal(t, len(want), len(got))
			assert.Equal(t, want, got)
		})
		t.Run("And the entity scope is teams and individual ID is provided", func(t *testing.T) {
			teamID := pickRandomTeam(r, teams)
			members := teams[teamID.String()]
			j := r.Intn(len(members))
			partyID := members[j].ID

			want := filterForPartyID(filterForEpochs(50, 50, gamesData), partyID.String())
			got, _, err := stores.games.ListGames(ctx, nil, ptr.From(vega.EntityScope_ENTITY_SCOPE_TEAMS), nil, nil, nil, ptr.From(partyID), entities.CursorPagination{})
			require.NoError(t, err)
			require.Equal(t, len(want), len(got))
			assert.Equal(t, want, got)
		})
		t.Run("And the entity scope is individuals and individual ID is provided", func(t *testing.T) {
			// choose a random individual
			j := r.Intn(len(individuals))
			partyID := individuals[j].ID

			want := filterForPartyID(filterForEpochs(50, 50, gamesData), partyID.String())
			got, _, err := stores.games.ListGames(ctx, nil, ptr.From(vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS), nil, nil, nil, ptr.From(partyID), entities.CursorPagination{})
			require.NoError(t, err)
			require.Equal(t, len(want), len(got))
			assert.Equal(t, want, got)
		})
	})
}

func setupGamesTest(t *testing.T, ctx context.Context) gameStores {
	t.Helper()
	return gameStores{
		blocks:    sqlstore.NewBlocks(connectionSource),
		assets:    sqlstore.NewAssets(connectionSource),
		accounts:  sqlstore.NewAccounts(connectionSource),
		transfers: sqlstore.NewTransfers(connectionSource),
		rewards:   sqlstore.NewRewards(ctx, connectionSource),
		parties:   sqlstore.NewParties(connectionSource),
		games:     sqlstore.NewGames(connectionSource),
		teams:     sqlstore.NewTeams(connectionSource),
	}
}

type gameDataKey struct {
	ID    string
	Epoch int64
}

func setupResultsStore(t *testing.T, gameIDs []string, epochCount int64) map[gameDataKey]entities.Game {
	t.Helper()

	store := make(map[gameDataKey]entities.Game)
	for _, id := range gameIDs {
		for epoch := int64(1); epoch <= epochCount; epoch++ {
			key := gameDataKey{
				ID:    id,
				Epoch: epoch,
			}
			store[key] = entities.Game{
				ID:    entities.GameID(id),
				Epoch: uint64(epoch),
			}
		}
	}

	return store
}

func setupGamesData(ctx context.Context, t *testing.T, stores gameStores, block entities.Block, epochCount int64) (
	[]entities.Game, []string, map[gameDataKey][]entities.Reward, map[string][]entities.Party, []entities.Party,
) {
	t.Helper()

	gameCount := 5
	teamCount := 3
	individualCount := 5

	gameIDs := getGameIDs(t, gameCount)
	gameEntities := setupResultsStore(t, gameIDs, epochCount)
	teams := getTeams(t, ctx, stores, block, teamCount)
	individuals := getIndividuals(t, ctx, stores, block, individualCount)
	gameAssets := make(map[string]*entities.Asset)
	gameEntityScopes := make(map[string]vega.EntityScope)

	i := 0
	for _, gameID := range gameIDs {
		gID := entities.GameID(gameID)
		asset := CreateAsset(t, ctx, stores.assets, block)
		fromAccount := CreateAccount(t, ctx, stores.accounts, block, AccountForAsset(asset))
		toAccount := CreateAccount(t, ctx, stores.accounts, block, AccountWithType(vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD), AccountForAsset(asset))
		// create the recurring transfers that are games
		var recurringTransfer eventspb.RecurringTransfer
		if i%2 == 0 {
			recurringTransfer = eventspb.RecurringTransfer{
				StartEpoch: 1,
				Factor:     "0.1",
				DispatchStrategy: &vega.DispatchStrategy{
					AssetForMetric:       asset.ID.String(),
					Metric:               vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
					EntityScope:          vega.EntityScope_ENTITY_SCOPE_TEAMS,
					DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
				},
			}
			gameEntityScopes[gameID] = vega.EntityScope_ENTITY_SCOPE_TEAMS
		} else {
			recurringTransfer = eventspb.RecurringTransfer{
				StartEpoch: 1,
				Factor:     "0.1",
				DispatchStrategy: &vega.DispatchStrategy{
					AssetForMetric:       asset.ID.String(),
					Metric:               vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
					EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
					IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_ALL,
					DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
				},
			}
			gameEntityScopes[gameID] = vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS
		}
		transfer := NewTransfer(t, ctx, stores.accounts, block,
			TransferWithAsset(asset),
			TransferFromToAccounts(fromAccount, toAccount),
			TransferAsRecurring(&recurringTransfer),
			TransferWithGameID(ptr.From(gID.String())),
		)

		err := stores.transfers.Upsert(ctx, transfer)
		require.NoError(t, err)
		gameAssets[gameID] = asset
		i++
	}

	rewards := make(map[gameDataKey][]entities.Reward)
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	teamTotalRewards := make(map[gameDataKey]map[string]*num.Uint)
	teamMemberTotalRewards := make(map[gameDataKey]map[string]map[string]*num.Uint)
	individualTotalRewards := make(map[gameDataKey]map[string]*num.Uint)

	for epoch := int64(1); epoch <= epochCount; epoch++ {
		block = addTestBlockForTime(t, ctx, stores.blocks, block.VegaTime.Add(time.Minute))
		seqNum := uint64(1)
		for _, gameID := range gameIDs {
			// create the rewards for the games
			// we want to create the rewards for each participant in the game
			participants := uint64(0)
			market := entities.MarketID(GenerateID())
			teamEntities := make([]entities.GameEntity, 0)
			individualEntities := make([]entities.GameEntity, 0)
			gID := entities.GameID(gameID)
			asset := gameAssets[gameID]
			gk := gameDataKey{
				ID:    gameID,
				Epoch: epoch,
			}
			pk := gameDataKey{
				ID:    gameID,
				Epoch: epoch - 1,
			}
			if gameEntityScopes[gameID] == vega.EntityScope_ENTITY_SCOPE_TEAMS {
				if teamTotalRewards[gk] == nil {
					teamTotalRewards[gk] = make(map[string]*num.Uint)
				}
				if teamMemberTotalRewards[gk] == nil {
					teamMemberTotalRewards[gk] = make(map[string]map[string]*num.Uint)
				}
				for team, members := range teams {
					if teamMemberTotalRewards[gk][team] == nil {
						teamMemberTotalRewards[gk][team] = make(map[string]*num.Uint)
						// carry forward the previous totals
						if teamMemberTotalRewards[pk] != nil && teamMemberTotalRewards[pk][team] != nil {
							for k, v := range teamMemberTotalRewards[pk][team] {
								teamMemberTotalRewards[gk][team][k] = v.Clone()
							}
						}
					}
					teamRewards := num.NewUint(0)
					teamVolume := num.DecimalZero()
					memberEntities := make([]*entities.IndividualGameEntity, 0)
					for _, member := range members {
						amount := num.DecimalFromInt64(r.Int63n(1000))
						reward := addTestReward(t, ctx, stores.rewards, member, *asset, market, epoch, "", block.VegaTime, block, seqNum, amount, generateTxHash(), &gID)
						reward.TeamID = ptr.From(entities.TeamID(team))
						rewards[gk] = append(rewards[gk], reward)
						rewardEarned, _ := num.UintFromDecimal(amount)
						individualEntity := entities.IndividualGameEntity{
							Individual:          member.ID.String(),
							Rank:                0,
							Volume:              num.DecimalZero(),
							RewardMetric:        vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
							RewardEarned:        rewardEarned,
							RewardEarnedQuantum: rewardEarned,
						}
						teamRewards = teamRewards.Add(teamRewards, individualEntity.RewardEarned)
						teamVolume = teamVolume.Add(individualEntity.Volume)
						if _, ok := teamMemberTotalRewards[pk][team][member.ID.String()]; !ok {
							teamMemberTotalRewards[gk][team][member.ID.String()] = num.NewUint(0)
						} else {
							// carry forward the previous totals
							teamMemberTotalRewards[gk][team][member.ID.String()] = teamMemberTotalRewards[pk][team][member.ID.String()].Clone()
						}
						teamMemberTotalRewards[gk][team][member.ID.String()] = teamMemberTotalRewards[gk][team][member.ID.String()].
							Add(teamMemberTotalRewards[gk][team][member.ID.String()], individualEntity.RewardEarned)
						individualEntity.TotalRewardsEarned = teamMemberTotalRewards[gk][team][member.ID.String()]
						individualEntity.TotalRewardsEarnedQuantum = teamMemberTotalRewards[gk][team][member.ID.String()]
						memberEntities = append(memberEntities, &individualEntity)
						participants++
						seqNum++
					}
					// Rank the individual members of the team participating in the game
					sort.Slice(memberEntities, func(i, j int) bool {
						return memberEntities[i].TotalRewardsEarned.GT(memberEntities[j].TotalRewardsEarned)
					})
					// now assign the individual member ranks
					for i := range memberEntities {
						memberEntities[i].Rank = uint64(i + 1)
					}
					teamEntity := entities.TeamGameEntity{
						Team: entities.TeamGameParticipation{
							TeamID:               entities.TeamID(team),
							MembersParticipating: memberEntities,
						},
						Rank:                0,
						Volume:              teamVolume,
						RewardMetric:        vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
						RewardEarned:        teamRewards,
						RewardEarnedQuantum: teamRewards,
					}
					if teamTotalRewards[gk][team] == nil {
						if teamTotalRewards[pk] == nil || teamTotalRewards[pk][team] == nil {
							teamTotalRewards[gk][team] = num.NewUint(0)
						} else {
							teamTotalRewards[gk][team] = teamTotalRewards[pk][team].Clone()
						}
					}
					teamTotalRewards[gk][team] = teamTotalRewards[gk][team].Add(teamTotalRewards[gk][team], teamRewards)
					teamEntity.TotalRewardsEarned = teamTotalRewards[gk][team]
					teamEntity.TotalRewardsEarnedQuantum = teamTotalRewards[gk][team]
					teamEntities = append(teamEntities, &teamEntity)
				}
				// now let's order the team totals and set the ranks for each team
				teamRanking := rankEntity(teamTotalRewards[gk])
				for _, ge := range teamEntities {
					te := ge.(*entities.TeamGameEntity)
					memberRankings := rankEntity(teamMemberTotalRewards[gk][te.Team.TeamID.String()])
					te.Rank = teamRanking[te.Team.TeamID.String()]
					for _, m := range te.Team.MembersParticipating {
						m.Rank = memberRankings[m.Individual]
					}
					// now that the team members have been ranked, we need to order the team members by rank
					sort.Slice(te.Team.MembersParticipating, func(i, j int) bool {
						return te.Team.MembersParticipating[i].Rank < te.Team.MembersParticipating[j].Rank || (te.Team.MembersParticipating[i].Rank == te.Team.MembersParticipating[j].Rank &&
							te.Team.MembersParticipating[i].Individual < te.Team.MembersParticipating[j].Individual)
					})
				}

				// now that we have the ranks for the teams ranked, we need to order the team entities by rank
				sort.Slice(teamEntities, func(i, j int) bool {
					return teamEntities[i].(*entities.TeamGameEntity).Rank < teamEntities[j].(*entities.TeamGameEntity).Rank || (teamEntities[i].(*entities.TeamGameEntity).Rank == teamEntities[j].(*entities.TeamGameEntity).Rank &&
						teamEntities[i].(*entities.TeamGameEntity).Team.TeamID.String() < teamEntities[j].(*entities.TeamGameEntity).Team.TeamID.String())
				})

				gameEntity := gameEntities[gk]
				gameEntity.Participants = participants
				gameEntity.Entities = teamEntities
				gameEntity.RewardAssetID = asset.ID

				gameEntities[gk] = gameEntity
			} else {
				if individualTotalRewards[gk] == nil {
					individualTotalRewards[gk] = make(map[string]*num.Uint)
					if individualTotalRewards[pk] != nil {
						// carry forward the previous totals for the individuals
						for k, v := range individualTotalRewards[pk] {
							individualTotalRewards[gk][k] = v.Clone()
						}
					}
				}
				for i, individual := range individuals {
					amount := num.DecimalFromInt64(r.Int63n(1000))
					reward := addTestReward(t, ctx, stores.rewards, individual, *asset, market, epoch, "", block.VegaTime, block, seqNum, amount, generateTxHash(), &gID)
					rewards[gk] = append(rewards[gk], reward)
					rewardEarned, _ := num.UintFromDecimal(amount)
					individualEntity := entities.IndividualGameEntity{
						Individual:          individual.ID.String(),
						Rank:                uint64(i + 1),
						Volume:              num.DecimalZero(),
						RewardMetric:        vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
						RewardEarned:        rewardEarned,
						RewardEarnedQuantum: rewardEarned,
					}
					individualEntities = append(individualEntities, &individualEntity)
					seqNum++
					participants++
					if _, ok := individualTotalRewards[gk][individual.ID.String()]; !ok {
						individualTotalRewards[gk][individual.ID.String()] = num.NewUint(0)
					} else {
						// carry forward the previous totals
						individualTotalRewards[gk][individual.ID.String()] = individualTotalRewards[pk][individual.ID.String()].Clone()
					}
					individualTotalRewards[gk][individual.ID.String()].
						Add(individualTotalRewards[gk][individual.ID.String()], individualEntity.RewardEarned)
					individualEntity.TotalRewardsEarned = individualTotalRewards[gk][individual.ID.String()]
					individualEntity.TotalRewardsEarnedQuantum = individualTotalRewards[gk][individual.ID.String()]
				}
				individualRanking := rankEntity(individualTotalRewards[gk])
				for _, ge := range individualEntities {
					ie := ge.(*entities.IndividualGameEntity)
					ie.Rank = individualRanking[ie.Individual]
				}
				sort.Slice(individualEntities, func(i, j int) bool {
					return individualEntities[i].(*entities.IndividualGameEntity).Rank < individualEntities[j].(*entities.IndividualGameEntity).Rank || (individualEntities[i].(*entities.IndividualGameEntity).Rank == individualEntities[j].(*entities.IndividualGameEntity).Rank &&
						individualEntities[i].(*entities.IndividualGameEntity).Individual < individualEntities[j].(*entities.IndividualGameEntity).Individual)
				})

				gameEntity := gameEntities[gk]
				gameEntity.Participants = participants
				gameEntity.Entities = individualEntities
				gameEntity.RewardAssetID = asset.ID

				gameEntities[gk] = gameEntity
			}
		}
	}

	results := make([]entities.Game, 0, len(gameEntities))
	for _, game := range gameEntities {
		results = append(results, game)
	}

	// IMPORTANT!!!! We MUST refresh the materialized views or the tests will fail because there will be NO DATA!!!
	_, err := connectionSource.Connection.Exec(ctx, "REFRESH MATERIALIZED VIEW game_stats")
	require.NoError(t, err)
	_, err = connectionSource.Connection.Exec(ctx, "REFRESH MATERIALIZED VIEW game_stats_current")
	require.NoError(t, err)

	return orderResults(results), gameIDs, rewards, teams, individuals
}

func orderResults(results []entities.Game) []entities.Game {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Epoch > results[j].Epoch ||
			(results[i].Epoch == results[j].Epoch && results[i].ID.String() < results[j].ID.String())
	})
	return results
}

func filterForEpochs(start, end int64, gamesData []entities.Game) []entities.Game {
	validEpochs := make([]int64, end-start+1)
	for i := range validEpochs {
		validEpochs[i] = start + int64(i)
	}
	filtered := make([]entities.Game, 0)
	for _, game := range gamesData {
		if slice.Contains(validEpochs, int64(game.Epoch)) {
			filtered = append(filtered, game)
		}
	}
	// ensure we are correctly ordered
	return orderResults(filtered)
}

func filterForGameID(gamesData []entities.Game, gameID string) []entities.Game {
	filtered := make([]entities.Game, 0)
	for _, game := range gamesData {
		if game.ID.String() == gameID {
			filtered = append(filtered, game)
		}
	}
	return orderResults(filtered)
}

func filterForTeamID(gamesData []entities.Game, teamID string) []entities.Game {
	filtered := make([]entities.Game, 0)
	for _, game := range gamesData {
		for _, entity := range game.Entities {
			if teamEntity, ok := entity.(*entities.TeamGameEntity); ok {
				if teamEntity.Team.TeamID.String() == teamID {
					filtered = append(filtered, game)
					break
				}
			}
		}
	}

	return filtered
}

func filterForPartyID(gamesData []entities.Game, partyID string) []entities.Game {
	filtered := make([]entities.Game, 0)
	for _, game := range gamesData {
		for _, entity := range game.Entities {
			switch e := entity.(type) {
			case *entities.TeamGameEntity:
				for _, member := range e.Team.MembersParticipating {
					if member.Individual == partyID {
						filtered = append(filtered, game)
						break
					}
				}
			case *entities.IndividualGameEntity:
				if e.Individual == partyID {
					filtered = append(filtered, game)
					break
				}
			}
		}
	}
	return orderResults(filtered)
}

func filterForEntityScope(gamesData []entities.Game, entityScope vega.EntityScope) []entities.Game {
	filtered := make([]entities.Game, 0)
	for _, game := range gamesData {
		if len(game.Entities) == 0 {
			continue
		}
		switch game.Entities[0].(type) {
		case *entities.TeamGameEntity:
			if entityScope == vega.EntityScope_ENTITY_SCOPE_TEAMS {
				filtered = append(filtered, game)
			}
		case *entities.IndividualGameEntity:
			if entityScope == vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS {
				filtered = append(filtered, game)
			}
		}
	}
	return orderResults(filtered)
}

func getGameIDs(t *testing.T, count int) []string {
	t.Helper()
	ids := make([]string, count)
	for i := 0; i < count; i++ {
		ids[i] = GenerateID()
	}
	return ids
}

func getTeams(t *testing.T, ctx context.Context, stores gameStores, block entities.Block, count int) map[string][]entities.Party {
	t.Helper()
	teams := make(map[string][]entities.Party)
	for i := 0; i < count; i++ {
		teamID := entities.TeamID(GenerateID())
		referrer := entities.PartyID(GenerateID())
		team := entities.Team{
			ID:             teamID,
			Referrer:       referrer,
			Name:           "",
			TeamURL:        nil,
			AvatarURL:      nil,
			Closed:         false,
			CreatedAt:      block.VegaTime,
			CreatedAtEpoch: 0,
			VegaTime:       block.VegaTime,
		}
		err := stores.teams.AddTeam(ctx, &team)
		require.NoError(t, err)

		members := make([]entities.Party, count)
		for j := 0; j < count; j++ {
			members[j] = addTestParty(t, ctx, stores.parties, block)
			referee := entities.TeamMember{
				TeamID:        teamID,
				PartyID:       members[j].ID,
				JoinedAt:      block.VegaTime,
				JoinedAtEpoch: 0,
				VegaTime:      block.VegaTime,
			}
			err := stores.teams.RefereeJoinedTeam(ctx, &referee)
			require.NoError(t, err)
		}
		teams[teamID.String()] = members
	}
	return teams
}

func getIndividuals(t *testing.T, ctx context.Context, stores gameStores, block entities.Block, count int) []entities.Party {
	t.Helper()
	individuals := make([]entities.Party, count)
	for i := 0; i < count; i++ {
		individuals[i] = addTestParty(t, ctx, stores.parties, block)
	}
	return individuals
}

func rankEntity(entities map[string]*num.Uint) map[string]uint64 {
	type entityRank struct {
		ID    string
		Total *num.Uint
	}
	entityRanks := make([]entityRank, 0, len(entities))
	for k, v := range entities {
		entityRanks = append(entityRanks, entityRank{
			ID:    k,
			Total: v,
		})
	}
	sort.Slice(entityRanks, func(i, j int) bool {
		return entityRanks[i].Total.GT(entityRanks[j].Total) || (entityRanks[i].Total.EQ(entityRanks[j].Total) && entityRanks[i].ID < entityRanks[j].ID)
	})
	// now that we have the totals ordered, we can assign ranks
	ranks := make(map[string]uint64)
	for i, e := range entityRanks {
		if i > 0 && e.Total.EQ(entityRanks[i-1].Total) {
			// if the totals are the same, they should have the same rank
			ranks[e.ID] = ranks[entityRanks[i-1].ID]
			continue
		}
		ranks[e.ID] = uint64(i + 1)
	}
	return ranks
}

func pickRandomTeam(r *rand.Rand, teams map[string][]entities.Party) entities.TeamID {
	i := r.Intn(len(teams))
	j := 0
	for k := range teams {
		if i == j {
			return entities.TeamID(k)
		}
		j++
	}
	return ""
}
