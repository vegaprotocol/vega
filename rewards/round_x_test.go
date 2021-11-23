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
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 2)
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.DecimalFromFloat(5))
	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.DecimalZero())

	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	// start with 10 VEGA
	rewardBalance, _ := num.UintFromString("10000000000000000000", 10)
	err := testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], rewardBalance)
	require.Nil(t, err)

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

		payouts := engine.calculateRewardPayouts(context.Background(), epoch)
		res := payouts[0]

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

// test a 10000 rounds of reward distribution, make sure that the ratio the delegators get out of the reward balance for the epoch is always the same and doesn't drift.
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
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 2)
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(5))
	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.DecimalZero())

	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		PubKey:            "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(6000),
		Delegators:        delegatorForVal1,
	}
	validatorData := []*types.ValidatorData{validator1}

	// start with 10 VEGA reward balance
	rewardBalance, _ := num.UintFromString("10000000000000000000", 10)
	err := testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], rewardBalance)
	require.Nil(t, err)

	// run for 10000 epochs and verify that the ratio each party gets remains constant
	for round := uint64(0); round < 10000; round++ {
		epoch := types.Epoch{Seq: round}
		rewardForEpoch, _ := rs.GetReward(rewardBalance, epoch)
		println(round, rewardForEpoch.String())
		testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(validatorData)

		payouts := engine.calculateRewardPayouts(context.Background(), epoch)
		res := payouts[0]
		require.Equal(t, 2, len(res.partyToAmount))

		party1, _ := res.partyToAmount["party1"].ToDecimal().Div(rewardForEpoch.ToDecimal()).Float64()
		node1, _ := res.partyToAmount["node1"].ToDecimal().Div(rewardForEpoch.ToDecimal()).Float64()
		require.True(t, party1-0.883 < tolerance)
		require.True(t, node1-0.117 < tolerance)
		rewardBalance.Sub(rewardBalance, res.totalReward)
	}
}

// the bug this is reproducing is that sometimes the weights of the delegations of each delegator in a validator come to slightly more than 1 (in this case 1.0000000000000001).
func TestReproBug4220(t *testing.T) {
	for ct := 0; ct < 100; ct++ {
		testEngine := getEngine(t)
		engine := testEngine.engine
		engine.registerStakingAndDelegationRewardScheme()
		engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), 0.883)
		engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")
		engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
		engine.UpdateMaxPayoutPerEpochStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(100000000000000000000))
		engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), 1.1)
		engine.UpdatePayoutFractionForStakingRewardScheme(context.Background(), 0.1)
		engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 2)
		engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(5))
		engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.DecimalZero())
		engine.OnEpochEvent(context.Background(), types.Epoch{})

		rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

		delegatorForVal11 := map[string]*num.Uint{}
		delegatorForVal11["8696ea4067a708fc0d65a6989bc8e23ed0f4e34019586c1fbfad4b709d79dde2"] = num.NewUint(4000000000000000000)
		validator11 := &types.ValidatorData{
			NodeID:            "7d46981bb0b901ae471f3f592004ffcef667a4bd3a0c0b5d2b5346ba41f05a1f",
			PubKey:            "7d46981bb0b901ae471f3f592004ffcef667a4bd3a0c0b5d2b5346ba41f05a1f",
			SelfStake:         num.Zero(),
			StakeByDelegators: num.NewUint(4000000000000000000),
			Delegators:        delegatorForVal11,
		}

		validatorData1 := []*types.ValidatorData{validator11}

		delegatorForVal12 := map[string]*num.Uint{}
		delegatorForVal12["8696ea4067a708fc0d65a6989bc8e23ed0f4e34019586c1fbfad4b709d79dde2"] = num.NewUint(4000000000000000000)
		delegatorForVal12["8af6254cc39f67ec1564e8a179fd5ab2c0e7cd94bc5d0c1a6a14d4f154ae1996"] = num.NewUint(800000000000000000)
		validator12 := &types.ValidatorData{
			NodeID:            "7d46981bb0b901ae471f3f592004ffcef667a4bd3a0c0b5d2b5346ba41f05a1f",
			PubKey:            "7d46981bb0b901ae471f3f592004ffcef667a4bd3a0c0b5d2b5346ba41f05a1f",
			SelfStake:         num.Zero(),
			StakeByDelegators: num.NewUint(4800000000000000000),
			Delegators:        delegatorForVal12,
		}

		validatorData2 := []*types.ValidatorData{validator12}

		delegatorForVal13 := map[string]*num.Uint{}
		delegatorForVal13["8696ea4067a708fc0d65a6989bc8e23ed0f4e34019586c1fbfad4b709d79dde2"] = num.NewUint(4000000000000000000)
		delegatorForVal13["8af6254cc39f67ec1564e8a179fd5ab2c0e7cd94bc5d0c1a6a14d4f154ae1996"] = num.NewUint(800000000000000000)
		delegatorForVal13["e77492db04301678115c19086caf3982e9b8045be5be2d9bd09f85fb601bcb7c"] = num.NewUint(160000000000000000)

		validator13 := &types.ValidatorData{
			NodeID:            "7d46981bb0b901ae471f3f592004ffcef667a4bd3a0c0b5d2b5346ba41f05a1f",
			PubKey:            "7d46981bb0b901ae471f3f592004ffcef667a4bd3a0c0b5d2b5346ba41f05a1f",
			SelfStake:         num.Zero(),
			StakeByDelegators: num.NewUint(4960000000000000000),
			Delegators:        delegatorForVal13,
		}

		validatorData3 := []*types.ValidatorData{validator13}

		delegatorForVal14 := map[string]*num.Uint{}
		delegatorForVal14["8696ea4067a708fc0d65a6989bc8e23ed0f4e34019586c1fbfad4b709d79dde2"] = num.NewUint(4000000000000000000)
		delegatorForVal14["8af6254cc39f67ec1564e8a179fd5ab2c0e7cd94bc5d0c1a6a14d4f154ae1996"] = num.NewUint(800000000000000000)
		delegatorForVal14["e77492db04301678115c19086caf3982e9b8045be5be2d9bd09f85fb601bcb7c"] = num.NewUint(160000000000000000)
		delegatorForVal14["7f5897164da5bc6db8f05bc04a7b0d0b6eb689d3c80526e95cd2023bfd619ad9"] = num.NewUint(32000000000000000)

		validator14 := &types.ValidatorData{
			NodeID:            "7d46981bb0b901ae471f3f592004ffcef667a4bd3a0c0b5d2b5346ba41f05a1f",
			PubKey:            "7d46981bb0b901ae471f3f592004ffcef667a4bd3a0c0b5d2b5346ba41f05a1f",
			SelfStake:         num.Zero(),
			StakeByDelegators: num.NewUint(4992000000000000000),
			Delegators:        delegatorForVal14,
		}

		validatorData4 := []*types.ValidatorData{validator14}

		baseRewardIncrement, _ := num.UintFromString("100000000000000000000", 10)
		testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], baseRewardIncrement)

		var res *payout
		for round := uint64(187); round < 370; round++ {
			if round == 319 || round == 350 || round == 365 {
				testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], baseRewardIncrement)
			}
			epoch := types.Epoch{Seq: round}
			validatorDataToReturn := validatorData1
			if round > 368 {
				validatorDataToReturn = validatorData4
			} else if round > 353 {
				validatorDataToReturn = validatorData3
			} else if round > 322 {
				validatorDataToReturn = validatorData2
			}
			testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(validatorDataToReturn)

			payouts := engine.calculateRewardPayouts(context.Background(), epoch)
			res = payouts[0]

			println("reward amount for validator 7d46981bb0b901ae471f3f592004ffcef667a4bd3a0c0b5d2b5346ba41f05a1f for round", round, res.partyToAmount["7d46981bb0b901ae471f3f592004ffcef667a4bd3a0c0b5d2b5346ba41f05a1f"].String())
			println("reward amount for party 8696ea4067a708fc0d65a6989bc8e23ed0f4e34019586c1fbfad4b709d79dde2 for round", round, res.partyToAmount["8696ea4067a708fc0d65a6989bc8e23ed0f4e34019586c1fbfad4b709d79dde2"].String())
			if round > 322 {
				println("reward amount for party 8af6254cc39f67ec1564e8a179fd5ab2c0e7cd94bc5d0c1a6a14d4f154ae1996 for round", round, res.partyToAmount["8af6254cc39f67ec1564e8a179fd5ab2c0e7cd94bc5d0c1a6a14d4f154ae1996"].String())
			}
			if round > 353 {
				println("reward amount for party e77492db04301678115c19086caf3982e9b8045be5be2d9bd09f85fb601bcb7c for round", round, res.partyToAmount["e77492db04301678115c19086caf3982e9b8045be5be2d9bd09f85fb601bcb7c"].String())
			}
		}

		expectedTotalRewardDistributed, _ := num.UintFromString("7963389516750398902", 10)

		actualTotalRewardDistributed := num.Zero().AddSum(
			res.partyToAmount["7d46981bb0b901ae471f3f592004ffcef667a4bd3a0c0b5d2b5346ba41f05a1f"],
			res.partyToAmount["7f5897164da5bc6db8f05bc04a7b0d0b6eb689d3c80526e95cd2023bfd619ad9"],
			res.partyToAmount["8696ea4067a708fc0d65a6989bc8e23ed0f4e34019586c1fbfad4b709d79dde2"],
			res.partyToAmount["8af6254cc39f67ec1564e8a179fd5ab2c0e7cd94bc5d0c1a6a14d4f154ae1996"],
			res.partyToAmount["e77492db04301678115c19086caf3982e9b8045be5be2d9bd09f85fb601bcb7c"])

		println("actualTotalRewardDistributed", actualTotalRewardDistributed.String())

		require.True(t, expectedTotalRewardDistributed.Sub(expectedTotalRewardDistributed, actualTotalRewardDistributed).LTE(num.NewUint(2)))
	}
}
