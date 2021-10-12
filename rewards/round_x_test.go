package rewards

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func testRoundXReward(t *testing.T) {
	round := 1

	initialReward, _ := num.UintFromString("133948860171808410010", 10)
	fraction, _ := num.DecimalFromString("0.1")
	delegatorShare, _ := num.DecimalFromString("0.883")

	for ; round < 10; round++ {
		rewardForEpoch, _ := num.UintFromDecimal(initialReward.ToDecimal().Mul(fraction))
		println("reward for epoch", rewardForEpoch.String())

		delegatorReward, _ := num.UintFromDecimal(initialReward.ToDecimal().Mul(fraction).Mul(delegatorShare))
		println("delegator reward for epoch", round, delegatorReward.String())

		validatorReward := num.Zero().Sub(rewardForEpoch, delegatorReward)
		println("validator reward for epoch", round, validatorReward.String())

		initialReward = initialReward.Sub(initialReward, num.Sum(delegatorReward, validatorReward))
	}
}

// test a 100 rounds of reward distribution, make sure that the ratio the delegators get out of the reward balance for the epoch is always the same and doesn't drift
func TestNoDriftdRounds(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), 0.883)
	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateMaxPayoutPerEpochStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(1000000000))
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), 1.1)
	engine.UpdatePayoutFractionForStakingRewardScheme(context.Background(), 0.1)
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(6000),
		Delegators:        delegatorForVal1,
	}
	validatorData := []*types.ValidatorData{validator1}

	rewardBalance := num.NewUint(1000000)
	var round uint64 = 0
	for ; round < 100; round++ {
		epoch := types.Epoch{Seq: round}
		rewardForEpoch, _ := rs.GetReward(rewardBalance, epoch)
		println(round, rewardForEpoch.String())
		testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(validatorData)

		res := engine.calculateRewards(context.Background(), "ETH", rs.RewardPoolAccountIDs[0], rs, rewardForEpoch, epoch)
		require.Equal(t, 2, len(res.partyToAmount))

		party1, _ := res.partyToAmount["party1"].ToDecimal().Div(rewardForEpoch.ToDecimal()).Float64()
		node1, _ := res.partyToAmount["node1"].ToDecimal().Div(rewardForEpoch.ToDecimal()).Float64()
		require.True(t, party1-0.883 < 1e-4)
		require.True(t, node1-0.117 < 1e-4)
		rewardBalance.Sub(rewardBalance, res.totalReward)
	}

}
