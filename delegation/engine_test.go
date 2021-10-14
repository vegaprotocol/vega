package delegation

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEngine struct {
	engine          *Engine
	ctrl            *gomock.Controller
	broker          *mocks.MockBroker
	stakingAccounts *TestStakingAccount
	topology        *TestTopology
}

func Test(t *testing.T) {
	// delegate tests
	t.Run("Delegation to an unknown node fails", testDelegateInvalidNode)
	t.Run("Delegation with no staking account fails", testDelegateNoStakingAccount)
	t.Run("Delegation of an amount lower than min delegation amount fails", testDelegateLessThanMinDelegationAmount)
	t.Run("Delegation of an amount greater than the available balance for delegation - no pending delegations, no active delegations fails", testDelegateInsufficientBalanceNoPendingNoCommitted)
	t.Run("Delegation of an amount greater than the available balance for delegation - with pending delegations, no active delegations fails", testDelegateInsufficientBalanceIncludingPendingDelegation)
	t.Run("Delegation of an amount greater than the available balance for delegation - no pending delegations, with active delegations fails", testDelegateInsufficientBalanceIncludingCommitted)
	t.Run("Delegation of an amount greater than the available balance for delegation - with pending delegations and active delegations fails", testDelegateInsufficientBalanceIncludingPendingAndCommitted)
	t.Run("Delegation of an amount greater than the available balance for delegation - with pending undelegations and active delegations fails", testDelegateInsufficientBalanceIncludingPendingUndelegations)
	t.Run("Delegation of an amount less than the available balance for delegation - with no previous active delegation succeeds", testDelegateSuccesNoCommitted)
	t.Run("Delegation of an amount less than the available balance for delegation with previous pending undelegations covered by delegation succeeds", testDelegateSuccessWithPreviousPendingUndelegateFullyCovered)
	t.Run("Delegation of an amount less than the available balance for delegation with previous pending undelegations covered partly by delegation succeeds", testDelegateSuccessWithPreviousPendingUndelegatePartiallyCovered)
	t.Run("Delegation of an amount less than the available balance for delegation with previous pending undelegations countering exactly the undelegated amount succeeds", testDelegateSuccessWithPreviousPendingUndelegateExactlyCovered)
	t.Run("Delegation of an amount fails due to insufficient funds to cover existing committed delegations", testDelegateInsufficientBalanceCoveringExisting)
	t.Run("Delegation of an amount fails due to insufficient funds to cover existing pending delegations", testDelegateInsufficientBalanceCoveringPending)

	// undelegate tests
	t.Run("Undelegation to an unknown node fails", testUndelegateInvalidNode)
	t.Run("Undelegation more than the delegated balance fails", testUndelegateInvalidAmount)
	t.Run("Undelegate incrememtntally the whole delegated balance succeeds", testUndelegateSuccessNoPreviousPending)
	t.Run("Undelegate incrememtntally with pending exactly covered by undelegate succeeds", testUndelegateSuccessWithPreviousPendingDelegateExactlyCovered)
	t.Run("Undelegate with pending delegated covered partly succeeds", testUndelegateSuccessWithPreviousPendingDelegatePartiallyCovered)
	t.Run("Undelegate with pending delegated fully covered succeeds", testUndelegateSuccessWithPreviousPendingDelegateFullyCovered)

	// undelegatenow tests
	t.Run("Undelegate now of an amount larger than available fails", testUndelegateNowIncorrectAmount)
	t.Run("Undelegate all with only pending delegation succeeds", testUndelegateNowAllWithPendingOnly)
	t.Run("Undelegate all with only active delegation succeeds", testUndelegateNowAllWithCommittedOnly)
	t.Run("Undelegate all with both active and pending delegation succeeds", testUndelegateNowAll)
	t.Run("Undelegate an amount with pending only delegation succeeds", testUndelegateNowWithPendingOnly)
	t.Run("Undelegate an amount with active only delegation succeeds", testUndelegateNowWithCommittedOnly)
	t.Run("Undelegate an amount with both active and pending delegation - sufficient cover in pending succeeds", testUndelegateNowPendingCovers)
	t.Run("Undelegate an amount with both active and pending delegation - insufficient cover in pending succeeds", testUndelegateNowCommittedCovers)
	t.Run("Undelegate an amount with both active and pending delegation - all delegation removed", testUndelegateNowAllCleared)

	// test preprocess
	t.Run("preprocess with no forced undelegation needed", testPreprocessForRewardingNoForcedUndelegationNeeded)
	t.Run("preprocess with forced undelegation needed single validator node", testPreprocessForRewardingWithForceUndelegateSingleValidator)
	t.Run("preprocess with forced undelegation needed multiple validator nodes with no remainder", testPreprocessForRewardingWithForceUndelegateMultiValidatorNoRemainder)
	t.Run("preprocess with forced undelegation needed multiple validator nodes with remainder", testPreprocessForRewardingWithForceUndelegateMultiValidatorWithRemainder)

	// test process pending undelegation
	t.Run("process pending undelegations empty succeeds", testPendingUndelegationEmpty)
	t.Run("process pending undelegations with nothing left to undelegate succeeds", testPendingUndelegationNothingToUndelegate)
	t.Run("process pending undelegations with more than the delegated balance succeeds", testPendingUndelegationGTDelegateddBalance)
	t.Run("process pending undelegations with less than the delegated succeeds", testPendingUndelegationLTDelegateddBalance)
	t.Run("process pending undelegations undeledate everything for a party succeeds", testPendingUndelegationAllBalanceForParty)
	t.Run("process pending undelegations undeledate everything for a node succeeds", testPendingUndelegationAllBalanceForNode)

	// test process pending delegation
	t.Run("process pending delegations empty succeeds", testPendingDelegationEmpty)
	t.Run("process pending delegations with insufficient staking account balance ignored", testPendingDelegationInsufficientBalance)
	t.Run("process pending delegations with no space left on validator ignored", testPendingDelegationValidatorAllocationMaxedOut)
	t.Run("process pending delegations amount adjusted to fit the validator allocation upper bound", testPendingDelegationAmountAdjusted)
	t.Run("process pending delegations no adjustment", testPendingDelegationSuccess)

	// test process pending
	t.Run("process pending is delegating and undelegating and clearing the pending state successfully", testProcessPending)

	// test get validators
	t.Run("get empty list of validators succeeds", testGetValidatorsEmpty)
	t.Run("get list of validators succeeds", testGetValidatorsSuccess)
	t.Run("setup delegation with self and parties", testGetValidatorsSuccessWithSelfDelegation)

	// test calculation of total delegated tokens
	t.Run("calculate total delegated token successful", testCalculateTotalDelegatedTokens)
	t.Run("calculate the max stake per validator", testMaxStakePerValidator)

	// test auto delegation
	t.Run("a party having all of their stake delegated get into auto delegation successfully", testCheckPartyEnteringAutoDelegation)
	t.Run("a party is in auto delegation mode which gets cancelled by manually undelegating at the end of an epoch", testCheckPartyExitingAutoDelegationThroughUndelegateEOE)
	t.Run("a party is in auto delegation mode which gets cancelled by manually undelegating during an epoch", testCheckPartyExitingAutoDelegationThroughUndelegateNow)

	// test checkpoints
	t.Run("sorting consistently active delegations for checkpoint", testSortActive)
	t.Run("sorting consistently pending delegations for checkpoint", testSortPending)
	t.Run("test roundtrip of checkpoint calculation with no pending delegations", testCheckpointRoundtripNoPending)
	t.Run("test roundtrip of checkpoint calculation with no active delegations", testCheckpointRoundtripOnlyPending)

	// test snapshots
	t.Run("test roundtrip snapshot for active delegations", testActiveSnapshotRoundTrip)
	t.Run("test roundtrip snapshot for pending delegations", testPendingSnapshotRoundTrip)
	t.Run("test roundtrip snapshot for auto delegations", testAutoSnapshotRoundTrip)
}

// test round trip of active snapshot hash and serialisation.
func testActiveSnapshotRoundTrip(t *testing.T) {
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 14, 7)

	testEngine.engine.ProcessEpochDelegations(context.Background(), types.Epoch{Seq: 0})

	// get the has and serialised state
	hash, err := testEngine.engine.GetHash(activeKey)
	require.Nil(t, err)
	state, err := testEngine.engine.GetState(activeKey)
	require.Nil(t, err)

	// verify hash is consistent in the absence of change
	hashNoChange, err := testEngine.engine.GetHash(activeKey)
	require.Nil(t, err)
	stateNoChange, err := testEngine.engine.GetState(activeKey)
	require.Nil(t, err)

	require.True(t, bytes.Equal(hash, hashNoChange))
	require.True(t, bytes.Equal(state, stateNoChange))

	// reload the state
	var active snapshot.Payload
	proto.Unmarshal(state, &active)
	payload := types.PayloadFromProto(&active)
	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	testEngine.engine.LoadState(context.Background(), payload)

	// verify hash and state match
	hashPostReload, _ := testEngine.engine.GetHash(activeKey)
	require.True(t, bytes.Equal(hash, hashPostReload))
	statePostReload, _ := testEngine.engine.GetState(activeKey)
	require.True(t, bytes.Equal(state, statePostReload))
}

// test round trip of pending snapshot hash and serialisation.
func testPendingSnapshotRoundTrip(t *testing.T) {
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 20, 7)

	// setup pending delegations
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(3))
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node1", num.NewUint(1))

	// get the has and serialised state
	hash, err := testEngine.engine.GetHash(pendingKey)
	require.Nil(t, err)
	state, err := testEngine.engine.GetState(pendingKey)
	require.Nil(t, err)

	// verify hash is consistent in the absence of change
	hashNoChange, err := testEngine.engine.GetHash(pendingKey)
	require.Nil(t, err)
	stateNoChange, err := testEngine.engine.GetState(pendingKey)
	require.Nil(t, err)

	require.True(t, bytes.Equal(hash, hashNoChange))
	require.True(t, bytes.Equal(state, stateNoChange))

	// reload the state
	var pending snapshot.Payload
	proto.Unmarshal(state, &pending)
	payload := types.PayloadFromProto(&pending)
	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	err = testEngine.engine.LoadState(context.Background(), payload)
	require.Nil(t, err)
	hashPostReload, _ := testEngine.engine.GetHash(pendingKey)
	require.True(t, bytes.Equal(hash, hashPostReload))
	statePostReload, _ := testEngine.engine.GetState(pendingKey)
	require.True(t, bytes.Equal(state, statePostReload))
}

// test round trip of auto snapshot hash and serialisation.
func testAutoSnapshotRoundTrip(t *testing.T) {
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 10, 5)

	testEngine.engine.ProcessEpochDelegations(context.Background(), types.Epoch{Seq: 0})

	// by now, auto delegation should be set for both party1 and party2 as all of their association is nominated
	hash, err := testEngine.engine.GetHash(autoKey)
	require.Nil(t, err)
	state, err := testEngine.engine.GetState(autoKey)
	require.Nil(t, err)

	// verify hash is consistent in the absence of change
	hashNoChange, err := testEngine.engine.GetHash(autoKey)
	require.Nil(t, err)
	stateNoChange, err := testEngine.engine.GetState(autoKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(hash, hashNoChange))
	require.True(t, bytes.Equal(state, stateNoChange))

	// undelegate now to get out of auto for party1 to verify hash changed
	testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.NewUint(3))
	hashPostUndelegate, err := testEngine.engine.GetHash(autoKey)
	require.Nil(t, err)
	statePostUndelegate, err := testEngine.engine.GetState(autoKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(hash, hashPostUndelegate))
	require.False(t, bytes.Equal(state, statePostUndelegate))

	// reload the state
	var auto snapshot.Payload
	proto.Unmarshal(statePostUndelegate, &auto)
	payload := types.PayloadFromProto(&auto)
	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)

	testEngine.engine.LoadState(context.Background(), payload)
	hashPostReload, _ := testEngine.engine.GetHash(autoKey)
	require.True(t, bytes.Equal(hashPostUndelegate, hashPostReload))
	statePostReload, _ := testEngine.engine.GetState(autoKey)
	require.True(t, bytes.Equal(statePostUndelegate, statePostReload))
}

// pass an invalid node id
// expect an ErrInvalidNodeID.
func testDelegateInvalidNode(t *testing.T) {
	testEngine := getEngine(t)
	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(10))
	assert.EqualError(t, err, ErrInvalidNodeID.Error())
}

// pass a party with no staking account
// expect ErrPartyHasNoStakingAccount.
func testDelegateNoStakingAccount(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(10))
	assert.EqualError(t, err, ErrPartyHasNoStakingAccount.Error())
}

// try to delegate less than the network param for min delegation amount
// expect ErrAmountLTMinAmountForDelegation.
func testDelegateLessThanMinDelegationAmount(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(5)
	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(1))
	assert.EqualError(t, err, ErrAmountLTMinAmountForDelegation.Error())
}

// party has insufficient balance in their staking account to delegate - they have nothing pending and no committed delegation
// expect ErrInsufficientBalanceForDelegation.
func testDelegateInsufficientBalanceNoPendingNoCommitted(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(5)
	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(10))
	assert.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())
}

func testDelegateInsufficientBalanceCoveringExisting(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// party1 has delegation of 10 by now
	// party1 withraws from their staking account
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(5)

	// now they don't have enough cover to their active delegations
	// trying to delegate the min amount should error with insufficient balance
	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))

	assert.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())
}

func testDelegateInsufficientBalanceCoveringPending(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(10)

	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(5))
	require.Nil(t, err)
	err = testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(5))
	require.Nil(t, err)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node2", num.NewUint(2))
	require.Nil(t, err)

	// so party1 has 8 pending delegations in total and they withdraw 5 from their staking account
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(5)

	// trying to delegate min amount should error with insufficient balance
	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	assert.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	err = testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(2))
	assert.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())
}

// party has pending delegations and is trying to exceed their stake account balance delegation, i.e. the balance of their pending delegation + requested delegation exceeds stake account balance.
func testDelegateInsufficientBalanceIncludingPendingDelegation(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true

	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(10)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(7)

	// setup pending
	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(5))
	require.Nil(t, err)

	err = testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(3))
	require.Nil(t, err)

	err = testEngine.engine.Delegate(context.Background(), "party2", "node1", num.NewUint(6))
	require.Nil(t, err)

	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)

	// by this point party1 has delegated 10 and party2 has delegate 6 which means party1 cannot delegage anything anymore and party2 can deleagate no more than 1
	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	assert.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	err = testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(2))
	assert.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	err = testEngine.engine.Delegate(context.Background(), "party2", "node1", num.NewUint(2))
	assert.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	err = testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(2))
	assert.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())
}

// setup committed deletations (delegations in effect in current epoch):
// node1 -> 8
// 		    party1 -> 6
//			party2 -> 2
// node 2 -> 7
// 			party1 -> 4
//			party2 -> 3
func setupDefaultDelegationState(testEngine *testEngine, party1Balance uint64, party2Balance uint64) {
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(party1Balance)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(party2Balance)

	engine := testEngine.engine

	engine.nodeDelegationState["node1"] = &validatorDelegation{
		nodeID:         "node1",
		totalDelegated: num.NewUint(8),
		partyToAmount:  make(map[string]*num.Uint),
	}
	engine.nodeDelegationState["node1"].partyToAmount["party1"] = num.NewUint(6)
	engine.nodeDelegationState["node1"].partyToAmount["party2"] = num.NewUint(2)

	// setup delegation for node2
	engine.nodeDelegationState["node2"] = &validatorDelegation{
		nodeID:         "node2",
		totalDelegated: num.NewUint(7),
		partyToAmount:  make(map[string]*num.Uint),
	}
	engine.nodeDelegationState["node2"].partyToAmount["party1"] = num.NewUint(4)
	engine.nodeDelegationState["node2"].partyToAmount["party2"] = num.NewUint(3)

	engine.partyDelegationState["party1"] = &partyDelegation{
		party:          "party1",
		totalDelegated: num.NewUint(10),
		nodeToAmount:   make(map[string]*num.Uint),
	}
	engine.partyDelegationState["party1"].nodeToAmount["node1"] = num.NewUint(6)
	engine.partyDelegationState["party1"].nodeToAmount["node2"] = num.NewUint(4)

	engine.partyDelegationState["party2"] = &partyDelegation{
		party:          "party2",
		totalDelegated: num.NewUint(5),
		nodeToAmount:   make(map[string]*num.Uint),
	}
	engine.partyDelegationState["party2"].nodeToAmount["node1"] = num.NewUint(2)
	engine.partyDelegationState["party2"].nodeToAmount["node2"] = num.NewUint(3)
}

// setup committed deletations (delegations in effect in current epoch):
// node1 -> 6
// 		    party1 -> 6
// node 2 -> 3
// 			party2 -> 3
func defaultSimpleDelegationState(testEngine *testEngine, party1Balance, party2Balance uint64) {
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(12)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(7)

	engine := testEngine.engine
	engine.nodeDelegationState["node1"] = &validatorDelegation{
		nodeID:         "node1",
		totalDelegated: num.NewUint(6),
		partyToAmount:  make(map[string]*num.Uint),
	}
	engine.nodeDelegationState["node1"].partyToAmount["party1"] = num.NewUint(6)

	// setup delegation for node2
	engine.nodeDelegationState["node2"] = &validatorDelegation{
		nodeID:         "node2",
		totalDelegated: num.NewUint(3),
		partyToAmount:  make(map[string]*num.Uint),
	}
	engine.nodeDelegationState["node2"].partyToAmount["party2"] = num.NewUint(3)

	engine.partyDelegationState["party1"] = &partyDelegation{
		party:          "party1",
		totalDelegated: num.NewUint(6),
		nodeToAmount:   make(map[string]*num.Uint),
	}
	engine.partyDelegationState["party1"].nodeToAmount["node1"] = num.NewUint(6)

	engine.partyDelegationState["party2"] = &partyDelegation{
		party:          "party2",
		totalDelegated: num.NewUint(3),
		nodeToAmount:   make(map[string]*num.Uint),
	}
	engine.partyDelegationState["party2"].nodeToAmount["node2"] = num.NewUint(3)
}

// party has committed delegations and is trying to exceed their stake account balance delegations i.e. the balance of their pending delegation + requested delegation exceeds stake account balance.
func testDelegateInsufficientBalanceIncludingCommitted(t *testing.T) {
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 10, 7)

	// by this point party1 has 10 tokens delegated which means they can't delegate anything more
	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	require.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	err = testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(2))
	require.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	// by this point party2 has 5 tokens delegated which means they can delegate 2 more
	err = testEngine.engine.Delegate(context.Background(), "party2", "node1", num.NewUint(3))
	require.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	err = testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(3))
	require.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())
}

// party has both committed delegations and pending delegations and an additional delegation will exceed the amount of available tokens for delegations in their staking account.
func testDelegateInsufficientBalanceIncludingPendingAndCommitted(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// setup pending
	// by this point party1 has 10 tokens delegated which means they can delegate 2 more
	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)

	// by this point party2 has 5 tokens delegated which means they can delegate 2 more
	err = testEngine.engine.Delegate(context.Background(), "party2", "node1", num.NewUint(2))
	require.Nil(t, err)

	// both parties maxed out their delegation balance
	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	require.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	err = testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(2))
	require.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	err = testEngine.engine.Delegate(context.Background(), "party2", "node1", num.NewUint(2))
	require.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	err = testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(2))
	require.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())
}

// party has both committed delegations and pending undelegations.
func testDelegateInsufficientBalanceIncludingPendingUndelegations(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// setup pending
	// by this point party1 has 10 tokens delegated which means they can delegate 2 more - with the undelegation they can delegate 4
	err := testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)

	// by this point party2 has 5 tokens delegated which means they can delegate 2 more - with undelegation they can delegate 4
	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node1", num.NewUint(2))
	require.Nil(t, err)

	// try to delegate 1 more than available balance for delegation should fall
	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(5))
	require.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	err = testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(5))
	require.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	err = testEngine.engine.Delegate(context.Background(), "party2", "node1", num.NewUint(5))
	require.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	err = testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(5))
	require.EqualError(t, err, ErrInsufficientBalanceForDelegation.Error())

	// now delegate exacatly the balance available for delegation for success
	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)

	err = testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(2))
	require.Nil(t, err)

	err = testEngine.engine.Delegate(context.Background(), "party2", "node1", num.NewUint(2))
	require.Nil(t, err)

	err = testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(2))
	require.Nil(t, err)
}

// balance available for delegation is greater than delegation amount, delegation succeeds.
func testDelegateSuccesNoCommitted(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(10)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(7)

	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(5))
	require.Nil(t, err)

	err = testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(3))
	require.Nil(t, err)

	err = testEngine.engine.Delegate(context.Background(), "party2", "node1", num.NewUint(6))
	require.Nil(t, err)

	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)

	// summary:
	// party1 delegated 10 in total, 7 to node1 and 3 to node2
	// party2 delegated 6 in total, all to node1
	// verify the state

	pendingStateForEpoch := testEngine.engine.pendingState[1]

	require.Equal(t, num.NewUint(10), pendingStateForEpoch["party1"].totalDelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party1"].totalUndelegation)
	require.Equal(t, num.NewUint(6), pendingStateForEpoch["party2"].totalDelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party2"].totalUndelegation)
	require.Equal(t, num.NewUint(7), pendingStateForEpoch["party1"].nodeToDelegateAmount["node1"])
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party1"].nodeToDelegateAmount["node2"])
	require.Equal(t, num.NewUint(6), pendingStateForEpoch["party2"].nodeToDelegateAmount["node1"])
	require.Equal(t, 0, len(pendingStateForEpoch["party1"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party2"].nodeToUndelegateAmount))
	require.Equal(t, 2, len(pendingStateForEpoch["party1"].nodeToDelegateAmount))
	require.Equal(t, 1, len(pendingStateForEpoch["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(pendingStateForEpoch))
	require.Equal(t, 0, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 0, len(testEngine.engine.partyDelegationState))
}

// test delegation when there is already pending undelegation but the deletation is more than fully covering the pending undelegation.
func testDelegateSuccessWithPreviousPendingUndelegateFullyCovered(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	defaultSimpleDelegationState(testEngine, 12, 7)

	// setup pending undelegation
	err := testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(2))
	require.Nil(t, err)

	// show that the state before delegation matches expectation (i.e. that we have 2 for undelegation from party1 and party2 to node1 and node2 respectively)
	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party1"].totalUndelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party1"].totalDelegation)
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party2"].totalUndelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party2"].totalDelegation)
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party1"].nodeToUndelegateAmount["node1"])
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party2"].nodeToUndelegateAmount["node2"])
	require.Equal(t, 1, len(pendingStateForEpoch["party1"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(pendingStateForEpoch["party2"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(pendingStateForEpoch))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))

	// delegte 4 from party 1 to node 1
	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(4))
	require.Nil(t, err)

	// delegate 5 from party 2 to node2
	err = testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(5))
	require.Nil(t, err)

	// summary:
	// verify the state
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party1"].totalDelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party1"].totalUndelegation)
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party2"].totalDelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party2"].totalUndelegation)
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party1"].nodeToDelegateAmount["node1"])
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party2"].nodeToDelegateAmount["node2"])
	require.Equal(t, 0, len(pendingStateForEpoch["party1"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party2"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(pendingStateForEpoch["party1"].nodeToDelegateAmount))
	require.Equal(t, 1, len(pendingStateForEpoch["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(pendingStateForEpoch))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))
}

// test delegation when there is already pending undelegation and the delegation is covering part of the undelegation.
func testDelegateSuccessWithPreviousPendingUndelegatePartiallyCovered(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	defaultSimpleDelegationState(testEngine, 12, 7)

	// setup pending undelegation
	err := testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(4))
	require.Nil(t, err)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(3))
	require.Nil(t, err)

	// show that the state before delegation matches expectation (i.e. that we have 2 for undelegation from party1 and party2 to node1 and node2 respectively)
	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, num.NewUint(4), pendingStateForEpoch["party1"].totalUndelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party1"].totalDelegation)
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party2"].totalUndelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party2"].totalDelegation)
	require.Equal(t, num.NewUint(4), pendingStateForEpoch["party1"].nodeToUndelegateAmount["node1"])
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party2"].nodeToUndelegateAmount["node2"])
	require.Equal(t, 1, len(pendingStateForEpoch["party1"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(pendingStateForEpoch["party2"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(pendingStateForEpoch))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))

	// delegte 3 (< the pending undelegation of 4) from party 1 to node 1
	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(3))
	require.Nil(t, err)

	// delegate 2 (< the pending undelegation of 3) from party 2 to node2
	err = testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(2))
	require.Nil(t, err)

	// verify the state
	require.Equal(t, num.Zero(), pendingStateForEpoch["party1"].totalDelegation)
	require.Equal(t, num.NewUint(1), pendingStateForEpoch["party1"].totalUndelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party2"].totalDelegation)
	require.Equal(t, num.NewUint(1), pendingStateForEpoch["party2"].totalUndelegation)
	require.Equal(t, num.NewUint(1), pendingStateForEpoch["party1"].nodeToUndelegateAmount["node1"])
	require.Equal(t, num.NewUint(1), pendingStateForEpoch["party2"].nodeToUndelegateAmount["node2"])
	require.Equal(t, 0, len(pendingStateForEpoch["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party2"].nodeToDelegateAmount))
	require.Equal(t, 1, len(pendingStateForEpoch["party1"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(pendingStateForEpoch["party2"].nodeToUndelegateAmount))
	require.Equal(t, 2, len(pendingStateForEpoch))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))
}

// test delegation when there is already pending undelegation and the delegation is countering exactly the undelegation.
func testDelegateSuccessWithPreviousPendingUndelegateExactlyCovered(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	defaultSimpleDelegationState(testEngine, 12, 7)

	// setup pending undelegation
	err := testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(4))
	require.Nil(t, err)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(3))
	require.Nil(t, err)

	pendingStateForEpoch := testEngine.engine.pendingState[1]
	// show that the state before delegation matches expectation (i.e. that we have 2 for undelegation from party1 and party2 to node1 and node2 respectively)
	require.Equal(t, num.NewUint(4), pendingStateForEpoch["party1"].totalUndelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party1"].totalDelegation)
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party2"].totalUndelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party2"].totalDelegation)
	require.Equal(t, num.NewUint(4), pendingStateForEpoch["party1"].nodeToUndelegateAmount["node1"])
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party2"].nodeToUndelegateAmount["node2"])
	require.Equal(t, 1, len(pendingStateForEpoch["party1"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(pendingStateForEpoch["party2"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(pendingStateForEpoch))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))

	// delegte 4 (= the pending undelegation of 4) from party 1 to node 1
	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(4))
	require.Nil(t, err)

	// delegate 3 (= the pending undelegation of 3) from party 2 to node2
	err = testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(3))
	require.Nil(t, err)

	// verify the state
	// as we've countered all undelegation we expect the pending state to be empty
	require.Equal(t, 0, len(pendingStateForEpoch))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))
}

/// undelegate.
func testUndelegateInvalidNode(t *testing.T) {
	testEngine := getEngine(t)
	err := testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(10))
	assert.EqualError(t, err, ErrInvalidNodeID.Error())
}

// trying to undelegate more than the delegated amount when no undelegation or more than the delegated - undelegated if there are some.
func testUndelegateInvalidAmount(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(10)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(7)

	// first try undelegate with no delegation at all
	err := testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(10))
	assert.Error(t, err, ErrIncorrectTokenAmountForUndelegation)

	// now delegate some token to node1 and try to undelegate more than the balance
	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(5))
	assert.Nil(t, err)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(6))
	assert.Error(t, err, ErrIncorrectTokenAmountForUndelegation)
}

// trying to undelegate then incresae the undelegated amount until all is undelegated.
func testUndelegateSuccessNoPreviousPending(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	defaultSimpleDelegationState(testEngine, 12, 7)

	// setup pending undelegation
	err := testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(2))
	require.Nil(t, err)

	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party1"].totalUndelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party1"].totalDelegation)
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party2"].totalUndelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party2"].totalDelegation)
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party1"].nodeToUndelegateAmount["node1"])
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party2"].nodeToUndelegateAmount["node2"])
	require.Equal(t, 1, len(pendingStateForEpoch["party1"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(pendingStateForEpoch["party2"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(pendingStateForEpoch))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))

	// undelegate everything now
	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(4))
	require.Nil(t, err)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(1))
	require.Nil(t, err)

	// check that the state has updated correctly
	require.Equal(t, num.NewUint(6), pendingStateForEpoch["party1"].totalUndelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party1"].totalDelegation)
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party2"].totalUndelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party2"].totalDelegation)
	require.Equal(t, num.NewUint(6), pendingStateForEpoch["party1"].nodeToUndelegateAmount["node1"])
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party2"].nodeToUndelegateAmount["node2"])
	require.Equal(t, 1, len(pendingStateForEpoch["party1"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(pendingStateForEpoch["party2"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(pendingStateForEpoch))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))

	// trying to further undelegate will get an error
	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(1))
	assert.Error(t, err, ErrIncorrectTokenAmountForUndelegation)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(1))
	assert.Error(t, err, ErrIncorrectTokenAmountForUndelegation)
}

// delegate an amount that leave some delegation for the party.
func testUndelegateSuccessWithPreviousPendingDelegatePartiallyCovered(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(12)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(7)

	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(10))
	require.Nil(t, err)
	err = testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(2))
	require.Nil(t, err)
	err = testEngine.engine.Delegate(context.Background(), "party2", "node1", num.NewUint(4))
	require.Nil(t, err)
	err = testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(3))
	require.Nil(t, err)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(6))
	require.Nil(t, err)
	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(4))
	require.Nil(t, err)
	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node1", num.NewUint(4))
	require.Nil(t, err)

	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, num.Zero(), pendingStateForEpoch["party1"].totalUndelegation)
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party1"].totalDelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party2"].totalUndelegation)
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party2"].totalDelegation)
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party1"].nodeToDelegateAmount["node2"])
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party2"].nodeToDelegateAmount["node2"])
	require.Equal(t, 0, len(pendingStateForEpoch["party1"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party2"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(pendingStateForEpoch["party1"].nodeToDelegateAmount))
	require.Equal(t, 1, len(pendingStateForEpoch["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(pendingStateForEpoch))
	require.Equal(t, 0, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 0, len(testEngine.engine.partyDelegationState))
}

// undelegate incrementally to get all pending delegates countered.
func testUndelegateSuccessWithPreviousPendingDelegateExactlyCovered(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(12)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(7)

	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(10))
	require.Nil(t, err)
	err = testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(2))
	require.Nil(t, err)
	err = testEngine.engine.Delegate(context.Background(), "party2", "node1", num.NewUint(4))
	require.Nil(t, err)
	err = testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(3))
	require.Nil(t, err)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(6))
	require.Nil(t, err)
	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(4))
	require.Nil(t, err)
	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node2", num.NewUint(2))
	require.Nil(t, err)
	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node1", num.NewUint(4))
	require.Nil(t, err)
	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(3))
	require.Nil(t, err)

	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, 0, len(pendingStateForEpoch))
}

// undelegate such that delegation for some party and node goes from delegate to undelegate.
func testUndelegateSuccessWithPreviousPendingDelegateFullyCovered(t *testing.T) {
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 15, 10)

	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)
	err = testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(3))
	require.Nil(t, err)
	err = testEngine.engine.Delegate(context.Background(), "party2", "node1", num.NewUint(3))
	require.Nil(t, err)
	err = testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(2))
	require.Nil(t, err)

	// now undelegate more than pending delegation so that all pending delegation for a node is removed and pending undelegation is added

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(7))
	require.Nil(t, err)
	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(4))
	require.Nil(t, err)

	pendingStateForEpoch := testEngine.engine.pendingState[1]
	// party1 had pending delegation of 2 for node1 so now it should have pending undelegation of 5
	require.Equal(t, num.NewUint(5), pendingStateForEpoch["party1"].totalUndelegation)
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party1"].totalDelegation)
	require.Equal(t, 1, len(pendingStateForEpoch["party1"].nodeToDelegateAmount))
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party1"].nodeToDelegateAmount["node2"])

	// party2 had pending delegation of 2 for node2 so now it should have pending undelegation of 2
	require.Equal(t, num.NewUint(2), pendingStateForEpoch["party2"].totalUndelegation)
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party2"].totalDelegation)
	require.Equal(t, 1, len(pendingStateForEpoch["party2"].nodeToDelegateAmount))
	require.Equal(t, num.NewUint(3), pendingStateForEpoch["party2"].nodeToDelegateAmount["node1"])
}

// preprocess delegation state from last epoch for changes in stake balance - such that there were no changes so no forced undelegation is expected.
func testPreprocessForRewardingNoForcedUndelegationNeeded(t *testing.T) {
	testEngine := getEngine(t)

	setupDefaultDelegationState(testEngine, 12, 10)
	epochStart := time.Now()
	epochEnd := time.Now()
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart] = make(map[string]*num.Uint)
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart]["party1"] = num.NewUint(12)
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart]["party2"] = num.NewUint(10)

	// call preprocess to update the state based on the changes in staking account
	testEngine.engine.preprocessEpochForRewarding(context.Background(), types.Epoch{StartTime: epochStart, EndTime: epochEnd, Seq: 1})

	// the stake account balance for the epoch covers the delegation for both parties so we expect no changes in delegated balances
	require.Equal(t, num.NewUint(8), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(7), testEngine.engine.nodeDelegationState["node2"].totalDelegated)
	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(3), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(10), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(5), testEngine.engine.partyDelegationState["party2"].totalDelegated)
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(3), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node2"])
}

// preprocess delegation state from last epoch for changes in stake balance - such that some tokens have been taken out of the staking account and require undelegation
// from a single available node.
func testPreprocessForRewardingWithForceUndelegateSingleValidator(t *testing.T) {
	testEngine := getEngine(t)
	defaultSimpleDelegationState(testEngine, 12, 10)
	epochStart := time.Now()
	epochEnd := time.Now()
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart] = make(map[string]*num.Uint)
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart]["party1"] = num.NewUint(2)
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart]["party2"] = num.NewUint(0)

	// call preprocess to update the state based on the changes in staking account
	testEngine.engine.preprocessEpochForRewarding(context.Background(), types.Epoch{StartTime: epochStart, EndTime: epochEnd, Seq: 1})

	// both party1 and party2 withdrew tokens from their staking account that require undelegation
	// party1 requires undelegation of 4 tokens
	// party2 requires undelegation of 3 tokens

	// node1 has 2 tokens left delegated to it altogether all by party1
	// node2 has nothing delegated to it
	require.Equal(t, 1, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])

	require.Equal(t, 1, len(testEngine.engine.partyDelegationState))
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
}

// preprocess delegation state from last epoch for changes in stake balance - such that some tokens have been taken out of the staking account and require undelegation
// from a multiple validator with equal proportion available node - with is no remainder.
func testPreprocessForRewardingWithForceUndelegateMultiValidatorNoRemainder(t *testing.T) {
	testEngine := getEngine(t)
	epochStart := time.Now()
	epochEnd := time.Now()
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.topology.nodeToIsValidator["node3"] = true
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart] = make(map[string]*num.Uint)
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart]["party1"] = num.NewUint(15)

	// setup delegation
	// node1 -> 10
	// 		    party1 -> 10
	// node 2 -> 10
	//			party1 -> 10
	// node 3 -> 10
	//			party1 -> 10
	testEngine.engine.nodeDelegationState["node1"] = &validatorDelegation{
		nodeID:         "node1",
		totalDelegated: num.NewUint(10),
		partyToAmount:  make(map[string]*num.Uint),
	}
	testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"] = num.NewUint(10)

	// setup delegation for node2
	testEngine.engine.nodeDelegationState["node2"] = &validatorDelegation{
		nodeID:         "node2",
		totalDelegated: num.NewUint(10),
		partyToAmount:  make(map[string]*num.Uint),
	}
	testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"] = num.NewUint(10)

	// setup delegation for node3
	testEngine.engine.nodeDelegationState["node3"] = &validatorDelegation{
		nodeID:         "node3",
		totalDelegated: num.NewUint(10),
		partyToAmount:  make(map[string]*num.Uint),
	}
	testEngine.engine.nodeDelegationState["node3"].partyToAmount["party1"] = num.NewUint(10)

	testEngine.engine.partyDelegationState["party1"] = &partyDelegation{
		party:          "party1",
		totalDelegated: num.NewUint(30),
		nodeToAmount:   make(map[string]*num.Uint),
	}
	testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"] = num.NewUint(10)
	testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"] = num.NewUint(10)
	testEngine.engine.partyDelegationState["party1"].nodeToAmount["node3"] = num.NewUint(10)

	// call preprocess to update the state based on the changes in staking account
	testEngine.engine.preprocessEpochForRewarding(context.Background(), types.Epoch{StartTime: epochStart, EndTime: epochEnd, Seq: 1})

	// the stake account balance has gone down for party1 to 15 and they have 30 tokens delegated meaning we need to undelegate 15
	// with equal balance in all validators we expect to remove 5 from each

	require.Equal(t, 3, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, num.NewUint(5), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(5), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(5), testEngine.engine.nodeDelegationState["node2"].totalDelegated)
	require.Equal(t, num.NewUint(5), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(5), testEngine.engine.nodeDelegationState["node3"].totalDelegated)
	require.Equal(t, num.NewUint(5), testEngine.engine.nodeDelegationState["node3"].partyToAmount["party1"])
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState))
	require.Equal(t, num.NewUint(15), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(5), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(5), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(5), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node3"])
}

// preprocess delegation state from last epoch for changes in stake balance - such that some tokens have been taken out of the staking account and require undelegation
// from a multiple validator with equal proportion available node - with remainder.
func testPreprocessForRewardingWithForceUndelegateMultiValidatorWithRemainder(t *testing.T) {
	testEngine := getEngine(t)
	epochStart := time.Now()
	epochEnd := time.Now()
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.topology.nodeToIsValidator["node3"] = true
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart] = make(map[string]*num.Uint)
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart]["party1"] = num.NewUint(240)
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart]["party2"] = num.NewUint(50)
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart]["party3"] = num.NewUint(3)

	// setup delegation
	// node1 -> 120
	// 		    party1 -> 100
	// 		    party2 -> 20
	// node 2 -> 100
	//			party1 -> 90
	// 		    party2 -> 10
	// node 3 -> 85
	//			party1 -> 80
	//			party3 -> 5
	testEngine.engine.nodeDelegationState["node1"] = &validatorDelegation{
		nodeID:         "node1",
		totalDelegated: num.NewUint(120),
		partyToAmount:  make(map[string]*num.Uint),
	}
	testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"] = num.NewUint(100)
	testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"] = num.NewUint(20)

	// setup delegation for node2
	testEngine.engine.nodeDelegationState["node2"] = &validatorDelegation{
		nodeID:         "node2",
		totalDelegated: num.NewUint(100),
		partyToAmount:  make(map[string]*num.Uint),
	}
	testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"] = num.NewUint(90)
	testEngine.engine.nodeDelegationState["node2"].partyToAmount["party2"] = num.NewUint(10)

	// setup delegation for node3
	testEngine.engine.nodeDelegationState["node3"] = &validatorDelegation{
		nodeID:         "node3",
		totalDelegated: num.NewUint(85),
		partyToAmount:  make(map[string]*num.Uint),
	}
	testEngine.engine.nodeDelegationState["node3"].partyToAmount["party1"] = num.NewUint(80)
	testEngine.engine.nodeDelegationState["node3"].partyToAmount["party3"] = num.NewUint(5)

	testEngine.engine.partyDelegationState["party1"] = &partyDelegation{
		party:          "party1",
		totalDelegated: num.NewUint(270),
		nodeToAmount:   make(map[string]*num.Uint),
	}
	testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"] = num.NewUint(100)
	testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"] = num.NewUint(90)
	testEngine.engine.partyDelegationState["party1"].nodeToAmount["node3"] = num.NewUint(80)

	testEngine.engine.partyDelegationState["party2"] = &partyDelegation{
		party:          "party2",
		totalDelegated: num.NewUint(30),
		nodeToAmount:   make(map[string]*num.Uint),
	}
	testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"] = num.NewUint(20)
	testEngine.engine.partyDelegationState["party2"].nodeToAmount["node2"] = num.NewUint(10)

	testEngine.engine.partyDelegationState["party3"] = &partyDelegation{
		party:          "party3",
		totalDelegated: num.NewUint(5),
		nodeToAmount:   make(map[string]*num.Uint),
	}
	testEngine.engine.partyDelegationState["party3"].nodeToAmount["node3"] = num.NewUint(5)

	// call preprocess to update the state based on the changes in staking account
	testEngine.engine.preprocessEpochForRewarding(context.Background(), types.Epoch{StartTime: epochStart, EndTime: epochEnd, Seq: 1})

	// the stake account balance for party1 has gone down by 30 so we need to undelegate 30 tokens in total from node1, node2, and node3
	// we do it proportionally to the delegation party1 has in them
	require.Equal(t, num.NewUint(88), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(80), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(72), testEngine.engine.nodeDelegationState["node3"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(240), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(88), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(80), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(72), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node3"])

	// party2 stake account balance hasn't changed so no undelegation needed
	require.Equal(t, num.NewUint(20), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(10), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(30), testEngine.engine.partyDelegationState["party2"].totalDelegated)
	require.Equal(t, num.NewUint(20), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(10), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node2"])

	// party3 stake account balance is down by 2 so we need to undelegate 2 tokens
	require.Equal(t, num.NewUint(3), testEngine.engine.nodeDelegationState["node3"].partyToAmount["party3"])
	require.Equal(t, num.NewUint(3), testEngine.engine.partyDelegationState["party3"].totalDelegated)
	require.Equal(t, num.NewUint(3), testEngine.engine.partyDelegationState["party3"].nodeToAmount["node3"])

	require.Equal(t, 3, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 3, len(testEngine.engine.partyDelegationState))
}

// undelegate an empty slice of parties, no impact on state.
func testPendingUndelegationEmpty(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	testEngine.engine.partyDelegationState["party2"] = &partyDelegation{
		party:          "party2",
		totalDelegated: num.NewUint(5),
		nodeToAmount:   make(map[string]*num.Uint),
	}
	testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"] = num.NewUint(2)
	testEngine.engine.partyDelegationState["party2"].nodeToAmount["node2"] = num.NewUint(3)

	// no pending undelegations
	testEngine.engine.processPendingUndelegations([]string{}, types.Epoch{Seq: 1})

	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, 0, len(pendingStateForEpoch))
	require.Equal(t, num.NewUint(8), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(7), testEngine.engine.nodeDelegationState["node2"].totalDelegated)
	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(3), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(10), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(5), testEngine.engine.partyDelegationState["party2"].totalDelegated)
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(3), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node2"])
}

// undelegate a party with no delegation, no impact on state.
func testPendingUndelegationNothingToUndelegate(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// in this case party3 had delegate state which must have been cleared by the preprocessing step as the party withdrew from the staking account
	// but it still has an undelegation pending for execution - which will have no impact when executed
	testEngine.engine.processPendingUndelegations([]string{"party3"}, types.Epoch{Seq: 1})

	// expect no change in delegation state and clearing of the pending state
	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, 0, len(pendingStateForEpoch))
	require.Equal(t, num.NewUint(8), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(7), testEngine.engine.nodeDelegationState["node2"].totalDelegated)
	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(3), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(10), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(5), testEngine.engine.partyDelegationState["party2"].totalDelegated)
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(3), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node2"])
}

// undelegate an more than the delegated balance of party - the whole balance for the party for the node is cleared.
func testPendingUndelegationGTDelegateddBalance(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// undelegate
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(6))
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(3))

	// update the delegation state to reflect forced undelegation taking place due to party withdrawing from their staking account so that
	// by the time the delegation command is executed the delegated balance is lower than the undelegated amount

	testEngine.engine.nodeDelegationState["node1"] = &validatorDelegation{
		nodeID:         "node1",
		totalDelegated: num.NewUint(7),
		partyToAmount:  make(map[string]*num.Uint),
	}
	testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"] = num.NewUint(5)
	testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"] = num.NewUint(2)

	// setup delegation for node2
	testEngine.engine.nodeDelegationState["node2"] = &validatorDelegation{
		nodeID:         "node2",
		totalDelegated: num.NewUint(6),
		partyToAmount:  make(map[string]*num.Uint),
	}
	testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"] = num.NewUint(4)
	testEngine.engine.nodeDelegationState["node2"].partyToAmount["party2"] = num.NewUint(2)

	testEngine.engine.partyDelegationState["party1"] = &partyDelegation{
		party:          "party1",
		totalDelegated: num.NewUint(9),
		nodeToAmount:   make(map[string]*num.Uint),
	}
	testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"] = num.NewUint(5)
	testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"] = num.NewUint(4)

	testEngine.engine.partyDelegationState["party2"] = &partyDelegation{
		party:          "party2",
		totalDelegated: num.NewUint(4),
		nodeToAmount:   make(map[string]*num.Uint),
	}
	testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"] = num.NewUint(2)
	testEngine.engine.partyDelegationState["party2"].nodeToAmount["node2"] = num.NewUint(2)

	testEngine.engine.processPendingUndelegations([]string{"party1", "party2"}, types.Epoch{Seq: 1})
	require.Equal(t, 1, len(testEngine.engine.nodeDelegationState["node1"].partyToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, 1, len(testEngine.engine.nodeDelegationState["node2"].partyToAmount))
	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node2"].totalDelegated)
	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party2"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party2"].nodeToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"])
}

// undelegate less than the delegated balance of party - the difference between the balances is remained delegated.
func testPendingUndelegationLTDelegateddBalance(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// trying to undelegate more than the node has delegated from the party should just undelegate everything the party has on the node
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(3))
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(1))

	testEngine.engine.processPendingUndelegations([]string{"party1", "party2"}, types.Epoch{Seq: 1})
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState["node1"].partyToAmount))
	require.Equal(t, num.NewUint(5), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(3), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState["node2"].partyToAmount))
	require.Equal(t, num.NewUint(6), testEngine.engine.nodeDelegationState["node2"].totalDelegated)
	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party2"])

	require.Equal(t, num.NewUint(7), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(3), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])

	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party2"].totalDelegated)
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState["party2"].nodeToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node2"])
}

// undelegate the whole balance of a given party from all nodes.
func testPendingUndelegationAllBalanceForParty(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// trying to undelegate more than the node has delegated from the party should just undelegate everything the party has on the node
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(6))
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(3))

	testEngine.engine.processPendingUndelegations([]string{"party1", "party2"}, types.Epoch{Seq: 1})
	require.Equal(t, 1, len(testEngine.engine.nodeDelegationState["node1"].partyToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, 1, len(testEngine.engine.nodeDelegationState["node2"].partyToAmount))
	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node2"].totalDelegated)
	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party2"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party2"].nodeToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"])
}

// undelegate the whole balance of a given node.
func testPendingUndelegationAllBalanceForNode(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// trying to undelegate more than the node has delegated from the party should just undelegate everything the party has on the node
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(6))
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node1", num.NewUint(2))

	testEngine.engine.processPendingUndelegations([]string{"party1", "party2"}, types.Epoch{Seq: 1})
	require.Equal(t, 1, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))
	require.Equal(t, num.NewUint(7), testEngine.engine.nodeDelegationState["node2"].totalDelegated)
	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(3), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(3), testEngine.engine.partyDelegationState["party2"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party2"].nodeToAmount))
	require.Equal(t, num.NewUint(3), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node2"])
}

// no pending delegations to process.
func testPendingDelegationEmpty(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(12)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(7)

	testEngine.engine.processPendingDelegations([]string{}, num.NewUint(10), types.Epoch{Seq: 1})
	require.Equal(t, 0, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 0, len(testEngine.engine.partyDelegationState))
}

// delegation at the time of processing of the pending request has insufficient balance in the staking account.
func testPendingDelegationInsufficientBalance(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(12)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(7)

	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(10))
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(8)
	testEngine.engine.processPendingDelegations([]string{}, num.NewUint(10), types.Epoch{Seq: 1})
	require.Equal(t, 0, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 0, len(testEngine.engine.partyDelegationState))
}

// the validator has all of its allowed allocation filled and it accepts no additional delegation - delegation is ignored.
func testPendingDelegationValidatorAllocationMaxedOut(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// party1 has sufficient balance in their staking account to delegate 2 more
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))

	// expect the state hasn't changed and the delegation request has been ignored
	testEngine.engine.processPendingDelegations([]string{"party1", "party2"}, num.NewUint(8), types.Epoch{Seq: 1})
	require.Equal(t, num.NewUint(8), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(10), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
}

// delegation is adjusted to fit the max delegation per validator parameter.
func testPendingDelegationAmountAdjusted(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// party1 has sufficient balance in their staking account to delegate 2 more
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))

	// the delegation amount has been adjusted to 1 and is added to the state
	testEngine.engine.processPendingDelegations([]string{"party1", "party2"}, num.NewUint(9), types.Epoch{Seq: 1})
	require.Equal(t, num.NewUint(9), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(7), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(11), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(7), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
}

// process pending delegation successfully.
func testPendingDelegationSuccess(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// party1 has sufficient balance in their staking account to delegate 2 more
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))

	// the delegation has been applied on the state
	testEngine.engine.processPendingDelegations([]string{"party1", "party2"}, num.NewUint(10), types.Epoch{Seq: 1})
	require.Equal(t, num.NewUint(10), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(8), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(12), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(8), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
}

// process pending delegations and undelegations.
func testProcessPending(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// party1 has sufficient balance in their staking account to delegate 2 more
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(1))

	testEngine.engine.processPending(context.Background(), types.Epoch{Seq: 1}, num.NewUint(15))
	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, num.NewUint(10), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(8), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(12), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(8), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(6), testEngine.engine.nodeDelegationState["node2"].totalDelegated)
	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party2"].totalDelegated)
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node2"])
	require.Equal(t, 0, len(pendingStateForEpoch))
}

func testGetValidatorsEmpty(t *testing.T) {
	testEngine := getEngine(t)
	validators := testEngine.engine.getValidatorData()
	require.Equal(t, 5, len(validators))

	for i, v := range validators {
		require.Equal(t, strconv.Itoa(i+1), v.NodeID)
		require.Equal(t, num.Zero(), v.SelfStake)
		require.Equal(t, num.Zero(), v.StakeByDelegators)
	}
}

func testGetValidatorsSuccess(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["1"] = true
	testEngine.topology.nodeToIsValidator["2"] = true
	testEngine.topology.nodeToIsValidator["3"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(100)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(100)
	testEngine.stakingAccounts.partyToStake["party3"] = num.NewUint(100)
	testEngine.stakingAccounts.partyToStake["party4"] = num.NewUint(100)
	testEngine.stakingAccounts.partyToStake["party5"] = num.NewUint(100)

	testEngine.engine.Delegate(context.Background(), "party1", "1", num.NewUint(20))
	testEngine.engine.Delegate(context.Background(), "party1", "2", num.NewUint(30))
	testEngine.engine.Delegate(context.Background(), "party1", "3", num.NewUint(40))
	testEngine.engine.Delegate(context.Background(), "party2", "1", num.NewUint(30))
	testEngine.engine.Delegate(context.Background(), "party2", "2", num.NewUint(40))
	testEngine.engine.Delegate(context.Background(), "party2", "3", num.NewUint(20))
	testEngine.engine.Delegate(context.Background(), "party3", "1", num.NewUint(40))
	testEngine.engine.Delegate(context.Background(), "party3", "2", num.NewUint(20))
	testEngine.engine.Delegate(context.Background(), "party3", "3", num.NewUint(30))

	totalTokens := testEngine.engine.calcMaxDelegatableTokens(testEngine.engine.calcTotalDelegatedTokens(1, num.Zero()), num.DecimalFromFloat(3))
	testEngine.engine.processPending(context.Background(), types.Epoch{Seq: 1}, totalTokens)
	validators := testEngine.engine.getValidatorData()
	require.Equal(t, 5, len(validators))
	require.Equal(t, "1", validators[0].NodeID)
	require.Equal(t, num.NewUint(54), validators[0].StakeByDelegators)
	require.Equal(t, 3, len(validators[0].Delegators))
	require.Equal(t, num.NewUint(20), validators[0].Delegators["party1"])
	require.Equal(t, num.NewUint(30), validators[0].Delegators["party2"])
	require.Equal(t, num.NewUint(4), validators[0].Delegators["party3"])

	require.Equal(t, "2", validators[1].NodeID)
	require.Equal(t, num.NewUint(54), validators[1].StakeByDelegators)
	require.Equal(t, 2, len(validators[1].Delegators))
	require.Equal(t, num.NewUint(30), validators[1].Delegators["party1"])
	require.Equal(t, num.NewUint(24), validators[1].Delegators["party2"])

	require.Equal(t, "3", validators[2].NodeID)
	require.Equal(t, 2, len(validators[2].Delegators))
	require.Equal(t, num.NewUint(54), validators[2].StakeByDelegators)
	require.Equal(t, num.NewUint(40), validators[2].Delegators["party1"])
	require.Equal(t, num.NewUint(14), validators[2].Delegators["party2"])
}

func testGetValidatorsSuccessWithSelfDelegation(t *testing.T) {
	testEngine := getEngine(t)
	for i := 1; i < 6; i++ {
		testEngine.topology.nodeToIsValidator[strconv.Itoa(i)] = true
	}

	for i := 1; i < 6; i++ {
		testEngine.stakingAccounts.partyToStake[strconv.Itoa(i)] = num.NewUint(10000)
		err := testEngine.engine.Delegate(context.Background(), strconv.Itoa(i), strconv.Itoa(i), num.NewUint(200))
		require.Nil(t, err)
		for j := 1; j < 6; j++ {
			if i != j {
				err = testEngine.engine.Delegate(context.Background(), strconv.Itoa(i), strconv.Itoa(j), num.NewUint(100))
				require.Nil(t, err)
			}
		}
	}

	for i := 1; i < 11; i++ {
		testEngine.stakingAccounts.partyToStake["party"+strconv.Itoa(i)] = num.NewUint(100)
		for j := 1; j < 6; j++ {
			testEngine.engine.Delegate(context.Background(), "party"+strconv.Itoa(i), strconv.Itoa(j), num.NewUint(2))
		}
	}

	totalTokens := testEngine.engine.calcTotalDelegatedTokens(1, num.Zero())

	testEngine.engine.processPending(context.Background(), types.Epoch{Seq: 1}, totalTokens)
	validators := testEngine.engine.getValidatorData()

	require.Equal(t, 5, len(validators))
	for i := 1; i < 6; i++ {
		require.Equal(t, strconv.Itoa(i), validators[i-1].NodeID)
		// 100 from each other validator (400) + 2 from each party (20)
		require.Equal(t, num.NewUint(420), validators[i-1].StakeByDelegators)
		require.Equal(t, num.NewUint(200), validators[i-1].SelfStake)
		// 10 parties + 4 other validators
		require.Equal(t, 14, len(validators[i-1].Delegators))

		for j := 1; j < 11; j++ {
			require.Equal(t, num.NewUint(2), validators[i-1].Delegators["party"+strconv.Itoa(j)])
		}
	}
}

// try to undelegate more than delegated.
func testUndelegateNowIncorrectAmount(t *testing.T) {
	testEngine := getEngine(t)

	err := testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.NewUint(10))
	require.EqualError(t, err, ErrIncorrectTokenAmountForUndelegation.Error())

	// setup delegation state
	setupDefaultDelegationState(testEngine, 12, 7)

	// party1/node1 has 6 delegated, try to undelegate 8
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.NewUint(8))
	require.EqualError(t, err, ErrIncorrectTokenAmountForUndelegation.Error())

	// add pending delegation of 2
	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)

	// party1 have 8 delegated in total to node1 (6 committed 2 pending) - try to undelegate 10 should error
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.NewUint(10))
	require.EqualError(t, err, ErrIncorrectTokenAmountForUndelegation.Error())

	// show that undelegating 8 doesn't error
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.NewUint(8))
	require.Nil(t, err)
}

// undelegate all now, there are no committed delegations for the node, only pending and they are all cleared.
func testUndelegateNowAllWithPendingOnly(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(30)
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(10))
	testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(10))

	err := testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.Zero())
	require.Nil(t, err)
	pendingStateForEpoch := testEngine.engine.pendingState[1]

	require.Equal(t, 1, len(pendingStateForEpoch["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party1"].nodeToUndelegateAmount))
	require.Equal(t, num.NewUint(10), pendingStateForEpoch["party1"].totalDelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party1"].totalUndelegation)
	require.Equal(t, num.NewUint(10), pendingStateForEpoch["party1"].nodeToDelegateAmount["node2"])
}

// there's no pending delegation, remove all committed delegation.
func testUndelegateNowAllWithCommittedOnly(t *testing.T) {
	testEngine := getEngine(t)
	// setup delegation state
	setupDefaultDelegationState(testEngine, 12, 7)

	// undelegate now all for party1 node1
	err := testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.Zero())
	require.Nil(t, err)

	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, 0, len(pendingStateForEpoch))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])

	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.nodeDelegationState["node1"].partyToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])

	// undelegate now all for party1 node2
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node2", num.Zero())
	require.Nil(t, err)
	require.Equal(t, 0, len(pendingStateForEpoch))
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState))
}

// there's both committed and pending delegation, take all from both.
func testUndelegateNowAll(t *testing.T) {
	testEngine := getEngine(t)
	// setup delegation state
	setupDefaultDelegationState(testEngine, 12, 7)

	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)

	// undelegate now all for party1 node1 both committed and pending state should update
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.Zero())
	require.Nil(t, err)

	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, 0, len(pendingStateForEpoch))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])

	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.nodeDelegationState["node1"].partyToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])

	// undelegate now all for party1 node2
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node2", num.Zero())
	require.Nil(t, err)
	require.Equal(t, 0, len(pendingStateForEpoch))
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState))
}

func testUndelegateNowWithPendingOnly(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(30)
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(10))
	testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(10))

	err := testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.NewUint(5))
	require.Nil(t, err)

	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, 2, len(pendingStateForEpoch["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(pendingStateForEpoch["party1"].nodeToUndelegateAmount))
	require.Equal(t, num.NewUint(15), pendingStateForEpoch["party1"].totalDelegation)
	require.Equal(t, num.Zero(), pendingStateForEpoch["party1"].totalUndelegation)
	require.Equal(t, num.NewUint(5), pendingStateForEpoch["party1"].nodeToDelegateAmount["node1"])
	require.Equal(t, num.NewUint(10), pendingStateForEpoch["party1"].nodeToDelegateAmount["node2"])
}

func testUndelegateNowWithCommittedOnly(t *testing.T) {
	testEngine := getEngine(t)
	// setup delegation state
	setupDefaultDelegationState(testEngine, 12, 7)

	// undelegate now all for party1 node1
	err := testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.NewUint(4))
	require.Nil(t, err)

	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, 0, len(pendingStateForEpoch))
	require.Equal(t, num.NewUint(6), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])

	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState["node1"].partyToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
}

// undelegate now amount is fully covered in pending delegation, the committed state is unchanged.
func testUndelegateNowPendingCovers(t *testing.T) {
	testEngine := getEngine(t)
	// setup delegation state
	setupDefaultDelegationState(testEngine, 13, 7)

	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(3))
	require.Nil(t, err)

	// undelegate now all for party1 node1
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.NewUint(3))
	require.Nil(t, err)

	// pending state should have cleared
	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, 0, len(pendingStateForEpoch))

	// committed state should have stayed the same
	require.Equal(t, num.NewUint(10), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(8), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
}

// undelegate now takes all pending and some of the committed delegation.
func testUndelegateNowCommittedCovers(t *testing.T) {
	testEngine := getEngine(t)
	// setup delegation state
	setupDefaultDelegationState(testEngine, 13, 7)

	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(3))
	require.Nil(t, err)

	// undelegate now for party1 node1
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.NewUint(7))
	require.Nil(t, err)

	// pending state cleared
	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, 0, len(pendingStateForEpoch))

	// committed state lost 4 delegated tokens for party1 node1
	require.Equal(t, num.NewUint(6), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])

	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState["node1"].partyToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
}

// undelegate now with an amount equals to the total delegated (pending + committed).
func testUndelegateNowAllCleared(t *testing.T) {
	testEngine := getEngine(t)
	// setup delegation state
	setupDefaultDelegationState(testEngine, 13, 7)

	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(3))
	require.Nil(t, err)

	// undelegate now for party1 node1
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.NewUint(9))
	require.Nil(t, err)

	// pending state cleared
	pendingStateForEpoch := testEngine.engine.pendingState[1]
	require.Equal(t, 0, len(pendingStateForEpoch))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])

	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.nodeDelegationState["node1"].partyToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])

	// undelegate now all for party1 node2
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node2", num.NewUint(4))
	require.Nil(t, err)
	require.Equal(t, 0, len(pendingStateForEpoch))
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
}

func testCalculateTotalDelegatedTokens(t *testing.T) {
	testEngine := getEngine(t)

	// setup delegation state
	setupDefaultDelegationState(testEngine, 13, 7)
	require.Equal(t, num.NewUint(15), testEngine.engine.calcTotalDelegatedTokens(1, num.Zero()))

	err := testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)
	require.Equal(t, num.NewUint(13), testEngine.engine.calcTotalDelegatedTokens(1, num.Zero()))

	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(5))
	require.Nil(t, err)
	require.Equal(t, num.NewUint(18), testEngine.engine.calcTotalDelegatedTokens(1, num.Zero()))

	// with tokens available for auto delegation
	require.Equal(t, num.NewUint(28), testEngine.engine.calcTotalDelegatedTokens(1, num.NewUint(10)))
}

func testMaxStakePerValidator(t *testing.T) {
	testEngine := getEngine(t)
	// 1/a = 1/5 = 0.2
	// max per validator = 0.2 * 1000 = 200
	require.Equal(t, num.NewUint(200), testEngine.engine.calcMaxDelegatableTokens(num.NewUint(1000), num.DecimalFromFloat(3)))

	// 1/a = 11/1.1 = 0.1
	// max per validator = 0.1 * 1000 = 100
	require.Equal(t, num.NewUint(100), testEngine.engine.calcMaxDelegatableTokens(num.NewUint(1000), num.DecimalFromFloat(11)))
}

func testCheckPartyEnteringAutoDelegation(t *testing.T) {
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 10, 5)

	testEngine.engine.ProcessEpochDelegations(context.Background(), types.Epoch{Seq: 1})
	require.Contains(t, testEngine.engine.autoDelegationMode, "party1")
	require.Contains(t, testEngine.engine.autoDelegationMode, "party2")
}

func testCheckPartyExitingAutoDelegationThroughUndelegateEOE(t *testing.T) {
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 10, 5)
	testEngine.engine.ProcessEpochDelegations(context.Background(), types.Epoch{Seq: 1})
	require.Contains(t, testEngine.engine.autoDelegationMode, "party1")
	require.Contains(t, testEngine.engine.autoDelegationMode, "party2")

	testEngine.engine.onEpochEvent(context.Background(), types.Epoch{Seq: 2})
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(1))
	testEngine.engine.ProcessEpochDelegations(context.Background(), types.Epoch{Seq: 2})

	require.NotContains(t, testEngine.engine.autoDelegationMode, "party1")
	require.Contains(t, testEngine.engine.autoDelegationMode, "party2")
}

func testCheckPartyExitingAutoDelegationThroughUndelegateNow(t *testing.T) {
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 10, 5)
	testEngine.engine.ProcessEpochDelegations(context.Background(), types.Epoch{Seq: 1})
	require.Contains(t, testEngine.engine.autoDelegationMode, "party1")
	require.Contains(t, testEngine.engine.autoDelegationMode, "party2")

	testEngine.engine.onEpochEvent(context.Background(), types.Epoch{Seq: 2})
	testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.NewUint(1))
	require.NotContains(t, testEngine.engine.autoDelegationMode, "party1")
	require.Contains(t, testEngine.engine.autoDelegationMode, "party2")

	testEngine.engine.ProcessEpochDelegations(context.Background(), types.Epoch{Seq: 2})
	require.NotContains(t, testEngine.engine.autoDelegationMode, "party1")
	require.Contains(t, testEngine.engine.autoDelegationMode, "party2")
}

func TestPartyInAutoDelegateModeWithManualInterention(t *testing.T) {
	testEngine := getEngine(t)

	// epoch 0 - setup delegations
	testEngine.engine.onEpochEvent(context.Background(), types.Epoch{Seq: 0})
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.topology.nodeToIsValidator["node3"] = true
	testEngine.topology.nodeToIsValidator["node4"] = true
	testEngine.topology.nodeToIsValidator["node5"] = true
	testEngine.topology.nodeToIsValidator["node6"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(1000)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(1000)
	testEngine.stakingAccounts.partyToStake["node1"] = num.NewUint(10000)
	testEngine.stakingAccounts.partyToStake["node2"] = num.NewUint(10000)
	testEngine.stakingAccounts.partyToStake["node3"] = num.NewUint(10000)
	testEngine.stakingAccounts.partyToStake["node4"] = num.NewUint(10000)
	testEngine.stakingAccounts.partyToStake["node5"] = num.NewUint(10000)
	testEngine.stakingAccounts.partyToStake["node6"] = num.NewUint(10000)

	testEngine.engine.Delegate(context.Background(), "node1", "node1", num.NewUint(10000))
	testEngine.engine.Delegate(context.Background(), "node2", "node2", num.NewUint(10000))
	testEngine.engine.Delegate(context.Background(), "node3", "node3", num.NewUint(10000))
	testEngine.engine.Delegate(context.Background(), "node4", "node4", num.NewUint(10000))
	testEngine.engine.Delegate(context.Background(), "node5", "node5", num.NewUint(10000))
	testEngine.engine.Delegate(context.Background(), "node6", "node6", num.NewUint(10000))

	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(400))
	testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(600))
	testEngine.engine.Delegate(context.Background(), "party2", "node1", num.NewUint(800))
	testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(150))

	testEngine.engine.ProcessEpochDelegations(context.Background(), types.Epoch{Seq: 0})

	require.Contains(t, testEngine.engine.autoDelegationMode, "party1")
	require.Contains(t, testEngine.engine.autoDelegationMode, "party2")

	// // start epoch 1
	testEngine.engine.onEpochEvent(context.Background(), types.Epoch{Seq: 1})
	// increase association of party1 and party2
	testEngine.stakingAccounts.partyToStake["party1"].AddSum(num.NewUint(1000))
	testEngine.stakingAccounts.partyToStake["party2"].AddSum(num.NewUint(1500))
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(100))

	testEngine.engine.ProcessEpochDelegations(context.Background(), types.Epoch{Seq: 1})
	require.Contains(t, testEngine.engine.autoDelegationMode, "party1")
	require.Contains(t, testEngine.engine.autoDelegationMode, "party2")

	// party1 has delegated during the epoch so they don't qualify for auto delegation. party1 had 6 and 4 respectively to node1 and node2 and they manually
	// delegate 5 more to node 1
	require.Equal(t, num.NewUint(1100), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(500), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(600), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])

	// party2 has not delegated during the epoch so their newly available stake gets auto delegated
	// party2 had a delegation of 800 to node1 and 150 to node 2,
	// the same distribution is applied on the additional 1550 tokens and now they should have additional 1305 and 244 to node 1 and node 2 respectively
	require.Equal(t, num.NewUint(2499), testEngine.engine.partyDelegationState["party2"].totalDelegated)
	require.Equal(t, num.NewUint(2105), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(394), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node2"])

	// node1 had 10000 from itself, and 400 and 800 from party1 and party2 respectively. This epoch party1 has manually added 100 and party2 has added 1305 = 12605
	require.Equal(t, num.NewUint(12605), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(500), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2105), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])

	// node2 had 10000 from itself, and 600 and 150 from party1 and party2 respectively. This epoch party2 has added 244 = 10994
	require.Equal(t, num.NewUint(10994), testEngine.engine.nodeDelegationState["node2"].totalDelegated)
	require.Equal(t, num.NewUint(600), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(394), testEngine.engine.nodeDelegationState["node2"].partyToAmount["party2"])
}

func testSortActive(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for k := 0; k < 100; k++ {
		testEngine := getEngine(t)
		for j := 0; j < 5; j++ {
			active := []*types.DelegationEntry{}
			var epochSeq uint64 = 1

			active = append(active, &types.DelegationEntry{
				Party:    "party1",
				Node:     "node1",
				Amount:   num.NewUint(100),
				EpochSeq: epochSeq,
			})
			active = append(active, &types.DelegationEntry{
				Party:    "party1",
				Node:     "node2",
				Amount:   num.NewUint(200),
				EpochSeq: epochSeq,
			})
			active = append(active, &types.DelegationEntry{
				Party:    "party2",
				Node:     "node1",
				Amount:   num.NewUint(300),
				EpochSeq: epochSeq,
			})
			active = append(active, &types.DelegationEntry{
				Party:    "party2",
				Node:     "node2",
				Amount:   num.NewUint(400),
				EpochSeq: epochSeq,
			})

			rand.Shuffle(len(active), func(i, j int) { active[i], active[j] = active[j], active[i] })

			testEngine.engine.sortActive(active)
			require.Equal(t, "party1", active[0].Party)
			require.Equal(t, "node1", active[0].Node)
			require.Equal(t, "party1", active[1].Party)
			require.Equal(t, "node2", active[1].Node)
			require.Equal(t, "party2", active[2].Party)
			require.Equal(t, "node1", active[2].Node)
			require.Equal(t, "party2", active[3].Party)
			require.Equal(t, "node2", active[3].Node)
		}
	}
}

func testSortPending(t *testing.T) {
	for k := 0; k < 100; k++ {
		testEngine := getEngine(t)
		for j := 0; j < 5; j++ {
			pending := []*types.DelegationEntry{}
			var epochSeq uint64 = 1

			pending = append(pending, &types.DelegationEntry{
				Party:      "party1",
				Node:       "node1",
				Amount:     num.NewUint(100),
				EpochSeq:   epochSeq,
				Undelegate: false,
			})
			pending = append(pending, &types.DelegationEntry{
				Party:      "party1",
				Node:       "node2",
				Amount:     num.NewUint(200),
				EpochSeq:   epochSeq,
				Undelegate: false,
			})
			pending = append(pending, &types.DelegationEntry{
				Party:      "party2",
				Node:       "node1",
				Amount:     num.NewUint(300),
				EpochSeq:   epochSeq,
				Undelegate: false,
			})
			pending = append(pending, &types.DelegationEntry{
				Party:      "party2",
				Node:       "node2",
				Amount:     num.NewUint(400),
				EpochSeq:   epochSeq,
				Undelegate: false,
			})
			pending = append(pending, &types.DelegationEntry{
				Party:      "party1",
				Node:       "node1",
				Amount:     num.NewUint(50),
				EpochSeq:   epochSeq + 1,
				Undelegate: true,
			})
			pending = append(pending, &types.DelegationEntry{
				Party:      "party1",
				Node:       "node2",
				Amount:     num.NewUint(150),
				EpochSeq:   epochSeq + 1,
				Undelegate: false,
			})
			pending = append(pending, &types.DelegationEntry{
				Party:      "party2",
				Node:       "node1",
				Amount:     num.NewUint(250),
				EpochSeq:   epochSeq + 1,
				Undelegate: true,
			})
			pending = append(pending, &types.DelegationEntry{
				Party:      "party2",
				Node:       "node2",
				Amount:     num.NewUint(350),
				EpochSeq:   epochSeq + 1,
				Undelegate: false,
			})

			rand.Shuffle(len(pending), func(i, j int) { pending[i], pending[j] = pending[j], pending[i] })

			testEngine.engine.sortPending(pending)

			// sorting by party then node then epoch
			require.Equal(t, "party1", pending[0].Party)
			require.Equal(t, "node1", pending[0].Node)
			require.Equal(t, uint64(1), pending[0].EpochSeq)
			require.False(t, pending[0].Undelegate)
			require.Equal(t, num.NewUint(100), pending[0].Amount)

			require.Equal(t, "party1", pending[1].Party)
			require.Equal(t, "node1", pending[1].Node)
			require.Equal(t, uint64(2), pending[1].EpochSeq)
			require.True(t, pending[1].Undelegate)
			require.Equal(t, num.NewUint(50), pending[1].Amount)

			require.Equal(t, "party1", pending[2].Party)
			require.Equal(t, "node2", pending[2].Node)
			require.Equal(t, uint64(1), pending[2].EpochSeq)
			require.False(t, pending[2].Undelegate)
			require.Equal(t, num.NewUint(200), pending[2].Amount)

			require.Equal(t, "party1", pending[3].Party)
			require.Equal(t, "node2", pending[3].Node)
			require.Equal(t, uint64(2), pending[3].EpochSeq)
			require.False(t, pending[3].Undelegate)
			require.Equal(t, num.NewUint(150), pending[3].Amount)

			require.Equal(t, "party2", pending[4].Party)
			require.Equal(t, "node1", pending[4].Node)
			require.Equal(t, uint64(1), pending[4].EpochSeq)
			require.False(t, pending[4].Undelegate)
			require.Equal(t, num.NewUint(300), pending[4].Amount)

			require.Equal(t, "party2", pending[5].Party)
			require.Equal(t, "node1", pending[5].Node)
			require.Equal(t, uint64(2), pending[5].EpochSeq)
			require.True(t, pending[5].Undelegate)
			require.Equal(t, num.NewUint(250), pending[5].Amount)

			require.Equal(t, "party2", pending[6].Party)
			require.Equal(t, "node2", pending[6].Node)
			require.Equal(t, uint64(1), pending[6].EpochSeq)
			require.False(t, pending[6].Undelegate)
			require.Equal(t, num.NewUint(400), pending[6].Amount)

			require.Equal(t, "party2", pending[7].Party)
			require.Equal(t, "node2", pending[7].Node)
			require.Equal(t, uint64(2), pending[7].EpochSeq)
			require.False(t, pending[7].Undelegate)
			require.Equal(t, num.NewUint(350), pending[7].Amount)
		}
	}
}

func testCheckpointRoundtripNoPending(t *testing.T) {
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		testEngine := getEngine(t)
		testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
		setupDefaultDelegationState(testEngine, 12, 7)

		checkpoint, err := testEngine.engine.Checkpoint()
		require.Nil(t, err)

		testEngine.engine.Load(ctx, checkpoint)
		checkpoint2, err := testEngine.engine.Checkpoint()
		require.Nil(t, err)
		require.True(t, bytes.Equal(checkpoint, checkpoint2))
	}
}

func testCheckpointRoundtripOnlyPending(t *testing.T) {
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		testEngine := getEngine(t)
		testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(2)

		testEngine.topology.nodeToIsValidator["node1"] = true
		testEngine.topology.nodeToIsValidator["node2"] = true
		testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(100)
		testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(200)

		engine := testEngine.engine
		err := engine.Delegate(context.Background(), "party1", "node1", num.NewUint(60))
		require.Nil(t, err)
		err = engine.Delegate(context.Background(), "party1", "node2", num.NewUint(40))
		require.Nil(t, err)

		err = engine.Delegate(context.Background(), "party2", "node1", num.NewUint(70))
		require.Nil(t, err)
		err = engine.Delegate(context.Background(), "party2", "node2", num.NewUint(130))
		require.Nil(t, err)

		checkpoint, err := testEngine.engine.Checkpoint()
		require.Nil(t, err)

		testEngine.engine.Load(ctx, checkpoint)
		checkpoint2, err := testEngine.engine.Checkpoint()
		require.Nil(t, err)
		require.True(t, bytes.Equal(checkpoint, checkpoint2))
	}
}

func getEngine(t *testing.T) *testEngine {
	t.Helper()
	conf := NewDefaultConfig()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	logger := logging.NewTestLogger()
	stakingAccounts := newTestStakingAccount()
	topology := newTestTopology()

	engine := New(logger, conf, broker, topology, stakingAccounts, &TestEpochEngine{})
	engine.onEpochEvent(context.Background(), types.Epoch{Seq: 1})
	engine.OnMinAmountChanged(context.Background(), num.NewDecimalFromFloat(2))
	engine.OnCompLevelChanged(context.Background(), 1.1)
	engine.OnMinValidatorsChanged(context.Background(), 5)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	return &testEngine{
		engine:          engine,
		ctrl:            ctrl,
		broker:          broker,
		stakingAccounts: stakingAccounts,
		topology:        topology,
	}
}

type TestEpochEngine struct{}

func (t *TestEpochEngine) NotifyOnEpoch(f func(context.Context, types.Epoch)) {}

type TestStakingAccount struct {
	partyToStake         map[string]*num.Uint
	partyToStakeForEpoch map[time.Time]map[string]*num.Uint
}

func newTestStakingAccount() *TestStakingAccount {
	return &TestStakingAccount{
		partyToStake:         make(map[string]*num.Uint),
		partyToStakeForEpoch: make(map[time.Time]map[string]*num.Uint),
	}
}

func (t *TestStakingAccount) GetAvailableBalance(party string) (*num.Uint, error) {
	ret, ok := t.partyToStake[party]
	if !ok {
		return nil, fmt.Errorf("account not found")
	}
	return ret, nil
}

func (t *TestStakingAccount) GetAvailableBalanceInRange(party string, from, to time.Time) (*num.Uint, error) {
	ret, ok := t.partyToStakeForEpoch[from]
	if !ok {
		return nil, fmt.Errorf("account not found")
	}

	p, ok := ret[party]
	if !ok {
		return nil, fmt.Errorf("account not found")
	}
	return p, nil
}

type TestTopology struct {
	nodeToIsValidator map[string]bool
}

func newTestTopology() *TestTopology {
	return &TestTopology{
		nodeToIsValidator: make(map[string]bool),
	}
}

func (tt *TestTopology) IsValidatorNode(nodeID string) bool {
	v, ok := tt.nodeToIsValidator[nodeID]
	return ok && v
}

func (tt *TestTopology) AllNodeIDs() []string {
	return []string{"1", "2", "3", "4", "5"}
}
