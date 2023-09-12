package rewards

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/stretchr/testify/require"
)

func TestFindRank(t *testing.T) {
	rankingTable := []*vega.Rank{
		{StartRank: 1, ShareRatio: 10},
		{StartRank: 2, ShareRatio: 5},
		{StartRank: 4, ShareRatio: 2},
		{StartRank: 10, ShareRatio: 1},
		{StartRank: 20, ShareRatio: 0},
	}
	require.Equal(t, uint32(10), findRank(rankingTable, 1))
	require.Equal(t, uint32(5), findRank(rankingTable, 2))
	require.Equal(t, uint32(5), findRank(rankingTable, 3))
	require.Equal(t, uint32(2), findRank(rankingTable, 4))
	require.Equal(t, uint32(2), findRank(rankingTable, 5))
	require.Equal(t, uint32(2), findRank(rankingTable, 6))
	require.Equal(t, uint32(1), findRank(rankingTable, 10))
	require.Equal(t, uint32(1), findRank(rankingTable, 15))
	require.Equal(t, uint32(0), findRank(rankingTable, 20))
	require.Equal(t, uint32(0), findRank(rankingTable, 21))
}

func TestRankingRewardCalculator(t *testing.T) {
	rankingTable := []*vega.Rank{
		{StartRank: 1, ShareRatio: 10},
		{StartRank: 2, ShareRatio: 5},
		{StartRank: 4, ShareRatio: 0},
	}

	require.Equal(t, 0, len(rankingRewardCalculator([]*types.PartyContributionScore{}, rankingTable, map[string]num.Decimal{})))

	partyContribution := []*types.PartyContributionScore{
		{Party: "p1", Score: num.DecimalFromFloat(0.6)},
		{Party: "p2", Score: num.DecimalFromFloat(0.5)},
		{Party: "p3", Score: num.DecimalFromFloat(0.1)},
		{Party: "p4", Score: num.DecimalFromFloat(0.6)},
		{Party: "p5", Score: num.DecimalFromFloat(0.05)},
	}

	rankingResult := rankingRewardCalculator(partyContribution, rankingTable, map[string]num.Decimal{"p1": num.DecimalFromInt64(2), "p2": num.DecimalFromInt64(4)})
	// p1 is at the 1st rank => share ratio = 10. reward_multiplier = 2 => d_i = 20 => s_i = 20/50 = 0.4
	// p4 is at the 1st rank => share ratio = 10. reward_multiplier = 1 => d_i = 10 => 2_i = 10/50 = 0.2
	// p2 is at the 3rd rank => share ratio = 5. reward_multiplier = 4 => d_i => 20 => s_i = 20/50 = 0.4
	// p3 is at the 4th rank => share ratio = 0
	// p5 is at the 5th rank => share ratio = 0
	require.Equal(t, 3, len(rankingResult))
	require.Equal(t, "p1", rankingResult[0].Party)
	require.Equal(t, "0.4", rankingResult[0].Score.String())
	require.Equal(t, "p4", rankingResult[1].Party)
	require.Equal(t, "0.2", rankingResult[1].Score.String())
	require.Equal(t, "p2", rankingResult[2].Party)
	require.Equal(t, "0.4", rankingResult[2].Score.String())
}

func TestProRataRewardCalculator(t *testing.T) {
	require.Equal(t, 0, len(proRataRewardCalculator([]*types.PartyContributionScore{}, map[string]num.Decimal{})))

	partyContribution := []*types.PartyContributionScore{
		{Party: "p1", Score: num.DecimalFromFloat(0.6)},
		{Party: "p2", Score: num.DecimalFromFloat(0.5)},
		{Party: "p3", Score: num.DecimalFromFloat(0.1)},
		{Party: "p4", Score: num.DecimalFromFloat(0.6)},
		{Party: "p5", Score: num.DecimalFromFloat(0.05)},
	}
	rewardMultipliers := map[string]num.Decimal{"p2": num.DecimalFromFloat(2.5), "p3": num.DecimalFromInt64(5), "p4": num.DecimalFromFloat(2.5), "p5": num.DecimalFromInt64(3)}
	prorataResult := proRataRewardCalculator(partyContribution, rewardMultipliers)
	require.Equal(t, 5, len(prorataResult))

	// p1: 0.6 * 1 = 0.6
	// p2: 0.5 * 2.5 = 1.25
	// p3: 0.1 * 5 = 0.5
	// p4: 0.6 * 2.5  = 1.5
	// p5: 0.05 * 3 = 0.15
	// total = 4
	// s_1 = 0.6/4 = 0.15
	// s_2 = 1.25/4 = 0.3125
	// s_3 = 0.5/4 = 0.125
	// s_4 = 1.5/4 = 0.375
	// s_5 = 0.15/4 = 0.0375
	require.Equal(t, "p1", prorataResult[0].Party)
	require.Equal(t, "0.15", prorataResult[0].Score.String())
	require.Equal(t, "p2", prorataResult[1].Party)
	require.Equal(t, "0.3125", prorataResult[1].Score.String())
	require.Equal(t, "p3", prorataResult[2].Party)
	require.Equal(t, "0.125", prorataResult[2].Score.String())
	require.Equal(t, "p4", prorataResult[3].Party)
	require.Equal(t, "0.375", prorataResult[3].Score.String())
	require.Equal(t, "p5", prorataResult[4].Party)
	require.Equal(t, "0.0375", prorataResult[4].Score.String())
}

func TestCapAtOne(t *testing.T) {
	partyRewardScores := []*types.PartyContributionScore{{Party: "p1", Score: num.DecimalFromFloat(0.1)}, {Party: "p2", Score: num.MustDecimalFromString("0.900000001")}}
	total := num.DecimalFromFloat(0.1).Add(num.MustDecimalFromString("0.900000001"))

	capAtOne(partyRewardScores, total)
	require.Equal(t, "p2", partyRewardScores[0].Party)
	require.Equal(t, "0.9", partyRewardScores[0].Score.String())
	require.Equal(t, "p1", partyRewardScores[1].Party)
	require.Equal(t, "0.1", partyRewardScores[1].Score.String())
}
