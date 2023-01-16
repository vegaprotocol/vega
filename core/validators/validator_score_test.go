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

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	v1 "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type TestMultisigTopology struct {
	validators map[string]struct{}
}

func (t *TestMultisigTopology) IsSigner(address string) bool {
	_, ok := t.validators[address]
	return ok
}

func (t *TestMultisigTopology) ExcessSigners(addresses []string) bool {
	if len(t.validators) > len(addresses) {
		return true
	}

	m := make(map[string]struct{}, len(addresses))
	for _, v := range addresses {
		m[v] = struct{}{}
	}

	for k := range t.validators {
		if _, ok := m[k]; !ok {
			return true
		}
	}
	return false
}

func (t *TestMultisigTopology) GetSigners() []string {
	signers := make([]string, 0, len(t.validators))
	for k := range t.validators {
		signers = append(signers, k)
	}
	return signers
}

func (t *TestMultisigTopology) GetThreshold() uint32 {
	return 666
}

func TestScores(t *testing.T) {
	t.Run("test calculation of stake score with no anti-whaling", testStakeScore)
	t.Run("test calculation of performance score", testPerformanceScore)
	t.Run("test calculation of ranking score from stake and performance scores", testRankingScoreInternal)
	t.Run("test calculation of ranking score from delegation data and validators state", testRankingScore)
	t.Run("test score normalisation", testNormalisedScores)
	t.Run("test validator score with anti whaling", testValidatorScore)
	t.Run("test composition of raw validator score for rewards with performance score", testGetValScore)
	t.Run("test multisig score", testGetMultisigScore)
	t.Run("test multisig score more validators than signers", TestGetMultisigScoreMoreValidatorsThanSigners)
	t.Run("test calculate tm rewards scores", testCalculateTMScores)
	t.Run("test calculate ersatz rewards scores", testCalculateErsatzScores)
	t.Run("test calculate rewards scores", testGetRewardsScores)
}

func testStakeScore(t *testing.T) {
	validatorData := []*types.ValidatorData{
		{NodeID: "node1", PubKey: "node1PubKey", StakeByDelegators: num.NewUint(8000), SelfStake: num.NewUint(2000), Delegators: map[string]*num.Uint{}, TmPubKey: "key1"},
		{NodeID: "node2", PubKey: "node2PubKey", StakeByDelegators: num.NewUint(2000), SelfStake: num.NewUint(3000), Delegators: map[string]*num.Uint{}, TmPubKey: "key2"},
		{NodeID: "node3", PubKey: "node3PubKey", StakeByDelegators: num.NewUint(3000), SelfStake: num.NewUint(7000), Delegators: map[string]*num.Uint{}, TmPubKey: "key3"},
		{NodeID: "node4", PubKey: "node4PubKey", StakeByDelegators: num.NewUint(4000), SelfStake: num.NewUint(11000), Delegators: map[string]*num.Uint{}, TmPubKey: "key4"},
		{NodeID: "node5", PubKey: "node5PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(15000), Delegators: map[string]*num.Uint{}, TmPubKey: "key5"},
	}

	// node1 has 10000 / 60000 = 0.1666666667
	// node2 has 5000 / 60000 = 0.08333333333
	// node3 has 10000 / 60000 = 0.1666666667
	// node4 has 15000 / 60000 = 0.25
	// node5 has 20000 / 60000 = 0.3333333333
	scores := getStakeScore(validatorData)
	require.Equal(t, "0.1666666666666667", scores["node1"].String())
	require.Equal(t, "0.0833333333333333", scores["node2"].String())
	require.Equal(t, "0.1666666666666667", scores["node3"].String())
	require.Equal(t, "0.25", scores["node4"].String())
	require.Equal(t, "0.3333333333333333", scores["node5"].String())
}

func testPerformanceScore(t *testing.T) {
	topology := &Topology{}
	topology.log = logging.NewTestLogger()
	topology.validators = map[string]*valState{}
	delegation := []*types.ValidatorData{
		{NodeID: "node1", PubKey: "node1PubKey", StakeByDelegators: num.NewUint(8000), SelfStake: num.NewUint(2000), Delegators: map[string]*num.Uint{}, TmPubKey: "key1"},
		{NodeID: "node2", PubKey: "node2PubKey", StakeByDelegators: num.NewUint(2000), SelfStake: num.NewUint(3000), Delegators: map[string]*num.Uint{}, TmPubKey: "key2"},
		{NodeID: "node3", PubKey: "node3PubKey", StakeByDelegators: num.NewUint(3000), SelfStake: num.NewUint(7000), Delegators: map[string]*num.Uint{}, TmPubKey: "key3"},
		{NodeID: "node4", PubKey: "node4PubKey", StakeByDelegators: num.NewUint(4000), SelfStake: num.NewUint(11000), Delegators: map[string]*num.Uint{}, TmPubKey: "key4"},
		{NodeID: "node5", PubKey: "node5PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(15000), Delegators: map[string]*num.Uint{}, TmPubKey: "key5"},
	}

	// node1 has less than the minimum self stake => 0
	// node2 is a tendermint validator and gets a performance score of 0.8
	// node3 is pending and hasn't forwarded yet 3 events
	// node4 is pending and hasn't voted yet for 3 events
	// node5 is in the waiting list and has signed 9 of the 10 expected messages

	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID: "node1",
		},
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node2"] = &valState{
		data: ValidatorData{
			ID: "node2",
		},
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node3"] = &valState{
		data: ValidatorData{
			ID: "node3",
		},
		status:                          ValidatorStatusPending,
		numberOfEthereumEventsForwarded: 2,
		heartbeatTracker:                &validatorHeartbeatTracker{},
	}
	topology.validators["node4"] = &valState{
		data: ValidatorData{
			ID: "node4",
		},
		status:                          ValidatorStatusPending,
		numberOfEthereumEventsForwarded: 4,
		heartbeatTracker:                &validatorHeartbeatTracker{},
	}
	topology.validators["node5"] = &valState{
		data: ValidatorData{
			ID: "node5",
		},
		status:                          ValidatorStatusPending,
		numberOfEthereumEventsForwarded: 4,
		heartbeatTracker: &validatorHeartbeatTracker{
			blockSigs: [10]bool{false, true, true, true, true, true, true, true, true, true},
		},
	}

	topology.minimumStake = num.NewUint(3000)
	topology.validatorPerformance = &MockPerformanceScore{}

	scores := topology.getPerformanceScore(delegation)

	require.Equal(t, "0", scores["node1"].String())
	require.Equal(t, "0.8", scores["node2"].String())
	require.Equal(t, "0", scores["node3"].String())
	require.Equal(t, "0", scores["node4"].String())
	require.Equal(t, "0.9", scores["node5"].String())
}

func testRankingScoreInternal(t *testing.T) {
	stakeScores := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.1),
		"node2": num.DecimalFromFloat(0.15),
		"node3": num.DecimalFromFloat(0.2),
		"node4": num.DecimalFromFloat(0.25),
		"node5": num.DecimalFromFloat(0.3),
	}
	perfScores := map[string]num.Decimal{
		"node1": num.DecimalZero(),
		"node2": num.DecimalFromFloat(0.5),
		"node3": num.DecimalFromFloat(0.9),
		"node4": num.DecimalFromFloat(0.2),
		"node5": num.DecimalFromFloat(1),
	}

	topology := &Topology{}
	topology.log = logging.NewTestLogger()
	topology.validators = map[string]*valState{}
	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID: "node1",
		},
		status:           ValidatorStatusPending,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node2"] = &valState{
		data: ValidatorData{
			ID: "node2",
		},
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node3"] = &valState{
		data: ValidatorData{
			ID: "node3",
		},
		status:           ValidatorStatusPending,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node4"] = &valState{
		data: ValidatorData{
			ID: "node4",
		},
		status:           ValidatorStatusErsatz,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node5"] = &valState{
		data: ValidatorData{
			ID: "node5",
		},
		status:           ValidatorStatusErsatz,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}

	topology.validatorIncumbentBonusFactor = num.DecimalFromFloat(1.1)
	rankingScores := topology.getRankingScoreInternal(stakeScores, perfScores)

	// 0.1 * 0 = 0
	require.Equal(t, "0", rankingScores["node1"].String())
	// 0.15 * 0.5 * 1.1 = 0.0825
	require.Equal(t, "0.0825", rankingScores["node2"].String())
	// 0.2 * 0.9 = 0.18
	require.Equal(t, "0.18", rankingScores["node3"].String())
	// 0.25 * 0.2 * 1.1 = 0.055
	require.Equal(t, "0.055", rankingScores["node4"].String())
	// 0.3 * 1 * 1.1 = 0.33
	require.Equal(t, "0.33", rankingScores["node5"].String())
}

func testRankingScore(t *testing.T) {
	topology := &Topology{}
	topology.log = logging.NewTestLogger()
	topology.validators = map[string]*valState{}
	delegation := []*types.ValidatorData{
		{NodeID: "node1", PubKey: "node1PubKey", StakeByDelegators: num.NewUint(8000), SelfStake: num.NewUint(2000), Delegators: map[string]*num.Uint{}, TmPubKey: "key1"},
		{NodeID: "node2", PubKey: "node2PubKey", StakeByDelegators: num.NewUint(2000), SelfStake: num.NewUint(3000), Delegators: map[string]*num.Uint{}, TmPubKey: "key2"},
		{NodeID: "node3", PubKey: "node3PubKey", StakeByDelegators: num.NewUint(3000), SelfStake: num.NewUint(7000), Delegators: map[string]*num.Uint{}, TmPubKey: "key3"},
		{NodeID: "node4", PubKey: "node4PubKey", StakeByDelegators: num.NewUint(4000), SelfStake: num.NewUint(11000), Delegators: map[string]*num.Uint{}, TmPubKey: "key4"},
		{NodeID: "node5", PubKey: "node5PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(15000), Delegators: map[string]*num.Uint{}, TmPubKey: "key5"},
	}

	// node1 has less than the minimum self stake => 0
	// node2 is a tendermint validator and gets a performance score of 0.8
	// node3 is pending and hasn't forwarded yet 3 events
	// node4 is pending and hasn't voted yet for 3 events
	// node5 is in the waiting list and has signed 9 of the 10 expected messages

	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID: "node1",
		},
		status:           ValidatorStatusPending,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node2"] = &valState{
		data: ValidatorData{
			ID: "node2",
		},
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node3"] = &valState{
		data: ValidatorData{
			ID: "node3",
		},
		status:                          ValidatorStatusErsatz,
		numberOfEthereumEventsForwarded: 4,
		heartbeatTracker: &validatorHeartbeatTracker{
			blockSigs: [10]bool{false, false, true, false, true, false, true, true, true, true},
		},
	}
	topology.validators["node4"] = &valState{
		data: ValidatorData{
			ID: "node4",
		},
		status:                          ValidatorStatusPending,
		numberOfEthereumEventsForwarded: 4,
		heartbeatTracker:                &validatorHeartbeatTracker{},
	}
	topology.validators["node5"] = &valState{
		data: ValidatorData{
			ID: "node5",
		},
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}

	topology.minimumStake = num.NewUint(3000)
	topology.validatorPerformance = &MockPerformanceScore{}
	topology.validatorIncumbentBonusFactor = num.DecimalFromFloat(1.1)

	stakeScores, perfScores, rankingScores := topology.getRankingScore(delegation)
	require.Equal(t, "0.1666666666666667", stakeScores["node1"].String())
	require.Equal(t, "0.0833333333333333", stakeScores["node2"].String())
	require.Equal(t, "0.1666666666666667", stakeScores["node3"].String())
	require.Equal(t, "0.25", stakeScores["node4"].String())
	require.Equal(t, "0.3333333333333333", stakeScores["node5"].String())

	// less than min self stake
	require.Equal(t, "0", perfScores["node1"].String())
	// tm validator performance = 0.8
	require.Equal(t, "0.8", perfScores["node2"].String())
	// ersatz signed 6/10 => performance = 0.6
	require.Equal(t, "0.6", perfScores["node3"].String())
	// pending - min requirements not met yet
	require.Equal(t, "0", perfScores["node4"].String())
	// tm validator performance = 0.8
	require.Equal(t, "0.8", perfScores["node5"].String())

	// ranking scores:
	// 0.1666666666666667 * 0 = 0
	require.Equal(t, "0", rankingScores["node1"].String())
	// 0.0833333333333333*0.8*1.1 = 0.07333333333
	require.Equal(t, "0.073333333333333304", rankingScores["node2"].String())
	// 0.1666666666666667 * 0.6 * 1.1 = 0.11
	require.Equal(t, "0.110000000000000022", rankingScores["node3"].String())
	// 0.25 * 0 = 0
	require.Equal(t, "0", rankingScores["node4"].String())
	// 0.3333333333333333 * 1.1 * 0.8 = 0.2933333333
	require.Equal(t, "0.293333333333333304", rankingScores["node5"].String())
}

func testNormalisedScores(t *testing.T) {
	rnd := rand.New(rand.NewSource(100000))
	scores := map[string]num.Decimal{
		"node1": num.DecimalZero(),
		"node2": num.DecimalFromFloat(0.073333333333333304),
		"node3": num.DecimalFromFloat(0.110000000000000022),
		"node4": num.DecimalZero(),
		"node5": num.DecimalFromFloat(0.293333333333333304),
	}
	norm := normaliseScores(scores, rnd)
	require.Equal(t, "0", norm["node1"].String())
	require.Equal(t, "0.1538461538461538", norm["node2"].String())
	require.Equal(t, "0.2307692307692308", norm["node3"].String())
	require.Equal(t, "0", norm["node4"].String())
	require.Equal(t, "0.6153846153846154", norm["node5"].String())

	total := num.DecimalZero()
	for _, d := range norm {
		total = total.Add(d)
	}
	require.True(t, total.LessThanOrEqual(num.DecimalFromFloat(1)))
}

func testValidatorScore(t *testing.T) {
	validatorStake := num.DecimalFromInt64(10000)
	largeValidatorStake := num.DecimalFromInt64(40000)
	extraLargeValidatorStake := num.DecimalFromInt64(60000)
	extraExtraLargeValidatorStake := num.DecimalFromInt64(70000)
	totalStake := num.DecimalFromInt64(100000.0)
	minVal := num.DecimalFromInt64(5)
	compLevel, _ := num.DecimalFromString("1.1")
	optimalStakeMultiplier, _ := num.DecimalFromString("3.0")

	stakeScoreParams := types.StakeScoreParams{
		MinVal:                 minVal,
		CompLevel:              compLevel,
		OptimalStakeMultiplier: optimalStakeMultiplier,
	}

	// valStake = 10k, totalStake = 100k, optStake = 20k
	// valScore = 0.1
	require.Equal(t, "0.10", CalcValidatorScore(validatorStake, totalStake, num.DecimalFromInt64(20000), stakeScoreParams).StringFixed(2))

	// valStake = 20k, totalStake = 100k, optStake = 20k
	// valScore = 0.2
	// no pentalty
	require.Equal(t, "0.20", CalcValidatorScore(largeValidatorStake, totalStake, num.DecimalFromInt64(20000), stakeScoreParams).StringFixed(2))

	// valStake = 60k, totalStake = 100k, optStake = 20k
	// valScore = 0.2
	// with flat pentalty
	require.Equal(t, "0.20", CalcValidatorScore(extraLargeValidatorStake, totalStake, num.DecimalFromInt64(20000), stakeScoreParams).StringFixed(2))

	// valStake = 70k, totalStake = 100k, optStake = 20k
	// valScore = 0.1
	// with flat and down pentalty
	require.Equal(t, "0.10", CalcValidatorScore(extraExtraLargeValidatorStake, totalStake, num.DecimalFromInt64(20000), stakeScoreParams).StringFixed(2))

	// no stake => 0
	require.Equal(t, "0.00", CalcValidatorScore(num.DecimalZero(), num.DecimalZero(), num.DecimalFromInt64(20000), stakeScoreParams).StringFixed(2))
}

func testGetValScore(t *testing.T) {
	stakeScore := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.1),
		"node2": num.DecimalFromFloat(0.2),
		"node3": num.DecimalFromFloat(0.3),
		"node4": num.DecimalFromFloat(0.4),
		"node5": num.DecimalFromFloat(0.5),
	}
	perfScore := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.5),
		"node2": num.DecimalFromFloat(0.6),
		"node3": num.DecimalFromFloat(0.7),
		"node4": num.DecimalFromFloat(0.8),
		"node5": num.DecimalFromFloat(0.9),
	}

	valScore := getValScore(stakeScore, perfScore)
	require.Equal(t, "0.05", valScore["node1"].String())
	require.Equal(t, "0.12", valScore["node2"].String())
	require.Equal(t, "0.21", valScore["node3"].String())
	require.Equal(t, "0.32", valScore["node4"].String())
	require.Equal(t, "0.45", valScore["node5"].String())
}

func testGetMultisigScore(t *testing.T) {
	stakeScore := map[string]num.Decimal{
		"node1":  num.DecimalFromFloat(0.1),
		"node2":  num.DecimalFromFloat(0.2),
		"node3":  num.DecimalFromFloat(0.3),
		"node4":  num.DecimalFromFloat(0.4),
		"node5":  num.DecimalFromFloat(0.5),
		"node6":  num.DecimalFromFloat(0.55),
		"node7":  num.DecimalFromFloat(0.6),
		"node8":  num.DecimalFromFloat(0.65),
		"node9":  num.DecimalFromFloat(0.7),
		"node10": num.DecimalFromFloat(0.75),
	}
	perfScore := map[string]num.Decimal{
		"node1":  num.DecimalFromFloat(0.5),
		"node2":  num.DecimalFromFloat(0.6),
		"node3":  num.DecimalFromFloat(0.7),
		"node4":  num.DecimalFromFloat(0.8),
		"node5":  num.DecimalFromFloat(0.9),
		"node6":  num.DecimalFromFloat(0.9),
		"node7":  num.DecimalFromFloat(0.9),
		"node8":  num.DecimalFromFloat(0.9),
		"node9":  num.DecimalFromFloat(0.9),
		"node10": num.DecimalFromFloat(0.9),
	}

	multisigValidators := map[string]struct{}{
		"node1eth":  {},
		"node2eth":  {},
		"node5eth":  {},
		"node7eth":  {},
		"node8eth":  {},
		"node9eth":  {},
		"node10eth": {},
	}

	nodeIDToEthAddress := map[string]string{
		"node1":  "node1eth",
		"node2":  "node2eth",
		"node3":  "node3eth",
		"node4":  "node4eth",
		"node5":  "node5eth",
		"node6":  "node6eth",
		"node7":  "node7eth",
		"node8":  "node8eth",
		"node9":  "node9eth",
		"node10": "node10eth",
	}

	multisigTopology := &TestMultisigTopology{
		validators: multisigValidators,
	}

	log := logging.NewTestLogger()
	multisigScore := getMultisigScore(
		log,
		ValidatorStatusTendermint,
		stakeScore, perfScore, multisigTopology, 5,
		nodeIDToEthAddress,
	)

	// sorted by the score = stake x performance node 10 is the top and node 1 is the bottom.
	// looking at the top 5 that gives node10 - node6
	// out of those node 10,9,8,7 are in the multisig set
	// node 6 is not so it gets a multisig score of 0
	// all the other nodes are not required to be so their multisig score is 1.
	require.Equal(t, "1", multisigScore["node1"].String())
	require.Equal(t, "1", multisigScore["node2"].String())
	require.Equal(t, "1", multisigScore["node3"].String())
	require.Equal(t, "1", multisigScore["node4"].String())
	require.Equal(t, "1", multisigScore["node5"].String())
	require.Equal(t, "0", multisigScore["node6"].String())
	require.Equal(t, "1", multisigScore["node7"].String())
	require.Equal(t, "1", multisigScore["node8"].String())
	require.Equal(t, "1", multisigScore["node9"].String())
	require.Equal(t, "1", multisigScore["node10"].String())

	multisigValidators["node100"] = struct{}{}
	nodeIDToEthAddress["node100"] = "node100eth"

	multisigScore = getMultisigScore(log, ValidatorStatusTendermint, stakeScore, perfScore, multisigTopology, 5, nodeIDToEthAddress)
	require.Equal(t, "0", multisigScore["node1"].String())
	require.Equal(t, "0", multisigScore["node2"].String())
	require.Equal(t, "0", multisigScore["node3"].String())
	require.Equal(t, "0", multisigScore["node4"].String())
	require.Equal(t, "0", multisigScore["node5"].String())
	require.Equal(t, "0", multisigScore["node6"].String())
	require.Equal(t, "0", multisigScore["node7"].String())
	require.Equal(t, "0", multisigScore["node8"].String())
	require.Equal(t, "0", multisigScore["node9"].String())
	require.Equal(t, "0", multisigScore["node10"].String())
}

func TestGetMultisigScoreMoreValidatorsThanSigners(t *testing.T) {
	// 5 nodes all with an equal score, and none of them on the contract. We also set the number of signers we check to only 2
	// normally in this case we check the 2 nodes with the highest score, but when they are all equal we *should* instead sort
	// by nodeID
	nEthMultisigSigners := 2
	stakeScore := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.1),
		"node2": num.DecimalFromFloat(0.1),
		"node3": num.DecimalFromFloat(0.1),
		"node4": num.DecimalFromFloat(0.1),
		"node5": num.DecimalFromFloat(0.1),
	}
	perfScore := map[string]num.Decimal{
		"node1": num.DecimalFromFloat(0.1),
		"node2": num.DecimalFromFloat(0.1),
		"node3": num.DecimalFromFloat(0.1),
		"node4": num.DecimalFromFloat(0.1),
		"node5": num.DecimalFromFloat(0.1),
	}

	nodeIDToEthAddress := map[string]string{
		"node1": "node1eth",
		"node2": "node2eth",
		"node3": "node3eth",
		"node4": "node4eth",
		"node5": "node5eth",
	}

	multisigTopology := &TestMultisigTopology{
		validators: map[string]struct{}{},
	}

	log := logging.NewTestLogger()

	for i := 0; i < 100; i++ {
		multisigScore := getMultisigScore(
			log,
			ValidatorStatusTendermint,
			stakeScore, perfScore, multisigTopology, nEthMultisigSigners,
			nodeIDToEthAddress,
		)
		require.Equal(t, "0", multisigScore["node1"].String())
		require.Equal(t, "0", multisigScore["node2"].String())
		require.Equal(t, "1", multisigScore["node3"].String())
		require.Equal(t, "1", multisigScore["node4"].String())
		require.Equal(t, "1", multisigScore["node5"].String())
	}
}

func testCalculateTMScores(t *testing.T) {
	topology := &Topology{}
	topology.validators = map[string]*valState{}
	topology.log = logging.NewTestLogger()

	for i := 0; i < 10; i++ {
		index := num.NewUint(uint64(i) + 1).String()
		topology.validators["node"+index] = &valState{
			data: ValidatorData{
				ID:              "node" + index,
				TmPubKey:        "key" + index,
				EthereumAddress: "node" + index + "eth",
			},
			status:           ValidatorStatusTendermint,
			heartbeatTracker: &validatorHeartbeatTracker{},
		}
	}
	topology.validators["node11"] = &valState{
		data: ValidatorData{
			ID:              "node11",
			EthereumAddress: "node11eth",
		},
		status:           ValidatorStatusErsatz,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node12"] = &valState{
		data: ValidatorData{
			ID:              "node12",
			EthereumAddress: "node12eth",
		},
		status:           ValidatorStatusErsatz,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}

	topology.validatorIncumbentBonusFactor = num.DecimalFromFloat(1.1)
	topology.minimumStake = num.NewUint(3000)
	topology.validatorPerformance = &MockPerformanceScore{}
	topology.numberEthMultisigSigners = 7

	multisigTopology := &TestMultisigTopology{}
	multisigTopology.validators = map[string]struct{}{
		"node1eth": {},
		"node2eth": {},
		"node3eth": {},
		"node5eth": {},
		"node7eth": {},
		"node9eth": {},
	}
	topology.multiSigTopology = multisigTopology

	delegation := []*types.ValidatorData{
		{NodeID: "node1", PubKey: "node1PubKey", StakeByDelegators: num.NewUint(8000), SelfStake: num.NewUint(2000), Delegators: map[string]*num.Uint{}, TmPubKey: "key1"},
		{NodeID: "node2", PubKey: "node2PubKey", StakeByDelegators: num.NewUint(2000), SelfStake: num.NewUint(3000), Delegators: map[string]*num.Uint{}, TmPubKey: "key2"},
		{NodeID: "node3", PubKey: "node3PubKey", StakeByDelegators: num.NewUint(3000), SelfStake: num.NewUint(7000), Delegators: map[string]*num.Uint{}, TmPubKey: "key3"},
		{NodeID: "node4", PubKey: "node4PubKey", StakeByDelegators: num.NewUint(4000), SelfStake: num.NewUint(11000), Delegators: map[string]*num.Uint{}, TmPubKey: "key4"},
		{NodeID: "node5", PubKey: "node5PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(15000), Delegators: map[string]*num.Uint{}, TmPubKey: "key5"},
		{NodeID: "node6", PubKey: "node6PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(7000), Delegators: map[string]*num.Uint{}, TmPubKey: "key6"},
		{NodeID: "node7", PubKey: "node7PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(6000), Delegators: map[string]*num.Uint{}, TmPubKey: "key7"},
		{NodeID: "node8", PubKey: "node8PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(5000), Delegators: map[string]*num.Uint{}, TmPubKey: "key8"},
		{NodeID: "node9", PubKey: "node9PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(4000), Delegators: map[string]*num.Uint{}, TmPubKey: "key9"},
		{NodeID: "node10", PubKey: "node10PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(3000), Delegators: map[string]*num.Uint{}, TmPubKey: "key10"},
		{NodeID: "node11", PubKey: "node11PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(2000), Delegators: map[string]*num.Uint{}, TmPubKey: "key11"},
		{NodeID: "node12", PubKey: "node12PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(1000), Delegators: map[string]*num.Uint{}, TmPubKey: "key12"},
	}
	scoreData, _ := topology.calculateScores(delegation, ValidatorStatusTendermint, types.StakeScoreParams{MinVal: num.DecimalFromFloat(5), CompLevel: num.DecimalFromFloat(3.3), OptimalStakeMultiplier: num.DecimalFromFloat(3)}, nil)

	require.Equal(t, 10, len(scoreData.RawValScores))
	require.Equal(t, 10, len(scoreData.PerformanceScores))
	require.Equal(t, 10, len(scoreData.MultisigScores))
	require.Equal(t, 10, len(scoreData.ValScores))
	require.Equal(t, 10, len(scoreData.NormalisedScores))

	// raw scores
	// total = 110000
	// node1 = 10000/110000 = 0.09090909091
	// node2 = 5000/110000 = 0.04545454545
	// node3 = 10000/110000 = 0.09090909091
	// node4 = 15000/110000 = 0.1363636364
	// node5 = 20000/110000 = 0.1818181818
	// node6 = 12000/110000 = 0.1090909091
	// node7 = 11000/110000 = 0.1
	// node8 = 10000/110000 = 0.09090909091
	// node9 = 9000/110000 = 0.08181818182
	// node10 = 8000/110000 = 0.07272727273
	require.Equal(t, "0.09090909", scoreData.RawValScores["node1"].StringFixed(8))
	require.Equal(t, "0.04545455", scoreData.RawValScores["node2"].StringFixed(8))
	require.Equal(t, "0.09090909", scoreData.RawValScores["node3"].StringFixed(8))
	require.Equal(t, "0.13636364", scoreData.RawValScores["node4"].StringFixed(8))
	require.Equal(t, "0.18181818", scoreData.RawValScores["node5"].StringFixed(8))
	require.Equal(t, "0.10909091", scoreData.RawValScores["node6"].StringFixed(8))
	require.Equal(t, "0.10000000", scoreData.RawValScores["node7"].StringFixed(8))
	require.Equal(t, "0.09090909", scoreData.RawValScores["node8"].StringFixed(8))
	require.Equal(t, "0.08181818", scoreData.RawValScores["node9"].StringFixed(8))
	require.Equal(t, "0.07272727", scoreData.RawValScores["node10"].StringFixed(8))

	// performance score
	// node1 has less than the minimum self stake => 0
	// node2-5 0.8
	// node6 0.3
	// node7 0.7
	// node8 0.3
	// node9 0.7
	// node10 1
	require.Equal(t, "0", scoreData.PerformanceScores["node1"].String())
	require.Equal(t, "0.8", scoreData.PerformanceScores["node2"].String())
	require.Equal(t, "0.8", scoreData.PerformanceScores["node3"].String())
	require.Equal(t, "0.8", scoreData.PerformanceScores["node4"].String())
	require.Equal(t, "0.8", scoreData.PerformanceScores["node5"].String())
	require.Equal(t, "0.3", scoreData.PerformanceScores["node6"].String())
	require.Equal(t, "0.7", scoreData.PerformanceScores["node7"].String())
	require.Equal(t, "0.3", scoreData.PerformanceScores["node8"].String())
	require.Equal(t, "0.7", scoreData.PerformanceScores["node9"].String())
	require.Equal(t, "1", scoreData.PerformanceScores["node10"].String())

	// multisig score
	// stake_score x performance_score:
	// node1 = 0
	// node2 = 0.04545454545 * 0.8 = 0.03636363636
	// node3= 0.09090909091 * 0.8 = 0.07272727273
	// node4 = 0.1363636364 * 0.8 = 0.1090909091
	// node5 = 0.1818181818 * 0.8 = 0.1454545454
	// node6 = 0.1090909091 * 0.3 = 0.03272727273
	// node7 = 0.1 * 0.7 = 0.07
	// node8 = 0.09090909091 * 0.3 = 0.02727272727
	// node9 = 0.08181818182 * 0.7 = 0.05727272727
	// node10 = 0.07272727273 * 1 = 0.07272727273
	// sorted order is:
	// node5, node4, node3, node10, node7, node9, node2, node6, node8, node1
	// the net param is set to 7 so we're looking at the top 7 scores
	require.Equal(t, "1", scoreData.MultisigScores["node1"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node2"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node3"].String())
	require.Equal(t, "0", scoreData.MultisigScores["node4"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node5"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node6"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node7"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node8"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node9"].String())
	require.Equal(t, "0", scoreData.MultisigScores["node10"].String())

	// val scores = stake_score * perf_score * multisig_score
	require.Equal(t, "0", scoreData.ValScores["node1"].String())
	require.Equal(t, "0.03636363636", scoreData.ValScores["node2"].StringFixed(11))
	require.Equal(t, "0.07272727273", scoreData.ValScores["node3"].StringFixed(11))
	require.Equal(t, "0", scoreData.ValScores["node4"].String())
	require.Equal(t, "0.14545454545", scoreData.ValScores["node5"].StringFixed(11))
	require.Equal(t, "0.03272727272727273", scoreData.ValScores["node6"].String())
	require.Equal(t, "0.07", scoreData.ValScores["node7"].StringFixed(2))
	require.Equal(t, "0.02727272727272727", scoreData.ValScores["node8"].String())
	require.Equal(t, "0.05727272727", scoreData.ValScores["node9"].StringFixed(11))
	require.Equal(t, "0", scoreData.ValScores["node10"].String())

	// normalised scores
	// node2 = 0.03636363636 / 0.3818181818 = 0.09523809523
	// node3 = 0.07272727273 / 0.3818181818 = 0.1904761905
	// node5 = 0.14545454545 / 0.3818181818 = 0.380952381
	// node7 = 0.07 / 0.3818181818 = 0.1833333333
	// node9 = 0.05727272727 / 0.3818181818 = 0.15
	require.Equal(t, "0.08230452675", scoreData.NormalisedScores["node2"].StringFixed(11))
	require.Equal(t, "0.16460905350", scoreData.NormalisedScores["node3"].StringFixed(11))
	require.Equal(t, "0.32921810700", scoreData.NormalisedScores["node5"].StringFixed(11))
	require.Equal(t, "0.15843621399", scoreData.NormalisedScores["node7"].StringFixed(11))
	require.Equal(t, "0.13", scoreData.NormalisedScores["node9"].StringFixed(2))

	totalNormScore := num.DecimalZero()
	for _, d := range scoreData.NormalisedScores {
		totalNormScore = totalNormScore.Add(d)
	}
	require.True(t, totalNormScore.LessThanOrEqual(decimalOne))
}

func testCalculateErsatzScores(t *testing.T) {
	topology := &Topology{}
	topology.log = logging.NewTestLogger()
	topology.validators = map[string]*valState{}

	for i := 0; i < 10; i++ {
		index := num.NewUint(uint64(i) + 1).String()
		topology.validators["node"+index] = &valState{
			data: ValidatorData{
				ID:              "node" + index,
				EthereumAddress: "node" + index + "eth",
			},
			status:                          ValidatorStatusErsatz,
			heartbeatTracker:                &validatorHeartbeatTracker{},
			numberOfEthereumEventsForwarded: 4,
		}
		for j := 0; j < i; j++ {
			topology.validators["node"+index].heartbeatTracker.blockSigs[j] = true
		}
	}
	topology.validators["node11"] = &valState{
		data: ValidatorData{
			ID:              "node11",
			EthereumAddress: "node11eth",
		},
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}
	topology.validators["node12"] = &valState{
		data: ValidatorData{
			ID:              "node12",
			EthereumAddress: "node12eth",
		},
		status:           ValidatorStatusTendermint,
		heartbeatTracker: &validatorHeartbeatTracker{},
	}

	topology.validatorIncumbentBonusFactor = num.DecimalFromFloat(1.1)
	topology.minimumStake = num.NewUint(3000)
	topology.validatorPerformance = &MockPerformanceScore{}
	topology.numberEthMultisigSigners = 7
	topology.multiSigTopology = &TestMultisigTopology{
		validators: map[string]struct{}{
			"node1eth": {},
			"node2eth": {},
			"node3eth": {},
			"node5eth": {},
			"node7eth": {},
			"node9eth": {},
		},
	}

	delegation := []*types.ValidatorData{
		{NodeID: "node1", PubKey: "node1PubKey", StakeByDelegators: num.NewUint(8000), SelfStake: num.NewUint(2000), Delegators: map[string]*num.Uint{}, TmPubKey: "key1"},
		{NodeID: "node2", PubKey: "node2PubKey", StakeByDelegators: num.NewUint(2000), SelfStake: num.NewUint(3000), Delegators: map[string]*num.Uint{}, TmPubKey: "key2"},
		{NodeID: "node3", PubKey: "node3PubKey", StakeByDelegators: num.NewUint(3000), SelfStake: num.NewUint(7000), Delegators: map[string]*num.Uint{}, TmPubKey: "key3"},
		{NodeID: "node4", PubKey: "node4PubKey", StakeByDelegators: num.NewUint(4000), SelfStake: num.NewUint(11000), Delegators: map[string]*num.Uint{}, TmPubKey: "key4"},
		{NodeID: "node5", PubKey: "node5PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(15000), Delegators: map[string]*num.Uint{}, TmPubKey: "key5"},
		{NodeID: "node6", PubKey: "node6PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(7000), Delegators: map[string]*num.Uint{}, TmPubKey: "key6"},
		{NodeID: "node7", PubKey: "node7PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(6000), Delegators: map[string]*num.Uint{}, TmPubKey: "key7"},
		{NodeID: "node8", PubKey: "node8PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(5000), Delegators: map[string]*num.Uint{}, TmPubKey: "key8"},
		{NodeID: "node9", PubKey: "node9PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(4000), Delegators: map[string]*num.Uint{}, TmPubKey: "key9"},
		{NodeID: "node10", PubKey: "node10PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(3000), Delegators: map[string]*num.Uint{}, TmPubKey: "key10"},
		{NodeID: "node11", PubKey: "node11PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(2000), Delegators: map[string]*num.Uint{}, TmPubKey: "key11"},
		{NodeID: "node12", PubKey: "node12PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(1000), Delegators: map[string]*num.Uint{}, TmPubKey: "key12"},
	}
	topology.rng = rand.New(rand.NewSource(100000))
	optimalScore := num.DecimalFromInt64(10000)
	scoreData, _ := topology.calculateScores(delegation, ValidatorStatusErsatz, types.StakeScoreParams{MinVal: num.DecimalFromFloat(5), CompLevel: num.DecimalFromFloat(3.3), OptimalStakeMultiplier: num.DecimalFromFloat(5)}, &optimalScore)

	require.Equal(t, 10, len(scoreData.RawValScores))
	require.Equal(t, 10, len(scoreData.PerformanceScores))
	require.Equal(t, 10, len(scoreData.MultisigScores))
	require.Equal(t, 10, len(scoreData.ValScores))
	require.Equal(t, 10, len(scoreData.NormalisedScores))

	// raw scores
	// total = 110000
	// opt stake = 10000
	// node1 = 10000/110000 = 0.09090909091
	// node2 = 5000/110000 = 0.04545454545
	// node3 = 10000/110000 = 0.09090909091
	// node4 = 15000/110000 = 10000/110000 = 0.09090909091 (with flat penalty)
	// node5 = 20000/110000 = 10000/110000 = 0.09090909091 (with flat penalty)
	// node6 = 12000/110000 = 10000/110000 = 0.09090909091 (with flat penalty)
	// node7 = 11000/110000 = 10000/110000 = 0.09090909091 (with flat penalty)
	// node8 = 10000/110000 = 0.09090909091
	// node9 = 9000/110000 = 0.08181818182
	// node10 = 8000/110000 = 0.07272727273
	require.Equal(t, "0.09090909", scoreData.RawValScores["node1"].StringFixed(8))
	require.Equal(t, "0.04545455", scoreData.RawValScores["node2"].StringFixed(8))
	require.Equal(t, "0.09090909", scoreData.RawValScores["node3"].StringFixed(8))
	require.Equal(t, "0.09090909", scoreData.RawValScores["node4"].StringFixed(8))
	require.Equal(t, "0.09090909", scoreData.RawValScores["node5"].StringFixed(8))
	require.Equal(t, "0.09090909", scoreData.RawValScores["node6"].StringFixed(8))
	require.Equal(t, "0.09090909", scoreData.RawValScores["node7"].StringFixed(8))
	require.Equal(t, "0.09090909", scoreData.RawValScores["node8"].StringFixed(8))
	require.Equal(t, "0.08181818", scoreData.RawValScores["node9"].StringFixed(8))
	require.Equal(t, "0.07272727", scoreData.RawValScores["node10"].StringFixed(8))

	// performance score
	// node1 0
	// node2 0.1
	// node3 0.2
	// node4 0.3
	// node5 0.4
	// node6 0.5
	// node7 0.6
	// node8 0.7
	// node9 0.8
	// node10 0.9
	require.Equal(t, "0", scoreData.PerformanceScores["node1"].String())
	require.Equal(t, "0.1", scoreData.PerformanceScores["node2"].String())
	require.Equal(t, "0.2", scoreData.PerformanceScores["node3"].String())
	require.Equal(t, "0.3", scoreData.PerformanceScores["node4"].String())
	require.Equal(t, "0.4", scoreData.PerformanceScores["node5"].String())
	require.Equal(t, "0.5", scoreData.PerformanceScores["node6"].String())
	require.Equal(t, "0.6", scoreData.PerformanceScores["node7"].String())
	require.Equal(t, "0.7", scoreData.PerformanceScores["node8"].String())
	require.Equal(t, "0.8", scoreData.PerformanceScores["node9"].String())
	require.Equal(t, "0.9", scoreData.PerformanceScores["node10"].String())

	// multisig score
	// not relevant for ersatz validators should all be 1
	require.Equal(t, "1", scoreData.MultisigScores["node1"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node2"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node3"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node4"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node5"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node6"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node7"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node8"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node9"].String())
	require.Equal(t, "1", scoreData.MultisigScores["node10"].String())

	// val score = stake_score x performance_score:
	// node1 = 0
	// node2 = 0.04545454545 * 0.1 = 0.004545454545
	// node3 = 0.09090909091 * 0.2 = 0.01818181818
	// node4 = 0.09090909091 * 0.3 = 0.02727272727
	// node5 = 0.09090909091 * 0.4 = 0.03636363636
	// node6 = 0.09090909091 * 0.5 = 0.04545454545
	// node7 = 0.09090909091 * 0.6 = 0.05454545455
	// node8 = 0.09090909091 * 0.7 = 0.06363636364
	// node9 = 0.08181818182 * 0.8 = 0.06545454545
	// node10 = 0.07272727273 * 0.9 = 0.06545454545

	// val scores = stake_score * perf_score * multisig_score
	require.Equal(t, "0", scoreData.ValScores["node1"].String())
	require.Equal(t, "0.00454545455", scoreData.ValScores["node2"].StringFixed(11))
	require.Equal(t, "0.01818181818", scoreData.ValScores["node3"].StringFixed(11))
	require.Equal(t, "0.02727272727", scoreData.ValScores["node4"].StringFixed(11))
	require.Equal(t, "0.03636363636", scoreData.ValScores["node5"].StringFixed(11))
	require.Equal(t, "0.04545454545", scoreData.ValScores["node6"].StringFixed(11))
	require.Equal(t, "0.05454545455", scoreData.ValScores["node7"].StringFixed(11))
	require.Equal(t, "0.06363636364", scoreData.ValScores["node8"].StringFixed(11))
	require.Equal(t, "0.06545454545", scoreData.ValScores["node9"].StringFixed(11))
	require.Equal(t, "0.06545454545", scoreData.ValScores["node10"].StringFixed(11))

	// normalised scores
	// node1 = 0
	// node2 = 0.00454545455 / 0.3809090909 = 0.01193317424
	// node3 = 0.01818181818 / 0.3809090909 = 0.0477326969
	// node4 = 0.02727272727 / 0.3809090909 = 0.07159904534
	// node5 = 0.03636363636 / 0.3809090909 = 0.09546539379
	// node6 = 0.04545454545 / 0.3809090909 = 0.1193317422
	// node7 = 0.05454545455 / 0.3809090909 = 0.1431980907
	// node8 = 0.06363636364 / 0.3809090909 = 0.1670644391
	// node9 = 0.06545454545 / 0.3809090909 = 0.1718377088
	// node10 = 0.06545454545 / 0.3809090909 = 0.1718377088
	require.Equal(t, "0.00000000000", scoreData.NormalisedScores["node1"].StringFixed(11))
	require.Equal(t, "0.0119331742", scoreData.NormalisedScores["node2"].StringFixed(10))
	require.Equal(t, "0.0477326969", scoreData.NormalisedScores["node3"].StringFixed(10))
	require.Equal(t, "0.0715990453", scoreData.NormalisedScores["node4"].StringFixed(10))
	require.Equal(t, "0.0954653938", scoreData.NormalisedScores["node5"].StringFixed(10))
	require.Equal(t, "0.1193317422", scoreData.NormalisedScores["node6"].StringFixed(10))
	require.Equal(t, "0.1431980907", scoreData.NormalisedScores["node7"].StringFixed(10))
	require.Equal(t, "0.1670644391", scoreData.NormalisedScores["node8"].StringFixed(10))
	require.Equal(t, "0.1718377088", scoreData.NormalisedScores["node9"].StringFixed(10))
	require.Equal(t, "0.1718377088", scoreData.NormalisedScores["node10"].StringFixed(10))

	totalNormScore := num.DecimalZero()
	for _, d := range scoreData.NormalisedScores {
		totalNormScore = totalNormScore.Add(d)
	}
	require.True(t, totalNormScore.LessThanOrEqual(decimalOne))
}

func testGetRewardsScores(t *testing.T) {
	topology := &Topology{}
	topology.log = logging.NewTestLogger()
	topology.validators = map[string]*valState{}
	topology.validatorIncumbentBonusFactor = num.DecimalFromFloat(1.1)
	topology.minimumStake = num.NewUint(3000)
	topology.validatorPerformance = &MockPerformanceScore{}
	topology.numberEthMultisigSigners = 7
	topology.multiSigTopology = &TestMultisigTopology{
		validators: map[string]struct{}{
			"node1eth": {},
			"node2eth": {},
			"node3eth": {},
			"node5eth": {},
			"node7eth": {},
			"node9eth": {},
		},
	}

	for i := 0; i < 10; i++ {
		index := num.NewUint(uint64(i) + 1).String()
		topology.validators["node"+index] = &valState{
			data: ValidatorData{
				ID:              "node" + index,
				TmPubKey:        "key" + index,
				EthereumAddress: "node" + index + "eth",
			},
			status:           ValidatorStatusTendermint,
			heartbeatTracker: &validatorHeartbeatTracker{},
		}
	}
	for i := 10; i < 20; i++ {
		index := num.NewUint(uint64(i) + 1).String()
		topology.validators["node"+index] = &valState{
			data: ValidatorData{
				ID: "node" + index,
			},
			status:                          ValidatorStatusErsatz,
			numberOfEthereumEventsForwarded: 4,
			heartbeatTracker: &validatorHeartbeatTracker{
				blockSigs: [10]bool{},
			},
		}
		for j := 10; j < i; j++ {
			topology.validators["node"+index].heartbeatTracker.blockSigs[j-10] = true
		}
	}
	topology.rng = rand.New(rand.NewSource(100000))
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	broker := bmocks.NewMockBroker(ctrl)
	topology.broker = broker

	var bEvents []events.Event
	broker.EXPECT().SendBatch(gomock.Any()).Do(func(evnts []events.Event) { bEvents = evnts }).Times(1)

	delegation := []*types.ValidatorData{
		{NodeID: "node1", PubKey: "node1PubKey", StakeByDelegators: num.NewUint(8000), SelfStake: num.NewUint(2000), Delegators: map[string]*num.Uint{}, TmPubKey: "key1"},
		{NodeID: "node2", PubKey: "node2PubKey", StakeByDelegators: num.NewUint(2000), SelfStake: num.NewUint(3000), Delegators: map[string]*num.Uint{}, TmPubKey: "key2"},
		{NodeID: "node3", PubKey: "node3PubKey", StakeByDelegators: num.NewUint(3000), SelfStake: num.NewUint(7000), Delegators: map[string]*num.Uint{}, TmPubKey: "key3"},
		{NodeID: "node4", PubKey: "node4PubKey", StakeByDelegators: num.NewUint(4000), SelfStake: num.NewUint(11000), Delegators: map[string]*num.Uint{}, TmPubKey: "key4"},
		{NodeID: "node5", PubKey: "node5PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(15000), Delegators: map[string]*num.Uint{}, TmPubKey: "key5"},
		{NodeID: "node6", PubKey: "node6PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(7000), Delegators: map[string]*num.Uint{}, TmPubKey: "key6"},
		{NodeID: "node7", PubKey: "node7PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(6000), Delegators: map[string]*num.Uint{}, TmPubKey: "key7"},
		{NodeID: "node8", PubKey: "node8PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(5000), Delegators: map[string]*num.Uint{}, TmPubKey: "key8"},
		{NodeID: "node9", PubKey: "node9PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(4000), Delegators: map[string]*num.Uint{}, TmPubKey: "key9"},
		{NodeID: "node10", PubKey: "node10PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(3000), Delegators: map[string]*num.Uint{}, TmPubKey: "key10"},
		{NodeID: "node11", PubKey: "node11PubKey", StakeByDelegators: num.NewUint(8000), SelfStake: num.NewUint(2000), Delegators: map[string]*num.Uint{}, TmPubKey: "key11"},
		{NodeID: "node12", PubKey: "node12PubKey", StakeByDelegators: num.NewUint(2000), SelfStake: num.NewUint(3000), Delegators: map[string]*num.Uint{}, TmPubKey: "key12"},
		{NodeID: "node13", PubKey: "node13PubKey", StakeByDelegators: num.NewUint(3000), SelfStake: num.NewUint(7000), Delegators: map[string]*num.Uint{}, TmPubKey: "key13"},
		{NodeID: "node14", PubKey: "node14PubKey", StakeByDelegators: num.NewUint(4000), SelfStake: num.NewUint(11000), Delegators: map[string]*num.Uint{}, TmPubKey: "key14"},
		{NodeID: "node15", PubKey: "node15PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(15000), Delegators: map[string]*num.Uint{}, TmPubKey: "key15"},
		{NodeID: "node16", PubKey: "node16PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(7000), Delegators: map[string]*num.Uint{}, TmPubKey: "key16"},
		{NodeID: "node17", PubKey: "node17PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(6000), Delegators: map[string]*num.Uint{}, TmPubKey: "key17"},
		{NodeID: "node18", PubKey: "node18PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(5000), Delegators: map[string]*num.Uint{}, TmPubKey: "key18"},
		{NodeID: "node19", PubKey: "node19PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(4000), Delegators: map[string]*num.Uint{}, TmPubKey: "key19"},
		{NodeID: "node20", PubKey: "node20PubKey", StakeByDelegators: num.NewUint(5000), SelfStake: num.NewUint(3000), Delegators: map[string]*num.Uint{}, TmPubKey: "key20"},
	}

	tmScores, ezScores := topology.GetRewardsScores(context.Background(), "1", delegation, types.StakeScoreParams{MinVal: num.DecimalFromFloat(5), CompLevel: num.DecimalFromFloat(3.3), OptimalStakeMultiplier: num.DecimalFromFloat(5)})

	// tendermint
	require.Equal(t, "0.00000000000", tmScores.NormalisedScores["node1"].StringFixed(11))
	require.Equal(t, "0.08230452675", tmScores.NormalisedScores["node2"].StringFixed(11))
	require.Equal(t, "0.16460905350", tmScores.NormalisedScores["node3"].StringFixed(11))
	require.Equal(t, "0.00000000000", tmScores.NormalisedScores["node4"].StringFixed(11))
	require.Equal(t, "0.32921810700", tmScores.NormalisedScores["node5"].StringFixed(11))
	require.Equal(t, "0.07407407407", tmScores.NormalisedScores["node6"].StringFixed(11))
	require.Equal(t, "0.15843621399", tmScores.NormalisedScores["node7"].StringFixed(11))
	require.Equal(t, "0.06172839506", tmScores.NormalisedScores["node8"].StringFixed(11))
	require.Equal(t, "0.13", tmScores.NormalisedScores["node9"].StringFixed(2))
	require.Equal(t, "0.00000000000", tmScores.NormalisedScores["node10"].StringFixed(11))

	// ersatz
	require.Equal(t, "0.00000000000", ezScores.NormalisedScores["node11"].StringFixed(11))
	require.Equal(t, "0.01020408163", ezScores.NormalisedScores["node12"].StringFixed(11))
	require.Equal(t, "0.04081632653", ezScores.NormalisedScores["node13"].StringFixed(11))
	require.Equal(t, "0.09183673469", ezScores.NormalisedScores["node14"].StringFixed(11))
	require.Equal(t, "0.1632653061", ezScores.NormalisedScores["node15"].StringFixed(10))
	require.Equal(t, "0.1224489796", ezScores.NormalisedScores["node16"].StringFixed(10))
	require.Equal(t, "0.1346938776", ezScores.NormalisedScores["node17"].StringFixed(10))
	require.Equal(t, "0.1428571429", ezScores.NormalisedScores["node18"].StringFixed(10))
	require.Equal(t, "0.1469387755", ezScores.NormalisedScores["node19"].StringFixed(10))
	require.Equal(t, "0.1469387755", ezScores.NormalisedScores["node20"].StringFixed(10))

	require.Equal(t, 20, len(bEvents))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node1", "1", num.DecimalZero(), num.DecimalZero(), num.MustDecimalFromString("0.0909090909090909"), num.DecimalZero(), num.DecimalFromFloat(1), "tendermint"), bEvents[0].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node10", "1", num.DecimalZero(), num.DecimalZero(), num.MustDecimalFromString("0.0727272727272727"), num.DecimalFromFloat(1), num.DecimalZero(), "tendermint"), bEvents[1].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node2", "1", num.MustDecimalFromString("0.0363636363636364"), num.MustDecimalFromString("0.0823045267489713"), num.MustDecimalFromString("0.0454545454545455"), num.DecimalFromFloat(0.8), num.DecimalFromFloat(1), "tendermint"), bEvents[2].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node3", "1", num.MustDecimalFromString("0.07272727272727272"), num.MustDecimalFromString("0.1646090534979424"), num.MustDecimalFromString("0.0909090909090909"), num.DecimalFromFloat(0.8), num.DecimalFromFloat(1), "tendermint"), bEvents[3].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node4", "1", num.DecimalZero(), num.DecimalZero(), num.MustDecimalFromString("0.1363636363636364"), num.DecimalFromFloat(0.8), num.DecimalFromFloat(0), "tendermint"), bEvents[4].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node5", "1", num.MustDecimalFromString("0.14545454545454544"), num.MustDecimalFromString("0.3292181069958847"), num.MustDecimalFromString("0.1818181818181818"), num.DecimalFromFloat(0.8), num.DecimalFromFloat(1), "tendermint"), bEvents[5].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node6", "1", num.MustDecimalFromString("0.03272727272727273"), num.MustDecimalFromString("0.0740740740740741"), num.MustDecimalFromString("0.1090909090909091"), num.DecimalFromFloat(0.3), num.DecimalFromFloat(1), "tendermint"), bEvents[6].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node7", "1", num.MustDecimalFromString("0.07"), num.MustDecimalFromString("0.1584362139917695"), num.MustDecimalFromString("0.1"), num.DecimalFromFloat(0.7), num.DecimalFromFloat(1), "tendermint"), bEvents[7].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node8", "1", num.MustDecimalFromString("0.02727272727272727"), num.MustDecimalFromString("0.0617283950617284"), num.MustDecimalFromString("0.0909090909090909"), num.DecimalFromFloat(0.3), num.DecimalFromFloat(1), "tendermint"), bEvents[8].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node9", "1", num.MustDecimalFromString("0.05727272727272726"), num.MustDecimalFromString("0.1296296296296296"), num.MustDecimalFromString("0.0818181818181818"), num.DecimalFromFloat(0.7), num.DecimalFromFloat(1), "tendermint"), bEvents[9].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node11", "1", num.DecimalZero(), num.DecimalZero(), num.MustDecimalFromString("0.0909090909090909"), num.DecimalZero(), num.DecimalFromFloat(1), "ersatz"), bEvents[10].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node12", "1", num.MustDecimalFromString("0.00454545454545455"), num.MustDecimalFromString("0.0102040816326531"), num.MustDecimalFromString("0.0454545454545455"), num.DecimalFromFloat(0.1), num.DecimalFromFloat(1), "ersatz"), bEvents[11].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node13", "1", num.MustDecimalFromString("0.01818181818181818"), num.MustDecimalFromString("0.0408163265306122"), num.MustDecimalFromString("0.0909090909090909"), num.DecimalFromFloat(0.2), num.DecimalFromFloat(1), "ersatz"), bEvents[12].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node14", "1", num.MustDecimalFromString("0.04090909090909092"), num.MustDecimalFromString("0.0918367346938776"), num.MustDecimalFromString("0.1363636363636364"), num.DecimalFromFloat(0.3), num.DecimalFromFloat(1), "ersatz"), bEvents[13].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node15", "1", num.MustDecimalFromString("0.07272727272727272"), num.MustDecimalFromString("0.163265306122449"), num.MustDecimalFromString("0.1818181818181818"), num.DecimalFromFloat(0.4), num.DecimalFromFloat(1), "ersatz"), bEvents[14].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node16", "1", num.MustDecimalFromString("0.05454545454545455"), num.MustDecimalFromString("0.1224489795918368"), num.MustDecimalFromString("0.1090909090909091"), num.DecimalFromFloat(0.5), num.DecimalFromFloat(1), "ersatz"), bEvents[15].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node17", "1", num.MustDecimalFromString("0.06"), num.MustDecimalFromString("0.1346938775510204"), num.MustDecimalFromString("0.1"), num.DecimalFromFloat(0.6), num.DecimalFromFloat(1), "ersatz"), bEvents[16].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node18", "1", num.MustDecimalFromString("0.06363636363636363"), num.MustDecimalFromString("0.1428571428571429"), num.MustDecimalFromString("0.0909090909090909"), num.DecimalFromFloat(0.7), num.DecimalFromFloat(1), "ersatz"), bEvents[17].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node19", "1", num.MustDecimalFromString("0.06545454545454544"), num.MustDecimalFromString("0.1469387755102041"), num.MustDecimalFromString("0.0818181818181818"), num.DecimalFromFloat(0.8), num.DecimalFromFloat(1), "ersatz"), bEvents[18].(*events.ValidatorScore))
	verifyEvent(t, events.NewValidatorScore(context.Background(), "node20", "1", num.MustDecimalFromString("0.06545454545454543"), num.MustDecimalFromString("0.1469387755102039"), num.MustDecimalFromString("0.0727272727272727"), num.DecimalFromFloat(0.9), num.DecimalFromFloat(1), "ersatz"), bEvents[19].(*events.ValidatorScore))
}

func verifyEvent(t *testing.T, expected, actual *events.ValidatorScore) {
	t.Helper()
	require.Equal(t, expected.EpochSeq, actual.EpochSeq)
	require.Equal(t, expected.ValidatorScore, actual.ValidatorScore)
	require.Equal(t, expected.MultisigScore, actual.MultisigScore)
	require.Equal(t, expected.NodeID, actual.NodeID)
	require.Equal(t, expected.NormalisedScore, actual.NormalisedScore)
	require.Equal(t, expected.RawValidatorScore, actual.RawValidatorScore)
	require.Equal(t, expected.ValidatorPerformance, actual.ValidatorPerformance)
	require.Equal(t, expected.ValidatorStatus, actual.ValidatorStatus)
}

func TestAddressMapping(t *testing.T) {
	tmKey := "7kRL1jCJH8QUDTHK90/Nz9lIAvl8/s1Z70XL1EXFkaM="
	require.Equal(t, "13fa0b679d6064772567c7a6050b42cca1c7c8cd", tmPubKeyToAddress(tmKey))
}

type MockPerformanceScore struct{}

func (*MockPerformanceScore) ValidatorPerformanceScore(tmPubKey string, power, totalPower int64, scalingFactor num.Decimal) num.Decimal {
	if tmPubKey == "key6" || tmPubKey == "key8" {
		return num.DecimalFromFloat(0.3)
	}
	if tmPubKey == "key7" || tmPubKey == "key9" {
		return num.DecimalFromFloat(0.7)
	}
	if tmPubKey == "key10" {
		return num.DecimalFromFloat(1)
	}
	return num.DecimalFromFloat(0.8)
}

func (*MockPerformanceScore) BeginBlock(ctx context.Context, proposer string) {
}

func (*MockPerformanceScore) Serialize() *v1.ValidatorPerformance {
	return nil
}
func (*MockPerformanceScore) Deserialize(*v1.ValidatorPerformance) {}

func (*MockPerformanceScore) Reset() {}
