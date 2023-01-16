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

package validators

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"github.com/stretchr/testify/require"
)

func TestValidatorSet(t *testing.T) {
	t.Run("test update of multisig signers network parameter", testUpdateNumberEthMultisigSigners)
	t.Run("test update of number of validators network parameter", testUpdateNumberOfTendermintValidators)
	t.Run("test update of incumbent factor network parameter", testUpdateValidatorIncumbentBonusFactor)
	t.Run("test update of min eth events for new validator network parameter", testUpdateMinimumEthereumEventsForNewValidator)
	t.Run("test update of minimum required stake", testUpdateMinimumRequireSelfStake)
	t.Run("test counting of forwarded events for pending validators", testAddForwarder)
	t.Run("test the number of tendermint validators is reduced hence validators being demoted", testTendermintValidatorsNumberReduced)
	t.Run("test the number of tendermint validators is greater than the number of current tm, promotion is available", testTendermintFreeSlotsPromotion)
	t.Run("test swap of the best ersatz with the worst tendermint validators", testSwapBestErsatzWithWorstTendermint)
	t.Run("test the number of ersatz validators is reduced hence validators being demoted", testErsatzValidatorsNumberReduced)
	t.Run("test the number of ersatz validators is greater than the number of current tm, promotion is available", testErsatzFreeSlotsPromotion)
	t.Run("test swap of the best pending with the worst ersatz validators", testSwapBestPendingWithWorstErsatz)
	t.Run("test swap of from ez to tendermint with slot reduction in ersatz", testSwapAndErsatzSlotDecrease)
	t.Run("test swap of from ez to tendermint with slot increase in tendermint", testSwapAndTendermintSlotIncrease)
}

func TestDecreaseNumberOfTendermintValidators(t *testing.T) {
	tm := int64(1654747635)
	rng := rand.New(rand.NewSource(tm))
	byStatusChangeBlock := func(val1, val2 *valState) bool { return val1.statusChangeBlock < val2.statusChangeBlock }
	rankingScore := map[string]num.Decimal{
		"70b29f15c7d3cc430283dfee07e17775f041427749f7f1f8b9979bdde15ae908": num.NewDecimalFromFloat(0.6),
		"db14f8d4e4beebd085b22c7332d8a12d3e3841319ba78542a418c02d7740d117": num.NewDecimalFromFloat(0.3),
		"20a7d70939c3453613b6d0477650f8845a6dbc0e58d2416e0aa5c27500f563b3": num.NewDecimalFromFloat(0.7),
		"4a329b356c4a875077eb5babcc5b7b91f27d75fe35c52a1dc85fe079b9e14066": num.NewDecimalFromFloat(0.2),
		"4dd0e9f844b16777210d2815f81d8cc6f6ecc4f9bf7b895fcee3ab982e5c1ebd": num.DecimalOne(),
	}

	topology := &Topology{}
	topology.UpdateNumberOfTendermintValidators(context.Background(), num.NewUint(3))

	valStates := []*valState{
		{
			data: ValidatorData{
				ID:              "70b29f15c7d3cc430283dfee07e17775f041427749f7f1f8b9979bdde15ae908",
				EthereumAddress: crypto.RandomHash(),
			},
			statusChangeBlock: 1,
			status:            ValidatorStatusTendermint,
		},
		{
			data: ValidatorData{
				ID:              "db14f8d4e4beebd085b22c7332d8a12d3e3841319ba78542a418c02d7740d117",
				EthereumAddress: crypto.RandomHash(),
			},
			statusChangeBlock: 1,
			status:            ValidatorStatusTendermint,
		},
		{
			data: ValidatorData{
				ID:              "20a7d70939c3453613b6d0477650f8845a6dbc0e58d2416e0aa5c27500f563b3",
				EthereumAddress: crypto.RandomHash(),
			},
			statusChangeBlock: 1,
			status:            ValidatorStatusTendermint,
		},
		{
			data: ValidatorData{
				ID:              "4a329b356c4a875077eb5babcc5b7b91f27d75fe35c52a1dc85fe079b9e14066",
				EthereumAddress: crypto.RandomHash(),
			},
			statusChangeBlock: 1,
			status:            ValidatorStatusTendermint,
		},
		{
			data: ValidatorData{
				ID:              "4dd0e9f844b16777210d2815f81d8cc6f6ecc4f9bf7b895fcee3ab982e5c1ebd",
				EthereumAddress: crypto.RandomHash(),
			},
			statusChangeBlock: 1,
			status:            ValidatorStatusTendermint,
		},
	}

	// only 4 of the 5 validators are signers on the bridge
	signers := map[string]struct{}{}
	for _, vs := range valStates {
		signers[vs.data.EthereumAddress] = struct{}{}
	}
	// 4dd0e9f844b16777210d2815f81d8cc6f6ecc4f9bf7b895fcee3ab982e5c1ebd currently having a ranking score of 1 is not a signer
	delete(signers, valStates[len(valStates)-1].data.EthereumAddress)
	sortValidatorDescRankingScoreAscBlockcompare(valStates, rankingScore, byStatusChangeBlock, rng)

	// effectively can't remove any signer from the bridge and all validators are currently signers
	threshold := uint32(999)

	tendermintValidators, remainingValidators, removedFromTM := handleSlotChanges(valStates, []*valState{}, ValidatorStatusTendermint, ValidatorStatusErsatz, 3, int64(3), rankingScore, signers, threshold)
	require.Equal(t, 5, len(tendermintValidators))
	require.Equal(t, 0, len(remainingValidators))
	require.Equal(t, 0, len(removedFromTM))

	count := 0
	for _, valState := range valStates {
		if valState.status == ValidatorStatusTendermint {
			count++
		}
	}
	require.Equal(t, 5, count)

	// let change the ranking score of the validator who is not a signer to be lowest so that it can be removed regardless of the threshold
	rankingScore["4dd0e9f844b16777210d2815f81d8cc6f6ecc4f9bf7b895fcee3ab982e5c1ebd"] = num.DecimalFromFloat(0.1)
	sortValidatorDescRankingScoreAscBlockcompare(valStates, rankingScore, byStatusChangeBlock, rng)

	tendermintValidators, remainingValidators, removedFromTM = handleSlotChanges(valStates, []*valState{}, ValidatorStatusTendermint, ValidatorStatusErsatz, 3, int64(3), rankingScore, signers, threshold)
	require.Equal(t, 4, len(tendermintValidators))
	require.Equal(t, 1, len(remainingValidators))
	require.Equal(t, 1, len(removedFromTM))

	require.Equal(t, "4dd0e9f844b16777210d2815f81d8cc6f6ecc4f9bf7b895fcee3ab982e5c1ebd", removedFromTM[0])
	count = 0
	for _, valState := range valStates {
		if valState.status == ValidatorStatusTendermint {
			count++
		}
	}
	require.Equal(t, 4, count)

	// run for another epoch - non can be decreased anymore with the current threshold
	tendermintValidators, remainingValidators, removedFromTM = handleSlotChanges(tendermintValidators, remainingValidators, ValidatorStatusTendermint, ValidatorStatusErsatz, 3, int64(3), rankingScore, signers, threshold)
	require.Equal(t, 4, len(tendermintValidators))
	require.Equal(t, 1, len(remainingValidators))
	require.Equal(t, 0, len(removedFromTM))

	// change the threshold to 700 - expcect one validator to be removed
	threshold = 700
	tendermintValidators, remainingValidators, removedFromTM = handleSlotChanges(tendermintValidators, remainingValidators, ValidatorStatusTendermint, ValidatorStatusErsatz, 3, int64(3), rankingScore, signers, threshold)
	require.Equal(t, 3, len(tendermintValidators))
	require.Equal(t, 2, len(remainingValidators))
	require.Equal(t, 1, len(removedFromTM))

	require.Equal(t, "4a329b356c4a875077eb5babcc5b7b91f27d75fe35c52a1dc85fe079b9e14066", removedFromTM[0])
	count = 0
	for _, valState := range valStates {
		if valState.status == ValidatorStatusTendermint {
			count++
		}
	}
	require.Equal(t, 3, count)
}

func TestDecreaseNumberOfTendermintValidatorsNotUpdatingContract(t *testing.T) {
	tm := int64(1654747635)
	rng := rand.New(rand.NewSource(tm))
	byStatusChangeBlock := func(val1, val2 *valState) bool { return val1.statusChangeBlock < val2.statusChangeBlock }
	rankingScore := map[string]num.Decimal{
		"70b29f15c7d3cc430283dfee07e17775f041427749f7f1f8b9979bdde15ae908": num.NewDecimalFromFloat(0.6),
		"db14f8d4e4beebd085b22c7332d8a12d3e3841319ba78542a418c02d7740d117": num.NewDecimalFromFloat(0.3),
		"20a7d70939c3453613b6d0477650f8845a6dbc0e58d2416e0aa5c27500f563b3": num.NewDecimalFromFloat(0.7),
		"4a329b356c4a875077eb5babcc5b7b91f27d75fe35c52a1dc85fe079b9e14066": num.NewDecimalFromFloat(0.2),
		"4dd0e9f844b16777210d2815f81d8cc6f6ecc4f9bf7b895fcee3ab982e5c1ebd": num.DecimalOne(),
	}

	topology := &Topology{}
	topology.UpdateNumberOfTendermintValidators(context.Background(), num.NewUint(3))

	valStates := []*valState{
		{
			data: ValidatorData{
				ID:              "70b29f15c7d3cc430283dfee07e17775f041427749f7f1f8b9979bdde15ae908",
				EthereumAddress: crypto.RandomHash(),
			},
			statusChangeBlock: 1,
			status:            ValidatorStatusTendermint,
		},
		{
			data: ValidatorData{
				ID:              "db14f8d4e4beebd085b22c7332d8a12d3e3841319ba78542a418c02d7740d117",
				EthereumAddress: crypto.RandomHash(),
			},
			statusChangeBlock: 1,
			status:            ValidatorStatusTendermint,
		},
		{
			data: ValidatorData{
				ID:              "20a7d70939c3453613b6d0477650f8845a6dbc0e58d2416e0aa5c27500f563b3",
				EthereumAddress: crypto.RandomHash(),
			},
			statusChangeBlock: 1,
			status:            ValidatorStatusTendermint,
		},
		{
			data: ValidatorData{
				ID:              "4a329b356c4a875077eb5babcc5b7b91f27d75fe35c52a1dc85fe079b9e14066",
				EthereumAddress: crypto.RandomHash(),
			},
			statusChangeBlock: 1,
			status:            ValidatorStatusTendermint,
		},
		{
			data: ValidatorData{
				ID:              "4dd0e9f844b16777210d2815f81d8cc6f6ecc4f9bf7b895fcee3ab982e5c1ebd",
				EthereumAddress: crypto.RandomHash(),
			},
			statusChangeBlock: 1,
			status:            ValidatorStatusTendermint,
		},
	}

	signer3Address := valStates[3].data.EthereumAddress

	// starting with 5 signatures on the bridge
	signers := map[string]struct{}{}
	for _, vs := range valStates {
		signers[vs.data.EthereumAddress] = struct{}{}
	}
	sortValidatorDescRankingScoreAscBlockcompare(valStates, rankingScore, byStatusChangeBlock, rng)

	threshold := uint32(666)

	// one validator is removed
	tendermintValidators, remainingValidators, removedFromTM := handleSlotChanges(valStates, []*valState{}, ValidatorStatusTendermint, ValidatorStatusErsatz, 3, int64(3), rankingScore, signers, threshold)
	require.Equal(t, 4, len(tendermintValidators))
	require.Equal(t, 1, len(remainingValidators))
	require.Equal(t, 1, len(removedFromTM))

	count := 0
	for _, valState := range valStates {
		if valState.status == ValidatorStatusTendermint {
			count++
		}
	}
	require.Equal(t, 4, count)

	// we don't update the contract so it still has 5 signers - meaning it should not allow us to remove another validator now
	rankingScore["4dd0e9f844b16777210d2815f81d8cc6f6ecc4f9bf7b895fcee3ab982e5c1ebd"] = num.DecimalFromFloat(0.1)
	sortValidatorDescRankingScoreAscBlockcompare(valStates, rankingScore, byStatusChangeBlock, rng)

	tendermintValidators, remainingValidators, removedFromTM = handleSlotChanges(tendermintValidators, remainingValidators, ValidatorStatusTendermint, ValidatorStatusErsatz, 3, int64(3), rankingScore, signers, threshold)
	require.Equal(t, 4, len(tendermintValidators))
	require.Equal(t, 1, len(remainingValidators))
	require.Equal(t, 0, len(removedFromTM))

	// now update the contract to remove the signer
	delete(signers, signer3Address)
	tendermintValidators, remainingValidators, removedFromTM = handleSlotChanges(tendermintValidators, remainingValidators, ValidatorStatusTendermint, ValidatorStatusErsatz, 3, int64(3), rankingScore, signers, threshold)
	require.Equal(t, 3, len(tendermintValidators))
	require.Equal(t, 2, len(remainingValidators))
	require.Equal(t, 1, len(removedFromTM))
}

func TestApplyPromotionAllThingsEqual(t *testing.T) {
	tm := int64(1654747635)
	for i := 0; i < 100; i++ {
		rng := rand.New(rand.NewSource(tm))
		byStatusChangeBlock := func(val1, val2 *valState) bool { return val1.statusChangeBlock < val2.statusChangeBlock }
		rankingScore := map[string]num.Decimal{
			"70b29f15c7d3cc430283dfee07e17775f041427749f7f1f8b9979bdde15ae908": num.DecimalZero(),
			"db14f8d4e4beebd085b22c7332d8a12d3e3841319ba78542a418c02d7740d117": num.DecimalZero(),
			"20a7d70939c3453613b6d0477650f8845a6dbc0e58d2416e0aa5c27500f563b3": num.DecimalZero(),
			"4a329b356c4a875077eb5babcc5b7b91f27d75fe35c52a1dc85fe079b9e14066": num.DecimalZero(),
			"4dd0e9f844b16777210d2815f81d8cc6f6ecc4f9bf7b895fcee3ab982e5c1ebd": num.DecimalZero(),
		}

		topology := &Topology{}
		topology.UpdateNumberOfTendermintValidators(context.Background(), num.NewUint(3))

		valStates := []*valState{
			{
				data: ValidatorData{
					ID:              "70b29f15c7d3cc430283dfee07e17775f041427749f7f1f8b9979bdde15ae908",
					EthereumAddress: crypto.RandomHash(),
				},
				statusChangeBlock: 1,
				status:            ValidatorStatusTendermint,
			},
			{
				data: ValidatorData{
					ID:              "db14f8d4e4beebd085b22c7332d8a12d3e3841319ba78542a418c02d7740d117",
					EthereumAddress: crypto.RandomHash(),
				},
				statusChangeBlock: 1,
				status:            ValidatorStatusTendermint,
			},
			{
				data: ValidatorData{
					ID:              "20a7d70939c3453613b6d0477650f8845a6dbc0e58d2416e0aa5c27500f563b3",
					EthereumAddress: crypto.RandomHash(),
				},
				statusChangeBlock: 1,
				status:            ValidatorStatusTendermint,
			},
			{
				data: ValidatorData{
					ID:              "4a329b356c4a875077eb5babcc5b7b91f27d75fe35c52a1dc85fe079b9e14066",
					EthereumAddress: crypto.RandomHash(),
				},
				statusChangeBlock: 1,
				status:            ValidatorStatusTendermint,
			},
			{
				data: ValidatorData{
					ID:              "4dd0e9f844b16777210d2815f81d8cc6f6ecc4f9bf7b895fcee3ab982e5c1ebd",
					EthereumAddress: crypto.RandomHash(),
				},
				statusChangeBlock: 1,
				status:            ValidatorStatusTendermint,
			},
		}

		signer1Address := valStates[1].data.EthereumAddress

		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(valStates), func(i, j int) { valStates[i], valStates[j] = valStates[j], valStates[i] })

		sortValidatorDescRankingScoreAscBlockcompare(valStates, rankingScore, byStatusChangeBlock, rng)

		signers := map[string]struct{}{}
		for _, vs := range valStates {
			signers[vs.data.EthereumAddress] = struct{}{}
		}

		tendermintValidators, remainingValidators, removedFromTM := handleSlotChanges(valStates, []*valState{}, ValidatorStatusTendermint, ValidatorStatusErsatz, 3, int64(3), rankingScore, signers, 666)
		require.Equal(t, 4, len(tendermintValidators))
		require.Equal(t, 1, len(remainingValidators))
		require.Equal(t, 1, len(removedFromTM))

		require.Equal(t, "db14f8d4e4beebd085b22c7332d8a12d3e3841319ba78542a418c02d7740d117", removedFromTM[0])
		count := 0
		for _, valState := range valStates {
			if valState.status == ValidatorStatusTendermint {
				count++
			}
		}
		require.Equal(t, 4, count)

		delete(signers, signer1Address)

		tendermintValidators, remainingValidators, removedFromTM = handleSlotChanges(tendermintValidators, []*valState{}, ValidatorStatusTendermint, ValidatorStatusErsatz, 3, int64(3), rankingScore, signers, 666)
		require.Equal(t, 3, len(tendermintValidators))
		require.Equal(t, 1, len(remainingValidators))
		require.Equal(t, 1, len(removedFromTM))

		require.Equal(t, "70b29f15c7d3cc430283dfee07e17775f041427749f7f1f8b9979bdde15ae908", removedFromTM[0])
		count = 0
		for _, valState := range valStates {
			if valState.status == ValidatorStatusTendermint {
				count++
			}
		}
		require.Equal(t, 3, count)
	}
}

func TestSortValidatorDescRankingScoreAscBlockStatusChanged(t *testing.T) {
	tm := int64(1654747635)
	for i := 0; i < 100; i++ {
		rng := rand.New(rand.NewSource(tm))
		byStatusChangeBlock := func(val1, val2 *valState) bool { return val1.statusChangeBlock < val2.statusChangeBlock }
		valStates1 := []*valState{
			{
				data: ValidatorData{
					ID:         "node1",
					VegaPubKey: "node1Key",
				},
				statusChangeBlock: 1,
			},
			{
				data: ValidatorData{
					ID:         "node2",
					VegaPubKey: "node2Key",
				},
				statusChangeBlock: 2,
			},
			{
				data: ValidatorData{
					ID:         "node3",
					VegaPubKey: "node3Key",
				},
				statusChangeBlock: 3,
			},
		}
		rankingScore1 := map[string]num.Decimal{
			"node1": num.DecimalFromFloat(0.9),
			"node2": num.DecimalFromFloat(0.5),
			"node3": num.DecimalFromFloat(0.7),
		}

		// can sort simply by ranking score descending
		sortValidatorDescRankingScoreAscBlockcompare(valStates1, rankingScore1, byStatusChangeBlock, rng)
		require.Equal(t, "node1", valStates1[0].data.ID)
		require.Equal(t, "node3", valStates1[1].data.ID)
		require.Equal(t, "node2", valStates1[2].data.ID)

		valStates2 := []*valState{
			{
				data: ValidatorData{
					ID:         "node1",
					VegaPubKey: "node1Key",
				},
				statusChangeBlock: 1,
			},
			{
				data: ValidatorData{
					ID:         "node2",
					VegaPubKey: "node2Key",
				},
				statusChangeBlock: 2,
			},
			{
				data: ValidatorData{
					ID:         "node3",
					VegaPubKey: "node3Key",
				},
				statusChangeBlock: 3,
			},
		}
		rankingScore2 := map[string]num.Decimal{
			"node1": num.DecimalFromFloat(0.9),
			"node2": num.DecimalFromFloat(0.5),
			"node3": num.DecimalFromFloat(0.5),
		}

		// need to use last block change state sorted ascending as tie breaker - node 2 changed state before node 3 so it comes before it
		sortValidatorDescRankingScoreAscBlockcompare(valStates2, rankingScore2, byStatusChangeBlock, rng)
		require.Equal(t, "node1", valStates2[0].data.ID)
		require.Equal(t, "node2", valStates2[1].data.ID)
		require.Equal(t, "node3", valStates2[2].data.ID)

		valStates3 := []*valState{
			{
				data: ValidatorData{
					ID:         "node1",
					VegaPubKey: "node1Key",
				},
				statusChangeBlock: 1,
			},
			{
				data: ValidatorData{
					ID:         "node2",
					VegaPubKey: "node2Key",
				},
				statusChangeBlock: 1,
			},
			{
				data: ValidatorData{
					ID:         "node3",
					VegaPubKey: "node3Key",
				},
				statusChangeBlock: 1,
			},
			{
				data: ValidatorData{
					ID:         "node4",
					VegaPubKey: "node4Key",
				},
				statusChangeBlock: 1,
			},
		}
		rankingScore3 := map[string]num.Decimal{
			"node1": num.DecimalFromFloat(0.9),
			"node2": num.DecimalFromFloat(0.5),
			"node3": num.DecimalFromFloat(0.5),
			"node4": num.DecimalFromFloat(0.9),
		}

		// need to use last block change state sorted ascending as tie breaker - node 2 changed state before node 3 so it comes before it
		sortValidatorDescRankingScoreAscBlockcompare(valStates3, rankingScore3, byStatusChangeBlock, rng)
		require.Equal(t, "node1", valStates3[0].data.ID)
		require.Equal(t, "node4", valStates3[1].data.ID)
		require.Equal(t, "node2", valStates3[2].data.ID)
		require.Equal(t, "node3", valStates3[3].data.ID)

		valStates4 := []*valState{
			{
				data: ValidatorData{
					ID:         "node1",
					VegaPubKey: "node1Key",
				},
				statusChangeBlock: 1,
			},
			{
				data: ValidatorData{
					ID:         "node2",
					VegaPubKey: "node2Key",
				},
				statusChangeBlock: 1,
			},
			{
				data: ValidatorData{
					ID:         "node3",
					VegaPubKey: "node3Key",
				},
				statusChangeBlock: 1,
			},
			{
				data: ValidatorData{
					ID:         "node4",
					VegaPubKey: "node4Key",
				},
				statusChangeBlock: 1,
			},
		}
		rankingScore4 := map[string]num.Decimal{
			"node1": num.DecimalFromFloat(0.5),
			"node2": num.DecimalFromFloat(0.5),
			"node3": num.DecimalFromFloat(0.5),
			"node4": num.DecimalFromFloat(0.9),
		}
		sortValidatorDescRankingScoreAscBlockcompare(valStates4, rankingScore4, byStatusChangeBlock, rng)
		require.Equal(t, "node4", valStates4[0].data.ID)
		require.Equal(t, "node1", valStates4[1].data.ID)
		require.Equal(t, "node2", valStates4[2].data.ID)
		require.Equal(t, "node3", valStates4[3].data.ID)
	}
}

func testUpdateNumberEthMultisigSigners(t *testing.T) {
	topology := &Topology{}
	topology.UpdateNumberEthMultisigSigners(context.Background(), num.NewUint(10))
	require.Equal(t, 10, topology.numberEthMultisigSigners)
}

func testUpdateNumberOfTendermintValidators(t *testing.T) {
	topology := &Topology{}
	topology.UpdateNumberOfTendermintValidators(context.Background(), num.NewUint(20))
	topology.UpdateErsatzValidatorsFactor(context.Background(), num.DecimalFromFloat(0.5))
	require.Equal(t, 20, topology.numberOfTendermintValidators)
	require.Equal(t, 10, topology.numberOfErsatzValidators)
}

func testUpdateValidatorIncumbentBonusFactor(t *testing.T) {
	topology := &Topology{}
	topology.UpdateValidatorIncumbentBonusFactor(context.Background(), num.DecimalFromFloat(0.2))
	require.Equal(t, "1.2", topology.validatorIncumbentBonusFactor.String())
}

func testUpdateMinimumEthereumEventsForNewValidator(t *testing.T) {
	topology := &Topology{}
	topology.UpdateMinimumEthereumEventsForNewValidator(context.Background(), num.NewUint(4))
	require.Equal(t, uint64(4), topology.minimumEthereumEventsForNewValidator)
}

func testUpdateMinimumRequireSelfStake(t *testing.T) {
	topology := &Topology{}
	topology.UpdateMinimumRequireSelfStake(context.Background(), num.DecimalFromFloat(30000))
	require.Equal(t, num.NewUint(30000), topology.minimumStake)
}

func testAddForwarder(t *testing.T) {
	topology := NewTopology(logging.NewLoggerFromConfig(logging.Config{}), NewDefaultConfig(), nil, nil, true, nil, &DummyMultiSigTopology{}, &dummyTestTime{})
	// add unknown forwarder
	topology.AddForwarder("node1")
	require.Equal(t, 0, len(topology.validators))

	topology.validators["node2"] = &valState{
		data: ValidatorData{
			ID:         "node2",
			VegaPubKey: "node2Key",
		},
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node3"] = &valState{
		data: ValidatorData{
			ID:         "node3",
			VegaPubKey: "node3Key",
		},
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	require.Equal(t, uint64(0), topology.validators["node3"].numberOfEthereumEventsForwarded)
	require.Equal(t, uint64(0), topology.validators["node2"].numberOfEthereumEventsForwarded)
	topology.AddForwarder("node3Key")
	require.Equal(t, uint64(1), topology.validators["node3"].numberOfEthereumEventsForwarded)
	require.Equal(t, uint64(0), topology.validators["node2"].numberOfEthereumEventsForwarded)

	topology.AddForwarder("node2Key")
	require.Equal(t, uint64(1), topology.validators["node3"].numberOfEthereumEventsForwarded)
	require.Equal(t, uint64(1), topology.validators["node2"].numberOfEthereumEventsForwarded)
	topology.AddForwarder("node3Key")
	require.Equal(t, uint64(2), topology.validators["node3"].numberOfEthereumEventsForwarded)
	require.Equal(t, uint64(1), topology.validators["node2"].numberOfEthereumEventsForwarded)
	topology.AddForwarder("node3Key")
	require.Equal(t, uint64(3), topology.validators["node3"].numberOfEthereumEventsForwarded)
	require.Equal(t, uint64(1), topology.validators["node2"].numberOfEthereumEventsForwarded)
}

func testTendermintValidatorsNumberReduced(t *testing.T) {
	topology := NewTopology(logging.NewLoggerFromConfig(logging.Config{}), NewDefaultConfig(), nil, nil, true, nil, &DummyMultiSigTopology{}, &dummyTestTime{})
	topology.numberOfTendermintValidators = 5
	topology.rng = rand.New(rand.NewSource(100000))
	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID:       "node1",
			TmPubKey: "67g7+123M0kfMR35U7LLq09eEU1dVr6jHBEgEtPzkrs=",
		},
		blockAdded:       1,
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node2"] = &valState{
		data: ValidatorData{
			ID:       "node2",
			TmPubKey: "2w5hxsVqWFTV6/f0swyNVqOhY1vWI42MrfO0xkUqsiA=",
		},
		blockAdded:       1,
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node3"] = &valState{
		data: ValidatorData{
			ID:       "node3",
			TmPubKey: "QZNLWGlqoWv4J9lXqe0pkZQnCJuJbJfiJ50VOj/WsAs=",
		},
		blockAdded:       2,
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node4"] = &valState{
		data: ValidatorData{
			ID:       "node4",
			TmPubKey: "Lor28j7E369gLsU6Q9dW64yKPMn9XiD/IcS1XDXbPSQ=",
		},
		blockAdded:       3,
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node5"] = &valState{
		data: ValidatorData{
			ID:       "node5",
			TmPubKey: "pobW1cLYgsbQGGwbwiwVMqp15WuRzaVp3mn7z+g3ByM=",
		},
		blockAdded:       4,
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}

	perf := map[string]num.Decimal{
		"node1": decimalOne,
		"node2": decimalOne,
		"node3": decimalOne,
		"node4": decimalOne,
		"node5": decimalOne,
	}

	ranking := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.8),
		"node2": num.DecimalFromFloat(0.8),
		"node3": num.DecimalFromFloat(0.4),
		"node4": num.DecimalFromFloat(0.5),
		"node5": num.DecimalFromFloat(0.6),
	}

	delegations := []*types.ValidatorData{
		{NodeID: "node1", SelfStake: num.NewUint(3000), StakeByDelegators: num.NewUint(7000)},
		{NodeID: "node2", SelfStake: num.NewUint(3000), StakeByDelegators: num.NewUint(2000)},
		{NodeID: "node5", SelfStake: num.NewUint(85000), StakeByDelegators: num.NewUint(0)},
	}

	// reduce the number of tendermint validators to 3 to that 2 must be removed, i.e. nodes 3 and 4 with the lowest scores
	topology.currentBlockHeight = 1000
	topology.numberOfTendermintValidators = 3
	topology.numberOfErsatzValidators = 5
	res, _ := topology.applyPromotion(perf, ranking, delegations, types.StakeScoreParams{MinVal: num.DecimalFromFloat(3), CompLevel: num.DecimalFromFloat(1), OptimalStakeMultiplier: num.DecimalFromFloat(1)})

	// node1 has 10000 / 100000 = 0.1 => 0.6666666667 => 6666
	require.Equal(t, int64(6666), res[0].Power) // 10000 * 0.8/2.2
	// node2 has 5000 / 100000 = 0.05 => 0.3333333333 => 3333
	require.Equal(t, int64(3333), res[1].Power) // 10000 * 0.8/2.2
	// node3 is remove => 0
	require.Equal(t, int64(0), res[2].Power) // remove from rm
	// node4 is remove => 0
	require.Equal(t, int64(0), res[3].Power) // remove from rm
	// node5 is anti-whaled => 0 => 0 => 10
	require.Equal(t, int64(1), res[4].Power) // 10000 * 0.6/2.2

	require.Equal(t, "ersatz", ValidatorStatusToName[topology.validators["node3"].status])
	require.Equal(t, int64(1001), topology.validators["node3"].statusChangeBlock)
	require.Equal(t, "ersatz", ValidatorStatusToName[topology.validators["node4"].status])
	require.Equal(t, int64(1001), topology.validators["node4"].statusChangeBlock)
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node1"].status])
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node2"].status])
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node5"].status])
}

func testTendermintFreeSlotsPromotion(t *testing.T) {
	topology := NewTopology(logging.NewLoggerFromConfig(logging.Config{}), NewDefaultConfig(), nil, nil, true, nil, &DummyMultiSigTopology{}, &dummyTestTime{})
	topology.numberOfTendermintValidators = 5
	topology.numberOfErsatzValidators = 1
	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID:       "node1",
			TmPubKey: "67g7+123M0kfMR35U7LLq09eEU1dVr6jHBEgEtPzkrs=",
		},
		blockAdded:        1,
		status:            ValidatorStatusTendermint,
		statusChangeBlock: 900,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node2"] = &valState{
		data: ValidatorData{
			ID:       "node2",
			TmPubKey: "2w5hxsVqWFTV6/f0swyNVqOhY1vWI42MrfO0xkUqsiA=",
		},
		blockAdded:        1,
		status:            ValidatorStatusTendermint,
		statusChangeBlock: 901,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node3"] = &valState{
		data: ValidatorData{
			ID:       "node3",
			TmPubKey: "QZNLWGlqoWv4J9lXqe0pkZQnCJuJbJfiJ50VOj/WsAs=",
		},
		blockAdded:        2,
		status:            ValidatorStatusTendermint,
		statusChangeBlock: 902,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node4"] = &valState{
		data: ValidatorData{
			ID:       "node4",
			TmPubKey: "Lor28j7E369gLsU6Q9dW64yKPMn9XiD/IcS1XDXbPSQ=",
		},
		blockAdded:                      3,
		status:                          ValidatorStatusPending,
		statusChangeBlock:               903,
		numberOfEthereumEventsForwarded: 2,
		heartbeatTracker: &validatorHeartbeatTracker{
			blockSigs: [10]bool{true, true, true, true, false, true, true, true, true, false},
		},
	}
	topology.validators["node5"] = &valState{
		data: ValidatorData{
			ID:       "node5",
			TmPubKey: "pobW1cLYgsbQGGwbwiwVMqp15WuRzaVp3mn7z+g3ByM=",
		},
		blockAdded:                      4,
		statusChangeBlock:               904,
		status:                          ValidatorStatusErsatz,
		numberOfEthereumEventsForwarded: 4,
		heartbeatTracker: &validatorHeartbeatTracker{
			blockSigs: [10]bool{true, true, true, true, false, true, true, true, true, false},
		},
	}

	perfScore := map[string]num.Decimal{
		"node1": decimalOne,
		"node2": decimalOne,
		"node3": decimalOne,
		"node4": decimalOne,
		"node5": decimalOne,
	}

	ranking := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.8),
		"node2": num.DecimalFromFloat(0.8),
		"node3": num.DecimalFromFloat(0.4),
		"node4": num.DecimalFromFloat(0),
		"node5": num.DecimalFromFloat(0.6),
	}

	delegations := []*types.ValidatorData{
		{NodeID: "node1", SelfStake: num.NewUint(12000), StakeByDelegators: num.NewUint(8000)},
		{NodeID: "node2", SelfStake: num.NewUint(10000), StakeByDelegators: num.NewUint(5000)},
		{NodeID: "node3", SelfStake: num.NewUint(20000), StakeByDelegators: num.NewUint(5000)},
		{NodeID: "node4", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node5", SelfStake: num.NewUint(40000), StakeByDelegators: num.NewUint(0)},
	}

	// there are 5 slots for tm validators but only 3 currently
	// there are 2 potential validators for promotion but one of them has not completed their prereq
	topology.currentBlockHeight = 1000
	res, _ := topology.applyPromotion(perfScore, ranking, delegations, types.StakeScoreParams{MinVal: num.DecimalFromFloat(4), CompLevel: num.DecimalFromFloat(1), OptimalStakeMultiplier: num.DecimalFromFloat(1)})
	require.Equal(t, 4, len(res))
	// node1 => 20000 / 100000 => 0.2 => 0.2857142857 => 3333
	require.Equal(t, int64(2857), res[0].Power)
	// node2 => 15000 / 100000 => 0.15 => 0.2142857143 => 2142
	require.Equal(t, int64(2142), res[1].Power)
	// node3 => 25000 / 100000 => 0.25 => 0.3571428571 = 3571
	require.Equal(t, int64(3571), res[2].Power)
	// node5 => 40000 / 100000 => 0.1 (0.4 - 0.15 - 0.15)=> 0.1428571429 => 1428
	require.Equal(t, int64(1428), res[3].Power)

	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node1"].status])
	require.Equal(t, int64(900), topology.validators["node1"].statusChangeBlock)
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node2"].status])
	require.Equal(t, int64(901), topology.validators["node2"].statusChangeBlock)
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node3"].status])
	require.Equal(t, int64(902), topology.validators["node3"].statusChangeBlock)
	require.Equal(t, "pending", ValidatorStatusToName[topology.validators["node4"].status])
	require.Equal(t, int64(903), topology.validators["node4"].statusChangeBlock)
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node5"].status])
	require.Equal(t, int64(1001), topology.validators["node5"].statusChangeBlock)
}

func testSwapBestErsatzWithWorstTendermint(t *testing.T) {
	topology := NewTopology(logging.NewLoggerFromConfig(logging.Config{}), NewDefaultConfig(), nil, nil, true, nil, &DummyMultiSigTopology{}, &dummyTestTime{})
	topology.numberOfTendermintValidators = 4
	topology.numberOfErsatzValidators = 1
	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID:       "node1",
			TmPubKey: "67g7+123M0kfMR35U7LLq09eEU1dVr6jHBEgEtPzkrs=",
		},
		blockAdded:        1,
		status:            ValidatorStatusTendermint,
		statusChangeBlock: 900,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node2"] = &valState{
		data: ValidatorData{
			ID:       "node2",
			TmPubKey: "2w5hxsVqWFTV6/f0swyNVqOhY1vWI42MrfO0xkUqsiA=",
		},
		blockAdded:        1,
		status:            ValidatorStatusTendermint,
		statusChangeBlock: 901,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node3"] = &valState{
		data: ValidatorData{
			ID:       "node3",
			TmPubKey: "QZNLWGlqoWv4J9lXqe0pkZQnCJuJbJfiJ50VOj/WsAs=",
		},
		blockAdded:        2,
		status:            ValidatorStatusTendermint,
		statusChangeBlock: 902,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node4"] = &valState{
		data: ValidatorData{
			ID:       "node4",
			TmPubKey: "Lor28j7E369gLsU6Q9dW64yKPMn9XiD/IcS1XDXbPSQ=",
		},
		blockAdded:        3,
		status:            ValidatorStatusTendermint,
		statusChangeBlock: 903,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node5"] = &valState{
		data: ValidatorData{
			ID:       "node5",
			TmPubKey: "pobW1cLYgsbQGGwbwiwVMqp15WuRzaVp3mn7z+g3ByM=",
		},
		blockAdded:        4,
		statusChangeBlock: 904,
		status:            ValidatorStatusErsatz,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}

	perfScore := map[string]num.Decimal{
		"node1": decimalOne,
		"node2": decimalOne,
		"node3": decimalOne,
		"node4": decimalOne,
		"node5": decimalOne,
	}

	ranking := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.8),
		"node2": num.DecimalFromFloat(0.8),
		"node3": num.DecimalFromFloat(0.4),
		"node4": num.DecimalFromFloat(0.5),
		"node5": num.DecimalFromFloat(0.6),
	}

	delegations := []*types.ValidatorData{
		{NodeID: "node1", SelfStake: num.NewUint(12000), StakeByDelegators: num.NewUint(8000)},
		{NodeID: "node2", SelfStake: num.NewUint(10000), StakeByDelegators: num.NewUint(5000)},
		{NodeID: "node3", SelfStake: num.NewUint(40000), StakeByDelegators: num.NewUint(0)},
		{NodeID: "node4", SelfStake: num.NewUint(20000), StakeByDelegators: num.NewUint(5000)},
		{NodeID: "node5", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
	}

	// there are 4 slots for tm validators and the best ersatz (node5) has better performance than the worst tm (node3)
	// therefore node3 is kicked out of tm and becomes ersatz and node5 is added to tm
	topology.currentBlockHeight = 1000
	res, _ := topology.applyPromotion(perfScore, ranking, delegations, types.StakeScoreParams{MinVal: num.DecimalFromFloat(4), CompLevel: num.DecimalFromFloat(1), OptimalStakeMultiplier: num.DecimalFromFloat(1)})
	require.Equal(t, 5, len(res))
	// node1 => 20000 / 100000 => 0.2 => 0.2857142857 => 3333
	require.Equal(t, int64(2857), res[0].Power)
	// node2 => 15000 / 100000 => 0.15 => 0.2142857143 => 2142
	require.Equal(t, int64(2142), res[1].Power)
	require.Equal(t, int64(0), res[2].Power) // node3 kicked out of tm
	// node4 => 25000 / 100000 => 0.25 => 0.3571428571 = 3571
	require.Equal(t, int64(3571), res[3].Power)
	// node5 => 40000 / 100000 => 0.1 (0.4 - 0.15 - 0.15)=> 0.1428571429 => 1428
	require.Equal(t, int64(1428), res[4].Power)

	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node1"].status])
	require.Equal(t, int64(900), topology.validators["node1"].statusChangeBlock)
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node2"].status])
	require.Equal(t, int64(901), topology.validators["node2"].statusChangeBlock)
	require.Equal(t, "ersatz", ValidatorStatusToName[topology.validators["node3"].status])
	require.Equal(t, int64(1001), topology.validators["node3"].statusChangeBlock)
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node4"].status])
	require.Equal(t, int64(903), topology.validators["node4"].statusChangeBlock)
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node5"].status])
	require.Equal(t, int64(1001), topology.validators["node5"].statusChangeBlock)
}

func testErsatzFreeSlotsPromotion(t *testing.T) {
	topology := NewTopology(logging.NewLoggerFromConfig(logging.Config{}), NewDefaultConfig(), nil, nil, true, nil, &DummyMultiSigTopology{}, &dummyTestTime{})
	topology.numberOfTendermintValidators = 1
	topology.numberOfErsatzValidators = 4

	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID:       "node1",
			TmPubKey: "67g7+123M0kfMR35U7LLq09eEU1dVr6jHBEgEtPzkrs=",
		},
		blockAdded:        1,
		status:            ValidatorStatusTendermint,
		statusChangeBlock: 900,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node2"] = &valState{
		data: ValidatorData{
			ID:       "node2",
			TmPubKey: "2w5hxsVqWFTV6/f0swyNVqOhY1vWI42MrfO0xkUqsiA=",
		},
		blockAdded:        1,
		status:            ValidatorStatusErsatz,
		statusChangeBlock: 901,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node3"] = &valState{
		data: ValidatorData{
			ID:       "node3",
			TmPubKey: "QZNLWGlqoWv4J9lXqe0pkZQnCJuJbJfiJ50VOj/WsAs=",
		},
		blockAdded:        2,
		status:            ValidatorStatusErsatz,
		statusChangeBlock: 902,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node4"] = &valState{
		data: ValidatorData{
			ID:       "node4",
			TmPubKey: "Lor28j7E369gLsU6Q9dW64yKPMn9XiD/IcS1XDXbPSQ=",
		},
		blockAdded:                      3,
		status:                          ValidatorStatusPending,
		statusChangeBlock:               903,
		numberOfEthereumEventsForwarded: 2,
		heartbeatTracker: &validatorHeartbeatTracker{
			blockSigs: [10]bool{true, true, true, true, false, true, true, true, true, false},
		},
	}
	topology.validators["node5"] = &valState{
		data: ValidatorData{
			ID:       "node5",
			TmPubKey: "pobW1cLYgsbQGGwbwiwVMqp15WuRzaVp3mn7z+g3ByM=",
		},
		blockAdded:                      4,
		statusChangeBlock:               904,
		status:                          ValidatorStatusPending,
		numberOfEthereumEventsForwarded: 4,
		heartbeatTracker: &validatorHeartbeatTracker{
			blockSigs: [10]bool{true, true, true, true, false, true, true, true, true, false},
		},
	}

	perfScore := map[string]num.Decimal{
		"node1": decimalOne,
		"node2": decimalOne,
		"node3": decimalOne,
		"node4": decimalOne,
		"node5": decimalOne,
	}

	ranking := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.8),
		"node2": num.DecimalFromFloat(0.8),
		"node3": num.DecimalFromFloat(0.4),
		"node4": num.DecimalFromFloat(0),
		"node5": num.DecimalFromFloat(0.6),
	}

	delegations := []*types.ValidatorData{
		{NodeID: "node1", SelfStake: num.NewUint(12000), StakeByDelegators: num.NewUint(8000)},
		{NodeID: "node2", SelfStake: num.NewUint(10000), StakeByDelegators: num.NewUint(5000)},
		{NodeID: "node3", SelfStake: num.NewUint(40000), StakeByDelegators: num.NewUint(0)},
		{NodeID: "node4", SelfStake: num.NewUint(20000), StakeByDelegators: num.NewUint(5000)},
		{NodeID: "node5", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
	}

	// there's only 1 slot for tendermint and it's taken by the node with the highest rank anyways.
	// there are 4 slots for ersatz validators and only 2 taken so the other two can be promoted
	// there are 2 potential validators for promotion but one of them has not completed their prerequisites
	topology.currentBlockHeight = 1000
	res, _ := topology.applyPromotion(perfScore, ranking, delegations, types.StakeScoreParams{MinVal: num.DecimalFromFloat(2), CompLevel: num.DecimalFromFloat(1), OptimalStakeMultiplier: num.DecimalFromFloat(3)})
	require.Equal(t, int64(10000), res[0].Power) // only node 1 is here

	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node1"].status])
	require.Equal(t, int64(900), topology.validators["node1"].statusChangeBlock)
	require.Equal(t, "ersatz", ValidatorStatusToName[topology.validators["node2"].status])
	require.Equal(t, int64(901), topology.validators["node2"].statusChangeBlock)
	require.Equal(t, "ersatz", ValidatorStatusToName[topology.validators["node3"].status])
	require.Equal(t, int64(902), topology.validators["node3"].statusChangeBlock)
	require.Equal(t, "pending", ValidatorStatusToName[topology.validators["node4"].status])
	require.Equal(t, int64(903), topology.validators["node4"].statusChangeBlock)
	require.Equal(t, "ersatz", ValidatorStatusToName[topology.validators["node5"].status])
	require.Equal(t, int64(1001), topology.validators["node5"].statusChangeBlock)
}

func testSwapBestPendingWithWorstErsatz(t *testing.T) {
	topology := NewTopology(logging.NewLoggerFromConfig(logging.Config{}), NewDefaultConfig(), nil, nil, true, nil, &DummyMultiSigTopology{}, &dummyTestTime{})
	topology.numberOfTendermintValidators = 1
	topology.numberOfErsatzValidators = 2

	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID:       "node1",
			TmPubKey: "67g7+123M0kfMR35U7LLq09eEU1dVr6jHBEgEtPzkrs=",
		},
		blockAdded:        1,
		status:            ValidatorStatusTendermint,
		statusChangeBlock: 900,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node2"] = &valState{
		data: ValidatorData{
			ID:       "node2",
			TmPubKey: "2w5hxsVqWFTV6/f0swyNVqOhY1vWI42MrfO0xkUqsiA=",
		},
		blockAdded:        1,
		status:            ValidatorStatusErsatz,
		statusChangeBlock: 901,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node3"] = &valState{
		data: ValidatorData{
			ID:       "node3",
			TmPubKey: "QZNLWGlqoWv4J9lXqe0pkZQnCJuJbJfiJ50VOj/WsAs=",
		},
		blockAdded:        2,
		status:            ValidatorStatusErsatz,
		statusChangeBlock: 902,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node4"] = &valState{
		data: ValidatorData{
			ID:       "node4",
			TmPubKey: "Lor28j7E369gLsU6Q9dW64yKPMn9XiD/IcS1XDXbPSQ=",
		},
		blockAdded:                      3,
		status:                          ValidatorStatusPending,
		statusChangeBlock:               903,
		numberOfEthereumEventsForwarded: 2,
		heartbeatTracker: &validatorHeartbeatTracker{
			blockSigs: [10]bool{true, true, true, true, false, true, true, true, true, false},
		},
	}
	topology.validators["node5"] = &valState{
		data: ValidatorData{
			ID:       "node5",
			TmPubKey: "pobW1cLYgsbQGGwbwiwVMqp15WuRzaVp3mn7z+g3ByM=",
		},
		blockAdded:                      4,
		statusChangeBlock:               904,
		status:                          ValidatorStatusPending,
		numberOfEthereumEventsForwarded: 4,
		heartbeatTracker: &validatorHeartbeatTracker{
			blockSigs: [10]bool{true, true, true, true, false, true, true, true, true, false},
		},
	}

	perfScore := map[string]num.Decimal{
		"node1": decimalOne,
		"node2": decimalOne,
		"node3": decimalOne,
		"node4": decimalOne,
		"node5": decimalOne,
	}

	ranking := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.8),
		"node2": num.DecimalFromFloat(0.8),
		"node3": num.DecimalFromFloat(0.4),
		"node4": num.DecimalFromFloat(0),
		"node5": num.DecimalFromFloat(0.6),
	}

	delegations := []*types.ValidatorData{
		{NodeID: "node1", SelfStake: num.NewUint(12000), StakeByDelegators: num.NewUint(8000)},
		{NodeID: "node2", SelfStake: num.NewUint(10000), StakeByDelegators: num.NewUint(5000)},
		{NodeID: "node3", SelfStake: num.NewUint(40000), StakeByDelegators: num.NewUint(0)},
		{NodeID: "node4", SelfStake: num.NewUint(20000), StakeByDelegators: num.NewUint(5000)},
		{NodeID: "node5", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
	}

	// there's only 1 slot for tendermint and it's taken by the node with the highest rank anyways.
	// there are 2 slots for ersatz validators both taken
	// the score of node5 is higher than the lowest ersatz so they swap places
	topology.currentBlockHeight = 1000
	res, _ := topology.applyPromotion(perfScore, ranking, delegations, types.StakeScoreParams{MinVal: num.DecimalFromFloat(2), CompLevel: num.DecimalFromFloat(1), OptimalStakeMultiplier: num.DecimalFromFloat(5)})
	require.Equal(t, int64(10000), res[0].Power) // only node 1 is here

	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node1"].status])
	require.Equal(t, int64(900), topology.validators["node1"].statusChangeBlock)
	require.Equal(t, "ersatz", ValidatorStatusToName[topology.validators["node2"].status])
	require.Equal(t, int64(901), topology.validators["node2"].statusChangeBlock)
	require.Equal(t, "pending", ValidatorStatusToName[topology.validators["node3"].status])
	require.Equal(t, int64(1001), topology.validators["node3"].statusChangeBlock)
	require.Equal(t, "pending", ValidatorStatusToName[topology.validators["node4"].status])
	require.Equal(t, int64(903), topology.validators["node4"].statusChangeBlock)
	require.Equal(t, "ersatz", ValidatorStatusToName[topology.validators["node5"].status])
	require.Equal(t, int64(1001), topology.validators["node5"].statusChangeBlock)
}

func testErsatzValidatorsNumberReduced(t *testing.T) {
	topology := NewTopology(logging.NewLoggerFromConfig(logging.Config{}), NewDefaultConfig(), nil, nil, true, nil, &DummyMultiSigTopology{}, &dummyTestTime{})
	topology.numberOfTendermintValidators = 1
	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID:       "node1",
			TmPubKey: "67g7+123M0kfMR35U7LLq09eEU1dVr6jHBEgEtPzkrs=",
		},
		blockAdded:        1,
		statusChangeBlock: 900,
		status:            ValidatorStatusTendermint,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node2"] = &valState{
		data: ValidatorData{
			ID:       "node2",
			TmPubKey: "2w5hxsVqWFTV6/f0swyNVqOhY1vWI42MrfO0xkUqsiA=",
		},
		blockAdded:        1,
		statusChangeBlock: 901,
		status:            ValidatorStatusErsatz,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node3"] = &valState{
		data: ValidatorData{
			ID:       "node3",
			TmPubKey: "QZNLWGlqoWv4J9lXqe0pkZQnCJuJbJfiJ50VOj/WsAs=",
		},
		blockAdded:       2,
		status:           ValidatorStatusErsatz,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node4"] = &valState{
		data: ValidatorData{
			ID:       "node4",
			TmPubKey: "Lor28j7E369gLsU6Q9dW64yKPMn9XiD/IcS1XDXbPSQ=",
		},
		blockAdded:       3,
		status:           ValidatorStatusErsatz,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node5"] = &valState{
		data: ValidatorData{
			ID:       "node5",
			TmPubKey: "pobW1cLYgsbQGGwbwiwVMqp15WuRzaVp3mn7z+g3ByM=",
		},
		blockAdded:       4,
		status:           ValidatorStatusErsatz,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}

	perfScore := map[string]num.Decimal{
		"node1": decimalOne,
		"node2": decimalOne,
		"node3": decimalOne,
		"node4": decimalOne,
		"node5": decimalOne,
	}

	ranking := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.8),
		"node2": num.DecimalFromFloat(0.8),
		"node3": num.DecimalFromFloat(0.4),
		"node4": num.DecimalFromFloat(0.5),
		"node5": num.DecimalFromFloat(0.6),
	}

	delegations := []*types.ValidatorData{
		{NodeID: "node1", SelfStake: num.NewUint(12000), StakeByDelegators: num.NewUint(8000)},
		{NodeID: "node2", SelfStake: num.NewUint(10000), StakeByDelegators: num.NewUint(5000)},
		{NodeID: "node3", SelfStake: num.NewUint(40000), StakeByDelegators: num.NewUint(0)},
		{NodeID: "node4", SelfStake: num.NewUint(20000), StakeByDelegators: num.NewUint(5000)},
		{NodeID: "node5", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
	}

	// reduce the number of ersatz validators from 4 to 1 so that 3 with the lower scores are demoted to pending
	topology.currentBlockHeight = 1000
	topology.numberOfErsatzValidators = 1
	res, _ := topology.applyPromotion(perfScore, ranking, delegations, types.StakeScoreParams{MinVal: num.DecimalFromFloat(2), CompLevel: num.DecimalFromFloat(1), OptimalStakeMultiplier: num.DecimalFromFloat(5)})
	require.Equal(t, int64(10000), res[0].Power) // 10000 * 1

	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node1"].status])
	require.Equal(t, int64(900), topology.validators["node1"].statusChangeBlock)
	require.Equal(t, "ersatz", ValidatorStatusToName[topology.validators["node2"].status])
	require.Equal(t, int64(901), topology.validators["node2"].statusChangeBlock)
	require.Equal(t, "pending", ValidatorStatusToName[topology.validators["node3"].status])
	require.Equal(t, int64(1001), topology.validators["node3"].statusChangeBlock)
	require.Equal(t, "pending", ValidatorStatusToName[topology.validators["node4"].status])
	require.Equal(t, int64(1001), topology.validators["node4"].statusChangeBlock)
	require.Equal(t, "pending", ValidatorStatusToName[topology.validators["node5"].status])
	require.Equal(t, int64(1001), topology.validators["node5"].statusChangeBlock)
}

func testSwapAndErsatzSlotDecrease(t *testing.T) {
	topology := NewTopology(logging.NewLoggerFromConfig(logging.Config{}), NewDefaultConfig(), nil, nil, true, nil, &DummyMultiSigTopology{}, &dummyTestTime{})
	topology.numberOfTendermintValidators = 4
	topology.numberOfErsatzValidators = 2
	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID:       "node1",
			TmPubKey: "67g7+123M0kfMR35U7LLq09eEU1dVr6jHBEgEtPzkrs=",
		},
		blockAdded:        1,
		statusChangeBlock: 900,
		status:            ValidatorStatusTendermint,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node2"] = &valState{
		data: ValidatorData{
			ID:       "node2",
			TmPubKey: "2w5hxsVqWFTV6/f0swyNVqOhY1vWI42MrfO0xkUqsiA=",
		},
		blockAdded:        1,
		statusChangeBlock: 901,
		status:            ValidatorStatusTendermint,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node3"] = &valState{
		data: ValidatorData{
			ID:       "node3",
			TmPubKey: "QZNLWGlqoWv4J9lXqe0pkZQnCJuJbJfiJ50VOj/WsAs=",
		},
		blockAdded:       2,
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node4"] = &valState{
		data: ValidatorData{
			ID:       "node4",
			TmPubKey: "Lor28j7E369gLsU6Q9dW64yKPMn9XiD/IcS1XDXbPSQ=",
		},
		blockAdded:       3,
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node5"] = &valState{
		data: ValidatorData{
			ID:       "node5",
			TmPubKey: "pobW1cLYgsbQGGwbwiwVMqp15WuRzaVp3mn7z+g3ByM=",
		},
		blockAdded:       4,
		status:           ValidatorStatusErsatz,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node6"] = &valState{
		data: ValidatorData{
			ID:       "node5",
			TmPubKey: "pobW1cLYgsbQGGwbwiwVMqp15WuRzaVp3mn7z+g3ByM=",
		},
		blockAdded:       4,
		status:           ValidatorStatusErsatz,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}

	perfScore := map[string]num.Decimal{
		"node1": decimalOne,
		"node2": decimalOne,
		"node3": decimalOne,
		"node4": decimalOne,
		"node5": decimalOne,
		"node6": decimalOne,
	}

	ranking := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.1),
		"node2": num.DecimalFromFloat(0.8),
		"node3": num.DecimalFromFloat(0.8),
		"node4": num.DecimalFromFloat(0.8),
		"node5": num.DecimalFromFloat(0.8),
		"node6": num.DecimalFromFloat(0.5),
	}

	delegations := []*types.ValidatorData{
		{NodeID: "node1", SelfStake: num.NewUint(12000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node2", SelfStake: num.NewUint(10000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node3", SelfStake: num.NewUint(40000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node4", SelfStake: num.NewUint(20000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node5", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node5", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node6", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
	}
	topology.rng = rand.New(rand.NewSource(1000))
	// reduce the number of ersatz validators from 2 to 1
	topology.currentBlockHeight = 1000
	topology.numberOfErsatzValidators = 1
	topology.applyPromotion(perfScore, ranking, delegations, types.StakeScoreParams{MinVal: num.DecimalFromFloat(2), CompLevel: num.DecimalFromFloat(1), OptimalStakeMultiplier: num.DecimalFromFloat(5)})

	ezCount := 0
	for _, v := range topology.validators {
		if ValidatorStatusToName[v.status] == "ersatz" {
			ezCount++
		}
	}

	require.Equal(t, ezCount, topology.numberOfErsatzValidators)
}

func testSwapAndTendermintSlotIncrease(t *testing.T) {
	topology := NewTopology(logging.NewLoggerFromConfig(logging.Config{}), NewDefaultConfig(), nil, nil, true, nil, &DummyMultiSigTopology{}, &dummyTestTime{})
	topology.numberOfTendermintValidators = 4
	topology.numberOfErsatzValidators = 2
	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID:       "node1",
			TmPubKey: "67g7+123M0kfMR35U7LLq09eEU1dVr6jHBEgEtPzkrs=",
		},
		blockAdded:        1,
		statusChangeBlock: 900,
		status:            ValidatorStatusTendermint,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node2"] = &valState{
		data: ValidatorData{
			ID:       "node2",
			TmPubKey: "2w5hxsVqWFTV6/f0swyNVqOhY1vWI42MrfO0xkUqsiA=",
		},
		blockAdded:        1,
		statusChangeBlock: 901,
		status:            ValidatorStatusTendermint,
		heartbeatTracker:  &validatorHeartbeatTracker{},
	}
	topology.validators["node3"] = &valState{
		data: ValidatorData{
			ID:       "node3",
			TmPubKey: "QZNLWGlqoWv4J9lXqe0pkZQnCJuJbJfiJ50VOj/WsAs=",
		},
		blockAdded:       2,
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node4"] = &valState{
		data: ValidatorData{
			ID:       "node4",
			TmPubKey: "Lor28j7E369gLsU6Q9dW64yKPMn9XiD/IcS1XDXbPSQ=",
		},
		blockAdded:       3,
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node5"] = &valState{
		data: ValidatorData{
			ID:       "node5",
			TmPubKey: "pobW1cLYgsbQGGwbwiwVMqp15WuRzaVp3mn7z+g3ByM=",
		},
		blockAdded:       4,
		status:           ValidatorStatusPending,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node6"] = &valState{
		data: ValidatorData{
			ID:       "node6",
			TmPubKey: "pobW1cLYgsbQGGwbwiwVMqp15WuRzaVp3mn7z+g3ByM=",
		},
		blockAdded:       4,
		status:           ValidatorStatusPending,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}

	perfScore := map[string]num.Decimal{
		"node1": decimalOne,
		"node2": decimalOne,
		"node3": decimalOne,
		"node4": decimalOne,
		"node5": decimalOne,
		"node6": decimalOne,
	}

	ranking := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.0), // this TM validator has zero score so will get demoted
		"node2": num.DecimalFromFloat(0.8), // everyone else should be end up being TM due to the slot space, and a promotion
		"node3": num.DecimalFromFloat(0.8),
		"node4": num.DecimalFromFloat(0.8),
		"node5": num.DecimalFromFloat(0.8),
		"node6": num.DecimalFromFloat(0.8),
	}

	delegations := []*types.ValidatorData{
		{NodeID: "node1", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node2", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node3", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node4", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node5", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node5", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
		{NodeID: "node6", SelfStake: num.NewUint(30000), StakeByDelegators: num.NewUint(10000)},
	}
	topology.rng = rand.New(rand.NewSource(1000))
	// increase the number of tendermint validators to 5
	topology.currentBlockHeight = 1000
	topology.numberOfErsatzValidators = 1
	topology.numberOfTendermintValidators = 5
	topology.applyPromotion(perfScore, ranking, delegations, types.StakeScoreParams{MinVal: num.DecimalFromFloat(2), CompLevel: num.DecimalFromFloat(1), OptimalStakeMultiplier: num.DecimalFromFloat(5)})

	// node5 should get promoted due to and increase in slots while node6 gets promoted and swapped with node1
	require.Equal(t, "ersatz", ValidatorStatusToName[topology.validators["node1"].status])
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node2"].status])
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node3"].status])
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node4"].status])
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node5"].status])
	require.Equal(t, "tendermint", ValidatorStatusToName[topology.validators["node6"].status])
}

type DummyMultiSigTopology struct{}

func (*DummyMultiSigTopology) IsSigner(address string) bool {
	return true
}

func (*DummyMultiSigTopology) ExcessSigners(addresses []string) bool {
	return false
}

func (*DummyMultiSigTopology) GetThreshold() uint32 {
	return 666
}

func (*DummyMultiSigTopology) GetSigners() []string {
	return []string{}
}
