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

	rewardMultipliers := map[string]num.Decimal{"p11": num.DecimalFromFloat(2.5), "p12": num.DecimalFromFloat(3), "p22": num.DecimalFromFloat(1.5), "p32": num.DecimalFromInt64(4), "p41": num.DecimalFromFloat(2.5), "p51": num.DecimalFromInt64(6)}

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

	// p11 = 0.2 * 2.5 = 0.5
	// p12 = 0.5 * 3 = 1.5
	// =====================
	// s11 = 0.4 * 0.25 = 0.1 * 10000 = 1000
	// s12 = 0.4 * 0.75 = 0.3 * 10000 = 3000

	// p21 = 0.05 = 0.05
	// p22 = 0.3 * 1.5 = 0.45
	// =====================
	// s21 = 0.2 * 0.1 = 0.02 * 10000 = 200
	// s22 = 0.2 * 0.9 = 0.18 * 10000 = 1800

	// p41 = 0.2 = 0.2
	// =====================
	// s41 = 0.4 * 10000 = 4000
	require.Equal(t, "asset", po.asset)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, "accountID", po.fromAccount)
	require.Equal(t, uint64(2), po.lockedForEpochs)
	require.Equal(t, now.Unix(), po.timestamp)
	require.Equal(t, "1000", po.partyToAmount["p11"].String())
	require.Equal(t, "3000", po.partyToAmount["p12"].String())
	require.Equal(t, "200", po.partyToAmount["p21"].String())
	require.Equal(t, "1800", po.partyToAmount["p22"].String())
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

	rewardMultipliers := map[string]num.Decimal{"p11": num.DecimalFromFloat(2.5), "p12": num.DecimalFromFloat(3), "p22": num.DecimalFromFloat(1.5), "p32": num.DecimalFromInt64(4), "p41": num.DecimalFromFloat(2.5), "p51": num.DecimalFromInt64(6)}

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

	// p11 = 0.2 * 2.5 = 0.5
	// p12 = 0.5 * 3 = 1.5
	// =====================
	// s11 = 0.3 * 0.25 = 0.075 * 10000 = 750
	// s12 = 0.3 * 0.75 = 0.225 * 10000 = 2250

	// p21 = 0.05 = 0.05
	// p22 = 0.3 * 1.5 = 0.45
	// =====================
	// s21 = 0.25 * 0.1 = 0.025 * 10000 = 250
	// s22 = 0.25 * 0.9 = 0.225 * 10000 = 2250

	// p31 = 0.2 = 0.2
	// p32 = 0.3 * 4 = 1.2
	// p33 = 0.6 = 0.6
	// =====================
	// s31 = 0.05 * 0.1 = 0.005 * 10000 = 50
	// s32 = 0.05 * 0.6 = 0.03 * 10000 = 300
	// s32 = 0.05 * 0.3 = 0.015 * 10000 = 150

	// p41 = 0.2 = 0.2
	// =====================
	// s41 = 0.3 * 10000 = 3000

	// p51 = 0.2 * 6 = 1.2
	// p52 = 0.8
	// =====================
	// s51 = 0.1 * 0.6 = 0.06 * 10000 = 600
	// s52 = 0.1 * 0.4 = 0.04 * 10000 = 400

	require.Equal(t, "asset", po.asset)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, "accountID", po.fromAccount)
	require.Equal(t, uint64(2), po.lockedForEpochs)
	require.Equal(t, now.Unix(), po.timestamp)
	require.Equal(t, "750", po.partyToAmount["p11"].String())
	require.Equal(t, "2250", po.partyToAmount["p12"].String())
	require.Equal(t, "250", po.partyToAmount["p21"].String())
	require.Equal(t, "2250", po.partyToAmount["p22"].String())
	require.Equal(t, "50", po.partyToAmount["p31"].String())
	require.Equal(t, "300", po.partyToAmount["p32"].String())
	require.Equal(t, "150", po.partyToAmount["p33"].String())
	require.Equal(t, "3000", po.partyToAmount["p41"].String())
	require.Equal(t, "600", po.partyToAmount["p51"].String())
	require.Equal(t, "400", po.partyToAmount["p52"].String())
	require.Equal(t, "10000", po.totalReward.String())
}
