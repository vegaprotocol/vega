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

package rewards

import (
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
)

// given party contribution scores, reward multipliers and distribution strategy calculate the payout per party.
func calculateRewardsByContributionIndividual(epochSeq, asset, accountID string, balance *num.Uint, partyContribution []*types.PartyContributionScore, rewardFactors map[string]num.Decimal, timestamp time.Time, ds *vega.DispatchStrategy) *payout {
	po := &payout{
		asset:           asset,
		fromAccount:     accountID,
		epochSeq:        epochSeq,
		timestamp:       timestamp.Unix(),
		partyToAmount:   map[string]*num.Uint{},
		lockedForEpochs: ds.LockPeriod,
	}
	total := num.UintZero()
	rewardBalance := balance.ToDecimal()

	var partyScores []*types.PartyContributionScore
	if ds.DistributionStrategy == vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA {
		partyScores = proRataRewardCalculator(partyContribution, rewardFactors)
	} else if ds.DistributionStrategy == vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK {
		partyScores = rankingRewardCalculator(partyContribution, ds.RankTable, rewardFactors)
	}

	for _, p := range partyScores {
		partyReward, _ := num.UintFromDecimal(rewardBalance.Mul(p.Score))
		if !partyReward.IsZero() {
			po.partyToAmount[p.Party] = partyReward
			total.AddSum(partyReward)
		}
	}
	po.totalReward = total
	if total.IsZero() {
		return nil
	}
	return po
}

// given party contribution scores, reward multipliers and distribution strategy calculate the payout per party in a team.
func calculateRewardsByContributionTeam(epochSeq, asset, accountID string, balance *num.Uint, teamContribution []*types.PartyContributionScore, teamPartyContribution map[string][]*types.PartyContributionScore, rewardFactors map[string]num.Decimal, timestamp time.Time, ds *vega.DispatchStrategy) *payout {
	po := &payout{
		asset:           asset,
		fromAccount:     accountID,
		epochSeq:        epochSeq,
		timestamp:       timestamp.Unix(),
		partyToAmount:   map[string]*num.Uint{},
		lockedForEpochs: ds.LockPeriod,
	}
	total := num.UintZero()
	rewardBalance := balance.ToDecimal()

	var teamScores []*types.PartyContributionScore
	if ds.DistributionStrategy == vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA {
		teamScores = proRataRewardCalculator(teamContribution, map[string]num.Decimal{})
	} else if ds.DistributionStrategy == vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK {
		teamScores = rankingRewardCalculator(teamContribution, ds.RankTable, map[string]num.Decimal{})
	}

	partyScores := []*types.PartyContributionScore{}
	totalScore := num.DecimalZero()
	for _, teamScore := range teamScores {
		partyScores = append(partyScores, calcPartyInTeamRewardShare(teamScore, teamPartyContribution[teamScore.Party], rewardFactors)...)
	}
	for _, pcs := range partyScores {
		totalScore = totalScore.Add(pcs.Score)
	}

	capAtOne(partyScores, totalScore)

	for _, p := range partyScores {
		partyReward, _ := num.UintFromDecimal(rewardBalance.Mul(p.Score))
		if !partyReward.IsZero() {
			po.partyToAmount[p.Party] = partyReward
			total.AddSum(partyReward)
		}
	}
	po.totalReward = total
	if total.IsZero() {
		return nil
	}
	return po
}

func calcPartyInTeamRewardShare(teamScore *types.PartyContributionScore, partyToMetricScore []*types.PartyContributionScore, rewardFactors map[string]num.Decimal) []*types.PartyContributionScore {
	ps := make([]*types.PartyContributionScore, 0, len(partyToMetricScore))

	totalScores := num.DecimalZero()
	for _, pcs := range partyToMetricScore {
		if pcs.Score.IsZero() {
			continue
		}
		rewardFactor := num.DecimalOne()
		if factor, ok := rewardFactors[pcs.Party]; ok {
			rewardFactor = factor
		}
		score := rewardFactor.Mul(pcs.Score)
		ps = append(ps, &types.PartyContributionScore{Party: pcs.Party, Score: score})
		totalScores = totalScores.Add(score)
	}
	if totalScores.IsZero() {
		return []*types.PartyContributionScore{}
	}

	for _, pcs := range ps {
		pcs.Score = pcs.Score.Mul(teamScore.Score).Div(totalScores)
	}
	return ps
}
