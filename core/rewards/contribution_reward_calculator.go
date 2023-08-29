// Copyright (c) 2022 Gobalsky Labs Limited
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

package rewards

import (
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
)

// given party contribution scores, reward multipliers and distribution strategy calculate the payout per party.
func calculateRewardsByContributionIndividual(epochSeq, asset, accountID string, balance *num.Uint, partyContribution []*types.PartyContibutionScore, rewardFactors map[string]num.Decimal, timestamp time.Time, ds *vega.DispatchStrategy) *payout {
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

	var partyScores []*types.PartyContibutionScore
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
func calculateRewardsByContributionTeam(epochSeq, asset, accountID string, balance *num.Uint, teamContribution []*types.PartyContibutionScore, teamPartyContribution map[string][]*types.PartyContibutionScore, rewardFactors map[string]num.Decimal, timestamp time.Time, ds *vega.DispatchStrategy) *payout {
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

	var teamScores []*types.PartyContibutionScore
	if ds.DistributionStrategy == vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA {
		teamScores = proRataRewardCalculator(teamContribution, map[string]num.Decimal{})
	} else if ds.DistributionStrategy == vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK {
		teamScores = rankingRewardCalculator(teamContribution, ds.RankTable, map[string]num.Decimal{})
	}

	partyScores := []*types.PartyContibutionScore{}
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

func calcPartyInTeamRewardShare(teamScore *types.PartyContibutionScore, partyToMetricScore []*types.PartyContibutionScore, rewardFactors map[string]num.Decimal) []*types.PartyContibutionScore {
	ps := make([]*types.PartyContibutionScore, 0, len(partyToMetricScore))

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
		ps = append(ps, &types.PartyContibutionScore{Party: pcs.Party, Score: score})
		totalScores = totalScores.Add(score)
	}
	if totalScores.IsZero() {
		return []*types.PartyContibutionScore{}
	}

	for _, pcs := range ps {
		pcs.Score = pcs.Score.Mul(teamScore.Score).Div(totalScores)
	}
	return ps
}
