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

package sqlstore

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/shopspring/decimal"
)

type Games struct {
	*ConnectionSource
}

var gameOrdering = TableOrdering{
	ColumnOrdering{Name: "epoch_id", Sorting: ASC},
	ColumnOrdering{Name: "game_id", Sorting: DESC},
}

func NewGames(connectionSource *ConnectionSource) *Games {
	return &Games{
		ConnectionSource: connectionSource,
	}
}

type GameReward struct {
	PartyID                 entities.PartyID
	AssetID                 entities.AssetID
	MarketID                entities.MarketID
	EpochID                 int64
	Amount                  decimal.Decimal
	QuantumAmount           decimal.Decimal
	PercentOfTotal          float64
	RewardType              string
	Timestamp               time.Time
	TxHash                  entities.TxHash
	VegaTime                time.Time
	SeqNum                  uint64
	LockedUntilEpochID      int64
	GameID                  []byte
	DispatchStrategy        vega.DispatchStrategy
	TeamID                  entities.TeamID
	MemberRank              *int64
	TeamRank                *int64
	TotalRewards            num.Decimal
	TotalRewardsQuantum     num.Decimal
	TeamTotalRewards        *num.Decimal
	TeamTotalRewardsQuantum *num.Decimal
	EntityScope             string
}

func (g *Games) ListGames(ctx context.Context, gameID *string, entityScope *vega.EntityScope, epochFrom, epochTo *uint64,
	teamID *entities.TeamID, partyID *entities.PartyID, pagination entities.CursorPagination,
) ([]entities.Game, entities.PageInfo, error) {
	var pageInfo entities.PageInfo

	var gameRewards []GameReward

	// because we have to build the games data from the rewards data, paging backwards adds more complexity
	// therefore we aren't going to support it for now as the games data API is high priority
	if pagination.HasBackward() {
		return nil, pageInfo, fmt.Errorf("backward pagination is not currently supported")
	}

	query, args, err := g.buildGamesQuery(gameID, entityScope, epochFrom, epochTo, teamID, partyID, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, g.Connection, &gameRewards, query, args...); err != nil {
		return nil, pageInfo, err
	}

	games, err := parseGameRewards(gameRewards)
	if err != nil {
		return nil, pageInfo, err
	}

	games, pageInfo = entities.PageEntities[*v2.GameEdge](games, pagination)

	return games, pageInfo, nil
}

func (g *Games) parseEpochs(from, to *uint64) (uint64, uint64) {
	var eFrom, eTo uint64
	if from != nil || to != nil {
		// no more than 30 epochs for performance sake
		if from != nil && to == nil {
			eFrom, eTo = *from, *from+30-1
		} else if from == nil && to != nil {
			eTo, eFrom = *to, *to-30+1
		} else if from != nil && to != nil {
			eFrom, eTo = *from, *to
			if eTo-eFrom > 30 {
				eFrom = eTo - 30 + 1
			}
		}
	}
	return eFrom, eTo
}

func (g *Games) buildPagingQuery(selectTable string, gameID *string, entityScope *vega.EntityScope, epochFrom, epochTo uint64,
	teamID *entities.TeamID, partyID *entities.PartyID, pagination entities.CursorPagination,
) (string, []interface{}, error) {
	selectQuery := fmt.Sprintf(`select distinct game_id, epoch_id from %s`, selectTable)
	var where []string
	var args []interface{}

	if epochFrom > 0 && epochTo > 0 {
		where = append(where, fmt.Sprintf("epoch_id >= %s AND epoch_id <= %s",
			nextBindVar(&args, epochFrom), nextBindVar(&args, epochTo)))
	}

	if gameID != nil {
		where = append(where, fmt.Sprintf("game_id = %s", nextBindVar(&args, entities.GameID(*gameID))))
	}

	if entityScope != nil {
		where = append(where, fmt.Sprintf("entity_scope = %s", nextBindVar(&args, entityScope.String())))

		// only add the teams filter if the entity scope is teams or not specified
		if *entityScope == vega.EntityScope_ENTITY_SCOPE_TEAMS && teamID != nil {
			where = append(where, fmt.Sprintf("team_id = %s", nextBindVar(&args, teamID)))
		}
	} else if entityScope == nil && teamID != nil {
		where = append(where, fmt.Sprintf("team_id = %s", nextBindVar(&args, teamID)))
	}

	// We should be able to filter by party regardless of the entity scope
	if partyID != nil {
		where = append(where, fmt.Sprintf("party_id = %s", nextBindVar(&args, partyID)))
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	query := fmt.Sprintf("%s %s", selectQuery, whereClause)
	return PaginateQuery[entities.GameCursor](query, args, gameOrdering, pagination)
}

func (g *Games) buildGamesQuery(gameID *string, entityScope *vega.EntityScope, epochFrom, epochTo *uint64,
	teamID *entities.TeamID, partyID *entities.PartyID, pagination entities.CursorPagination,
) (string, []interface{}, error) {
	// Games are intrinsically created by a recurring transfer with a game ID
	// Rewards are paid out to participants of a game and the game ID is recorded on the reward
	// We need to query the rewards and build the games data from that.
	// If we page on the reward data, we may not have a complete data set for the game. Therefore we need to only page/filter on the distinct game IDs per epoch
	// and then use that data to query the corresponding rewards data we need for the API to return.

	eFrom, eTo := g.parseEpochs(epochFrom, epochTo)
	// The select table query determines if we should just be querying the games data for the most current epoch or all epochs
	selectTable := g.getSelectTable(eFrom, eTo)
	// The page query determines which games/epochs should be included in the result set for pagination
	// For example, if we have 100 games, and we want to page on the first 10, we would need to know which games to include rewards for
	// The number of rewards we may get back in order to build the data will be much more than just 10 records.
	pageQuery, args, err := g.buildPagingQuery(selectTable, gameID, entityScope, eFrom, eTo, teamID, partyID, pagination)
	if err != nil {
		return "", nil, err
	}

	query := fmt.Sprintf("select s.* from %s s join (%s) as p on s.game_id = p.game_id and s.epoch_id = p.epoch_id order by s.epoch_id desc, s.game_id", selectTable, pageQuery)

	return query, args, nil
}

func (g *Games) getSelectTable(from, to uint64) string {
	if from == 0 && to == 0 {
		return `game_stats_current`
	}
	return `game_stats`
}

func parseGameRewards(rewards []GameReward) ([]entities.Game, error) {
	if len(rewards) <= 0 {
		return []entities.Game{}, nil
	}

	type gameKey struct {
		EpochID uint64
		GameID  entities.GameID
	}
	games := make(map[gameKey]entities.Game)
	lastGameID := entities.GameID("")
	lastEpochID := int64(0)
	participants := uint64(0)

	gameIndividuals := make(map[gameKey][]entities.GameEntity)
	teamMembers := make(map[gameKey]map[entities.TeamID][]*entities.IndividualGameEntity)
	teamRanks := make(map[gameKey]map[entities.TeamID]uint64)

	var game entities.Game
	var gk gameKey

	// first go through all the rewards and build the participation stats
	// if the reward is for a team participant, i.e. there is a team ID then the participant will be added to the teamMembers map
	// otherwise we add it to the gameIndividuals map
	for i := range rewards {
		gID := hex.EncodeToString(rewards[i].GameID)
		currentGameID := entities.GameID(gID)
		currentEpochID := rewards[i].EpochID
		gk = gameKey{
			EpochID: uint64(currentEpochID),
			GameID:  currentGameID,
		}

		if currentGameID != lastGameID || currentEpochID != lastEpochID {
			// add the game to the map of games
			lastKey := gameKey{
				EpochID: uint64(lastEpochID),
				GameID:  lastGameID,
			}
			if lastGameID != "" && lastEpochID != 0 {
				game.Participants = participants
				games[lastKey] = game
			}

			game = entities.Game{
				ID:            currentGameID,
				Epoch:         uint64(currentEpochID),
				Participants:  participants,
				Entities:      []entities.GameEntity{},
				RewardAssetID: rewards[i].AssetID,
			}

			lastGameID = currentGameID
			lastEpochID = currentEpochID
			participants = 0
			games[gk] = game
		}

		rewardEarned, _ := num.UintFromDecimal(rewards[i].Amount)
		totalRewardsEarned, _ := num.UintFromDecimal(rewards[i].TotalRewards)
		rewardEarnedQuantum, _ := num.UintFromDecimal(rewards[i].QuantumAmount)
		totalRewardsEarnedQuantum, _ := num.UintFromDecimal(rewards[i].TotalRewardsQuantum)

		var rank uint64
		if rewards[i].MemberRank != nil {
			rank = uint64(*rewards[i].MemberRank)
		}

		individual := entities.IndividualGameEntity{
			Individual:                rewards[i].PartyID.String(),
			Rank:                      rank,
			Volume:                    num.DecimalZero(),
			RewardMetric:              rewards[i].DispatchStrategy.Metric,
			RewardEarned:              rewardEarned,
			TotalRewardsEarned:        totalRewardsEarned,
			RewardEarnedQuantum:       rewardEarnedQuantum,
			TotalRewardsEarnedQuantum: totalRewardsEarnedQuantum,
		}

		if rewards[i].TeamID != "" {
			currentTeamID := rewards[i].TeamID
			if teamMembers[gk] == nil {
				teamMembers[gk] = make(map[entities.TeamID][]*entities.IndividualGameEntity)
			}
			teamMembers[gk][currentTeamID] = append(teamMembers[gk][currentTeamID], &individual)
			if rewards[i].TeamRank == nil {
				return nil, fmt.Errorf("team rank is nil for team %s", currentTeamID)
			}

			if teamRanks[gk] == nil {
				teamRanks[gk] = make(map[entities.TeamID]uint64)
			}

			teamRanks[gk][currentTeamID] = uint64(*rewards[i].TeamRank)
		} else {
			gameIndividuals[gk] = append(gameIndividuals[gk], &individual)
		}
		participants++
	}

	game.Participants = participants
	games[gk] = game

	results := make([]entities.Game, 0, len(games))
	// now that we have the participation involvement, we can use that to build the game entities for each game.
	for key, game := range games {
		if teamMembers[key] != nil {
			for teamID, individuals := range teamMembers[key] {
				sort.Slice(individuals, func(i, j int) bool {
					return individuals[i].Rank < individuals[j].Rank || (individuals[i].Rank == individuals[j].Rank && individuals[i].Individual < individuals[j].Individual)
				})
				team := entities.TeamGameParticipation{
					TeamID:               teamID,
					MembersParticipating: individuals,
				}

				teamVolume := num.DecimalZero()
				teamRewardEarned := num.NewUint(0)
				teamTotalRewardsEarned := num.NewUint(0)
				teamRewardEarnedQuantum := num.NewUint(0)
				teamTotalRewardsEarnedQuantum := num.NewUint(0)
				rewardMetric := vega.DispatchMetric_DISPATCH_METRIC_UNSPECIFIED
				for _, individual := range individuals {
					if rewardMetric == vega.DispatchMetric_DISPATCH_METRIC_UNSPECIFIED {
						rewardMetric = individual.RewardMetric
					}
					teamVolume = teamVolume.Add(individual.Volume)
					teamRewardEarned = teamRewardEarned.Add(teamRewardEarned, individual.RewardEarned)
					teamTotalRewardsEarned = teamTotalRewardsEarned.Add(teamTotalRewardsEarned, individual.TotalRewardsEarned)
					teamRewardEarnedQuantum = teamRewardEarnedQuantum.Add(teamRewardEarnedQuantum, individual.RewardEarnedQuantum)
					teamTotalRewardsEarnedQuantum = teamTotalRewardsEarnedQuantum.Add(teamTotalRewardsEarnedQuantum, individual.TotalRewardsEarnedQuantum)
				}
				game.Entities = append(game.Entities, &entities.TeamGameEntity{
					Team:                      team,
					Rank:                      teamRanks[key][teamID],
					Volume:                    teamVolume,
					RewardMetric:              rewardMetric,
					RewardEarned:              teamRewardEarned,
					TotalRewardsEarned:        teamTotalRewardsEarned,
					RewardEarnedQuantum:       teamRewardEarnedQuantum,
					TotalRewardsEarnedQuantum: teamTotalRewardsEarnedQuantum,
				})
			}
			sort.Slice(game.Entities, func(i, j int) bool {
				return game.Entities[i].(*entities.TeamGameEntity).Rank < game.Entities[j].(*entities.TeamGameEntity).Rank ||
					(game.Entities[i].(*entities.TeamGameEntity).Rank == game.Entities[j].(*entities.TeamGameEntity).Rank &&
						game.Entities[i].(*entities.TeamGameEntity).Team.TeamID < game.Entities[j].(*entities.TeamGameEntity).Team.TeamID)
			})
		}
		if gameIndividuals[key] != nil {
			game.Entities = append(game.Entities, gameIndividuals[key]...)
			sort.Slice(game.Entities, func(i, j int) bool {
				return game.Entities[i].(*entities.IndividualGameEntity).Rank < game.Entities[j].(*entities.IndividualGameEntity).Rank ||
					(game.Entities[i].(*entities.IndividualGameEntity).Rank == game.Entities[j].(*entities.IndividualGameEntity).Rank &&
						game.Entities[i].(*entities.IndividualGameEntity).Individual < game.Entities[j].(*entities.IndividualGameEntity).Individual)
			})
		}
		results = append(results, game)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Epoch > results[j].Epoch ||
			(results[i].Epoch == results[j].Epoch && results[i].ID < results[j].ID)
	})
	return results, nil
}
