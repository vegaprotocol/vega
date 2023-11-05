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
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/require"
)

func TestCalculateRewardsByContributionIndividualProRata(t *testing.T) {
	partyContribution := []*types.PartyContributionScore{
		{Party: "p1", Score: num.DecimalFromFloat(0.6)},
		{Party: "p2", Score: num.DecimalFromFloat(0.5)},
		{Party: "p3", Score: num.DecimalFromFloat(0.1)},
		{Party: "p4", Score: num.DecimalFromFloat(0.6)},
		{Party: "p5", Score: num.DecimalFromFloat(0.05)},
	}
	rewardMultipliers := map[string]num.Decimal{"p2": num.DecimalFromFloat(2.5), "p3": num.DecimalFromInt64(5), "p4": num.DecimalFromFloat(2.5), "p5": num.DecimalFromInt64(3)}

	now := time.Now()
	ds := &vega.DispatchStrategy{
		DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
		LockPeriod:           2,
	}
	po := calculateRewardsByContributionIndividual("1", "asset", "accountID", num.NewUint(10000), partyContribution, rewardMultipliers, now, ds)

	require.Equal(t, "1500", po.partyToAmount["p1"].String())
	require.Equal(t, "3125", po.partyToAmount["p2"].String())
	require.Equal(t, "1250", po.partyToAmount["p3"].String())
	require.Equal(t, "3750", po.partyToAmount["p4"].String())
	require.Equal(t, "375", po.partyToAmount["p5"].String())
	require.Equal(t, "asset", po.asset)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, "accountID", po.fromAccount)
	require.Equal(t, uint64(2), po.lockedForEpochs)
	require.Equal(t, now.Unix(), po.timestamp)
	require.Equal(t, "10000", po.totalReward.String())
}

func TestCalculateRewardsByContributionIndividualRanking(t *testing.T) {
	partyContribution := []*types.PartyContributionScore{
		{Party: "p1", Score: num.DecimalFromFloat(0.6)},
		{Party: "p2", Score: num.DecimalFromFloat(0.5)},
		{Party: "p3", Score: num.DecimalFromFloat(0.1)},
		{Party: "p4", Score: num.DecimalFromFloat(0.6)},
		{Party: "p5", Score: num.DecimalFromFloat(0.05)},
	}

	rewardMultipliers := map[string]num.Decimal{"p1": num.DecimalFromInt64(2), "p2": num.DecimalFromInt64(4)}

	now := time.Now()
	ds := &vega.DispatchStrategy{
		DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK,
		LockPeriod:           2,
		RankTable: []*vega.Rank{
			{StartRank: 1, ShareRatio: 10},
			{StartRank: 2, ShareRatio: 5},
			{StartRank: 4, ShareRatio: 0},
		},
	}
	po := calculateRewardsByContributionIndividual("1", "asset", "accountID", num.NewUint(10000), partyContribution, rewardMultipliers, now, ds)

	require.Equal(t, 3, len(po.partyToAmount))
	require.Equal(t, "4000", po.partyToAmount["p1"].String())
	require.Equal(t, "4000", po.partyToAmount["p2"].String())
	require.Equal(t, "2000", po.partyToAmount["p4"].String())
	require.Equal(t, "asset", po.asset)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, "accountID", po.fromAccount)
	require.Equal(t, uint64(2), po.lockedForEpochs)
	require.Equal(t, now.Unix(), po.timestamp)
	require.Equal(t, "10000", po.totalReward.String())
}

func TestCalculateRewardsByContributionTeamsRank(t *testing.T) {
	teamContribution := []*types.PartyContributionScore{
		{Party: "t1", Score: num.DecimalFromFloat(0.6)},
		{Party: "t2", Score: num.DecimalFromFloat(0.5)},
		{Party: "t3", Score: num.DecimalFromFloat(0.1)},
		{Party: "t4", Score: num.DecimalFromFloat(0.6)},
		{Party: "t5", Score: num.DecimalFromFloat(0.2)},
	}

	t1PartyContribution := []*types.PartyContributionScore{
		{Party: "p11", Score: num.DecimalFromFloat(0.2)},
		{Party: "p12", Score: num.DecimalFromFloat(0.5)},
	}

	t2PartyContribution := []*types.PartyContributionScore{
		{Party: "p21", Score: num.DecimalFromFloat(0.05)},
		{Party: "p22", Score: num.DecimalFromFloat(0.3)},
	}

	t3PartyContribution := []*types.PartyContributionScore{
		{Party: "p31", Score: num.DecimalFromFloat(0.2)},
		{Party: "p32", Score: num.DecimalFromFloat(0.3)},
		{Party: "p33", Score: num.DecimalFromFloat(0.6)},
	}
	t4PartyContribution := []*types.PartyContributionScore{
		{Party: "p41", Score: num.DecimalFromFloat(0.2)},
	}
	t5PartyContribution := []*types.PartyContributionScore{
		{Party: "p51", Score: num.DecimalFromFloat(0.2)},
		{Party: "p52", Score: num.DecimalFromFloat(0.8)},
	}

	teamToPartyContribution := map[string][]*types.PartyContributionScore{
		"t1": t1PartyContribution,
		"t2": t2PartyContribution,
		"t3": t3PartyContribution,
		"t4": t4PartyContribution,
		"t5": t5PartyContribution,
	}

	rewardMultipliers := map[string]num.Decimal{"p11": num.DecimalFromFloat(2), "p12": num.DecimalFromFloat(3), "p22": num.DecimalFromFloat(1.5), "p32": num.DecimalFromInt64(4), "p41": num.DecimalFromFloat(2.5), "p51": num.DecimalFromInt64(6)}

	now := time.Now()
	ds := &vega.DispatchStrategy{
		DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK,
		LockPeriod:           2,
		RankTable: []*vega.Rank{
			{StartRank: 1, ShareRatio: 10},
			{StartRank: 2, ShareRatio: 5},
			{StartRank: 4, ShareRatio: 0},
		},
	}
	po := calculateRewardsByContributionTeam("1", "asset", "accountID", num.NewUint(10000), teamContribution, teamToPartyContribution, rewardMultipliers, now, ds)

	// t1: 0.4
	// t2: 0.2
	// t4: 0.4

	// r11 = 2
	// r12 = 3
	// =====================
	// s11 = 0.4 * 0.4 = 0.24 * 10000 = 1600
	// s12 = 0.4 * 0.6 = 0.16 * 10000 = 2400

	// r21 = 1
	// r22 = 1.5
	// =====================
	// s21 = 0.2 * 0.4 = 0.08 * 10000 = 800
	// s22 = 0.2 * 0.6 = 0.12 * 10000 = 1200

	// p41 = 1
	// =====================
	// s41 = 0.4 * 10000 = 4000
	require.Equal(t, "asset", po.asset)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, "accountID", po.fromAccount)
	require.Equal(t, uint64(2), po.lockedForEpochs)
	require.Equal(t, now.Unix(), po.timestamp)
	require.Equal(t, "1600", po.partyToAmount["p11"].String())
	require.Equal(t, "2400", po.partyToAmount["p12"].String())
	require.Equal(t, "800", po.partyToAmount["p21"].String())
	require.Equal(t, "1200", po.partyToAmount["p22"].String())
	require.Equal(t, "4000", po.partyToAmount["p41"].String())
	require.Equal(t, "10000", po.totalReward.String())
}

func TestCalculateRewardsByContributionTeamsProRata(t *testing.T) {
	teamContribution := []*types.PartyContributionScore{
		{Party: "t1", Score: num.DecimalFromFloat(0.6)},
		{Party: "t2", Score: num.DecimalFromFloat(0.5)},
		{Party: "t3", Score: num.DecimalFromFloat(0.1)},
		{Party: "t4", Score: num.DecimalFromFloat(0.6)},
		{Party: "t5", Score: num.DecimalFromFloat(0.2)},
	}

	t1PartyContribution := []*types.PartyContributionScore{
		{Party: "p11", Score: num.DecimalFromFloat(0.2)},
		{Party: "p12", Score: num.DecimalFromFloat(0.5)},
	}

	t2PartyContribution := []*types.PartyContributionScore{
		{Party: "p21", Score: num.DecimalFromFloat(0.05)},
		{Party: "p22", Score: num.DecimalFromFloat(0.3)},
	}

	t3PartyContribution := []*types.PartyContributionScore{
		{Party: "p31", Score: num.DecimalFromFloat(0.2)},
		{Party: "p32", Score: num.DecimalFromFloat(0.3)},
		{Party: "p33", Score: num.DecimalFromFloat(0.6)},
	}
	t4PartyContribution := []*types.PartyContributionScore{
		{Party: "p41", Score: num.DecimalFromFloat(0.2)},
	}
	t5PartyContribution := []*types.PartyContributionScore{
		{Party: "p51", Score: num.DecimalFromFloat(0.2)},
		{Party: "p52", Score: num.DecimalFromFloat(0.8)},
	}

	teamToPartyContribution := map[string][]*types.PartyContributionScore{
		"t1": t1PartyContribution,
		"t2": t2PartyContribution,
		"t3": t3PartyContribution,
		"t4": t4PartyContribution,
		"t5": t5PartyContribution,
	}

	rewardMultipliers := map[string]num.Decimal{"p11": num.DecimalFromFloat(2), "p12": num.DecimalFromFloat(3), "p22": num.DecimalFromFloat(1.5), "p32": num.DecimalFromInt64(3), "p41": num.DecimalFromFloat(2.5), "p51": num.DecimalFromInt64(7)}

	now := time.Now()
	ds := &vega.DispatchStrategy{
		DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
		LockPeriod:           2,
	}

	po := calculateRewardsByContributionTeam("1", "asset", "accountID", num.NewUint(10000), teamContribution, teamToPartyContribution, rewardMultipliers, now, ds)

	// t1: 0.6/2 = 0.3
	// t2: 0.5/2 = 0.25
	// t3: 0.1/2 = 0.05
	// t4: 0.6/2 = 0.3
	// t5: 0.2/2 = 0.1

	// r11 = 2 = 0.4
	// r12 = 3 = 0.6
	// =====================
	// s11 = 0.3 * 0.4 = 0.12 * 10000 = 1200
	// s12 = 0.3 * 0.6 = 0.18 * 10000 = 1800

	// r21 = 1
	// r22 = 1.5
	// =====================
	// s21 = 0.25 * 0.4 = 0.1 * 10000 = 1000
	// s22 = 0.25 * 0.5 = 0.15 * 10000 = 1500

	// r31 = 1
	// r32 = 3
	// r33 = 1
	// =====================
	// s31 = 0.05 * 0.2 = 0.01 * 10000 = 100
	// s32 = 0.05 * 0.6 = 0.03 * 10000 = 300
	// s32 = 0.05 * 0.2 = 0.01 * 10000 = 100

	// r41 = 2.5
	// =====================
	// s41 = 0.3 * 10000 = 3000

	// r51 = 6
	// r52 = 1
	// =====================
	// s51 = 0.1 * 0.875 = 0.0875 * 10000 = 875
	// s52 = 0.1 * 0.125 = 0.0125 * 10000 = 125

	require.Equal(t, "asset", po.asset)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, "accountID", po.fromAccount)
	require.Equal(t, uint64(2), po.lockedForEpochs)
	require.Equal(t, now.Unix(), po.timestamp)
	require.Equal(t, "1200", po.partyToAmount["p11"].String())
	require.Equal(t, "1800", po.partyToAmount["p12"].String())
	require.Equal(t, "1000", po.partyToAmount["p21"].String())
	require.Equal(t, "1500", po.partyToAmount["p22"].String())
	require.Equal(t, "100", po.partyToAmount["p31"].String())
	require.Equal(t, "300", po.partyToAmount["p32"].String())
	require.Equal(t, "100", po.partyToAmount["p33"].String())
	require.Equal(t, "3000", po.partyToAmount["p41"].String())
	require.Equal(t, "875", po.partyToAmount["p51"].String())
	require.Equal(t, "125", po.partyToAmount["p52"].String())
	require.Equal(t, "10000", po.totalReward.String())
}
