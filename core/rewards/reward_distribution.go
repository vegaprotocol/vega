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
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
)

func findRank(rankingTable []*vega.Rank, ind int) uint32 {
	var lastSeen *vega.Rank
	for _, rank := range rankingTable {
		if int(rank.StartRank) > ind {
			break
		}
		lastSeen = rank
	}

	if lastSeen == nil {
		return 0
	}

	return lastSeen.ShareRatio
}

func rankingRewardCalculator(partyMetric []*types.PartyContributionScore, rankingTable []*vega.Rank, partyRewardFactor map[string]num.Decimal) []*types.PartyContributionScore {
	partyScores := []*types.PartyContributionScore{}
	sort.Slice(partyMetric, func(i, j int) bool {
		return partyMetric[i].Score.GreaterThan(partyMetric[j].Score)
	})
	shareRatio := num.DecimalZero()
	totalScores := num.DecimalZero()
	for i, ps := range partyMetric {
		rewardFactor, ok := partyRewardFactor[ps.Party]
		if !ok {
			rewardFactor = num.DecimalOne()
		}
		if i == 0 || !ps.Score.Equal(partyMetric[i-1].Score) {
			shareRatio = num.DecimalFromInt64(int64(findRank(rankingTable, i+1)))
		}
		score := shareRatio.Mul(rewardFactor)
		if shareRatio.IsZero() {
			break
		}
		if score.IsZero() {
			continue
		}
		partyScores = append(partyScores, &types.PartyContributionScore{Party: ps.Party, Score: score})
		totalScores = totalScores.Add(score)
	}
	if totalScores.IsZero() {
		return []*types.PartyContributionScore{}
	}

	normalise(partyScores, totalScores)
	return partyScores
}

func proRataRewardCalculator(partyContribution []*types.PartyContributionScore, partyRewardFactor map[string]num.Decimal) []*types.PartyContributionScore {
	total := num.DecimalZero()
	partiesWithScore := []*types.PartyContributionScore{}
	for _, metric := range partyContribution {
		factor, ok := partyRewardFactor[metric.Party]
		if !ok {
			factor = num.DecimalOne()
		}
		score := factor.Mul(metric.Score)
		if score.IsZero() {
			continue
		}
		total = total.Add(score)
		partiesWithScore = append(partiesWithScore, &types.PartyContributionScore{Party: metric.Party, Score: score})
	}
	if total.IsZero() {
		return []*types.PartyContributionScore{}
	}

	normalise(partiesWithScore, total)
	return partiesWithScore
}

func normalise(partyRewardScores []*types.PartyContributionScore, total num.Decimal) {
	normalisedTotal := num.DecimalZero()
	for _, p := range partyRewardScores {
		p.Score = p.Score.Div(total)
		normalisedTotal = normalisedTotal.Add(p.Score)
	}
	if normalisedTotal.LessThanOrEqual(num.DecimalOne()) {
		return
	}

	capAtOne(partyRewardScores, normalisedTotal)
}

func capAtOne(partyRewardScores []*types.PartyContributionScore, total num.Decimal) {
	if total.LessThanOrEqual(num.DecimalOne()) {
		return
	}

	sort.SliceStable(partyRewardScores, func(i, j int) bool { return partyRewardScores[i].Score.GreaterThan(partyRewardScores[j].Score) })
	delta := total.Sub(num.DecimalFromInt64(1))
	partyRewardScores[0].Score = num.MaxD(num.DecimalZero(), partyRewardScores[0].Score.Sub(delta))
}
