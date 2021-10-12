package rewards

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

const tolerance = 1e-12

func TestRepeatedRounds(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), 0.3)
	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateMaxPayoutPerEpochStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(1000000000))
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), 1.1)
	engine.UpdatePayoutFractionForStakingRewardScheme(context.Background(), 0.1)
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	//start with 10 VEGA
	rewardBalance, _ := num.UintFromString("10000000000000000000", 10)
	expectedParty1Ratio := 0.104571
	expectedParty2Ratio := 0.024
	expectedNode1Ratio := 0.14
	expectedNode2Ratio := 0.4
	expectedNode3Ratio := 0.331428

	// run for 10000 epochs and verify that the ratio each party gets remains constant
	for round := uint64(0); round < 10000; round++ {
		epoch := types.Epoch{Seq: round}
		rewardForEpoch, _ := rs.GetReward(rewardBalance, epoch)
		println(round, rewardForEpoch.String())
		testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(testEngine.validatorData)

		res := engine.calculateRewards(context.Background(), "ETH", rs.RewardPoolAccountIDs[0], rs, rewardForEpoch, epoch)
		// node1, node2, node3, party1, party2
		require.Equal(t, 5, len(res.partyToAmount))

		party1Ratio, _ := res.partyToAmount["party1"].ToDecimal().Div(rewardForEpoch.ToDecimal()).Float64()
		party2Ratio, _ := res.partyToAmount["party2"].ToDecimal().Div(rewardForEpoch.ToDecimal()).Float64()
		node1Ratio, _ := res.partyToAmount["node1"].ToDecimal().Div(rewardForEpoch.ToDecimal()).Float64()
		node2Ratio, _ := res.partyToAmount["node2"].ToDecimal().Div(rewardForEpoch.ToDecimal()).Float64()
		node3Ratio, _ := res.partyToAmount["node3"].ToDecimal().Div(rewardForEpoch.ToDecimal()).Float64()

		if !res.partyToAmount["party1"].IsZero() {
			require.True(t, party1Ratio-expectedParty1Ratio < tolerance)
		}
		if !res.partyToAmount["party2"].IsZero() {
			require.True(t, party2Ratio-expectedParty2Ratio < tolerance)
		}
		if !res.partyToAmount["node1"].IsZero() {
			require.True(t, node1Ratio-expectedNode1Ratio < tolerance)
		}
		if !res.partyToAmount["node2"].IsZero() {
			require.True(t, node2Ratio-expectedNode2Ratio < tolerance)
		}
		if !res.partyToAmount["node3"].IsZero() {
			require.True(t, node3Ratio-expectedNode3Ratio < tolerance)
		}
		rewardBalance.Sub(rewardBalance, res.totalReward)
	}
}

// test a 10000 rounds of reward distribution, make sure that the ratio the delegators get out of the reward balance for the epoch is always the same and doesn't drift
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

	// start with 10 VEGA reward balance
	rewardBalance, _ := num.UintFromString("10000000000000000000", 10)

	// run for 10000 epochs and verify that the ratio each party gets remains constant
	for round := uint64(0); round < 10000; round++ {
		epoch := types.Epoch{Seq: round}
		rewardForEpoch, _ := rs.GetReward(rewardBalance, epoch)
		println(round, rewardForEpoch.String())
		testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(validatorData)

		res := engine.calculateRewards(context.Background(), "ETH", rs.RewardPoolAccountIDs[0], rs, rewardForEpoch, epoch)
		require.Equal(t, 2, len(res.partyToAmount))

		party1, _ := res.partyToAmount["party1"].ToDecimal().Div(rewardForEpoch.ToDecimal()).Float64()
		node1, _ := res.partyToAmount["node1"].ToDecimal().Div(rewardForEpoch.ToDecimal()).Float64()
		require.True(t, party1-0.883 < tolerance)
		require.True(t, node1-0.117 < tolerance)
		rewardBalance.Sub(rewardBalance, res.totalReward)
	}

}
