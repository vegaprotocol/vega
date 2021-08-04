package delegation

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/broker/mocks"
	gmock "code.vegaprotocol.io/vega/governance/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEngine struct {
	engine          *Engine
	ctrl            *gomock.Controller
	broker          *mocks.MockBroker
	stakingAccounts *TestStakingAccount
	topology        *TestTopology
	netp            *gmock.MockNetParams
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
	t.Run("Undelegation more than the delegated balance succeeds", testUndelegateInvalidAmount)
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
	t.Run("Undelegate an amount  with both active and pending delegation - insufficient cover in pending succeeds", testUndelegateNowCommittedCovers)
	t.Run("Undelegate an amount  with both active and pending delegation - all delegation removed", testUndelegateNowAllCleared)

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
}

// pass an invalid node id
// expect an ErrInvalidNodeID
func testDelegateInvalidNode(t *testing.T) {
	testEngine := getEngine(t)
	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(10))
	assert.EqualError(t, err, ErrInvalidNodeID.Error())
}

// pass a party with no staking account
// expect ErrPartyHasNoStakingAccount
func testDelegateNoStakingAccount(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(10))
	assert.EqualError(t, err, ErrPartyHasNoStakingAccount.Error())
}

// try to delegate less than the network param for min delegation amount
// expect ErrAmountLTMinAmountForDelegation
func testDelegateLessThanMinDelegationAmount(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(5)
	testEngine.netp.EXPECT().Get("validators.delegation.minAmount").Return("2", nil)
	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(1))
	assert.EqualError(t, err, ErrAmountLTMinAmountForDelegation.Error())
}

// party has insufficient balance in their staking account to delegate - they have nothing pending and no committed delegation
// expect ErrInsufficientBalanceForDelegation
func testDelegateInsufficientBalanceNoPendingNoCommitted(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(5)
	testEngine.netp.EXPECT().Get("validators.delegation.minAmount").Return("2", nil)
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

// party has pending delegations and is trying to exceed their stake account balance delegation, i.e. the balance of their pending delegation + requested delegation exceeds stake account balance
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

// party has committed delegations and is trying to exceed their stake account balance delegations i.e. the balance of their pending delegation + requested delegation exceeds stake account balance
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

// party has both committed delegations and pending delegations and an additional delegation will exceed the amount of available tokens for delegations in their staking account
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

// party has both committed delegations and pending undelegations
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

// balance available for delegation is greater than delegation amount, delegation succeeds
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
	//party1 delegated 10 in total, 7 to node1 and 3 to node2
	//party2 delegated 6 in total, all to node1
	// verify the state
	require.Equal(t, num.NewUint(10), testEngine.engine.pendingState["party1"].totalDelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party1"].totalUndelegation)
	require.Equal(t, num.NewUint(6), testEngine.engine.pendingState["party2"].totalDelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party2"].totalUndelegation)
	require.Equal(t, num.NewUint(7), testEngine.engine.pendingState["party1"].nodeToDelegateAmount["node1"])
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party1"].nodeToDelegateAmount["node2"])
	require.Equal(t, num.NewUint(6), testEngine.engine.pendingState["party2"].nodeToDelegateAmount["node1"])
	require.Equal(t, 0, len(testEngine.engine.pendingState["party1"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party2"].nodeToUndelegateAmount))
	require.Equal(t, 2, len(testEngine.engine.pendingState["party1"].nodeToDelegateAmount))
	require.Equal(t, 1, len(testEngine.engine.pendingState["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(testEngine.engine.pendingState))
	require.Equal(t, 0, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 0, len(testEngine.engine.partyDelegationState))
}

// test delegation when there is already pending undelegation but the deletation is more than fully covering the pending undelegation
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
	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party1"].totalUndelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party1"].totalDelegation)
	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party2"].totalUndelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party2"].totalDelegation)
	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party1"].nodeToUndelegateAmount["node1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party2"].nodeToUndelegateAmount["node2"])
	require.Equal(t, 1, len(testEngine.engine.pendingState["party1"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(testEngine.engine.pendingState["party2"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(testEngine.engine.pendingState))
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
	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party1"].totalDelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party1"].totalUndelegation)
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party2"].totalDelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party2"].totalUndelegation)
	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party1"].nodeToDelegateAmount["node1"])
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party2"].nodeToDelegateAmount["node2"])
	require.Equal(t, 0, len(testEngine.engine.pendingState["party1"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party2"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(testEngine.engine.pendingState["party1"].nodeToDelegateAmount))
	require.Equal(t, 1, len(testEngine.engine.pendingState["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(testEngine.engine.pendingState))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))
}

// test delegation when there is already pending undelegation and the delegation is covering part of the undelegation
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
	require.Equal(t, num.NewUint(4), testEngine.engine.pendingState["party1"].totalUndelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party1"].totalDelegation)
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party2"].totalUndelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party2"].totalDelegation)
	require.Equal(t, num.NewUint(4), testEngine.engine.pendingState["party1"].nodeToUndelegateAmount["node1"])
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party2"].nodeToUndelegateAmount["node2"])
	require.Equal(t, 1, len(testEngine.engine.pendingState["party1"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(testEngine.engine.pendingState["party2"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(testEngine.engine.pendingState))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))

	// delegte 3 (< the pending undelegation of 4) from party 1 to node 1
	err = testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(3))
	require.Nil(t, err)

	// delegate 2 (< the pending undelegation of 3) from party 2 to node2
	err = testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(2))
	require.Nil(t, err)

	// verify the state
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party1"].totalDelegation)
	require.Equal(t, num.NewUint(1), testEngine.engine.pendingState["party1"].totalUndelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party2"].totalDelegation)
	require.Equal(t, num.NewUint(1), testEngine.engine.pendingState["party2"].totalUndelegation)
	require.Equal(t, num.NewUint(1), testEngine.engine.pendingState["party1"].nodeToUndelegateAmount["node1"])
	require.Equal(t, num.NewUint(1), testEngine.engine.pendingState["party2"].nodeToUndelegateAmount["node2"])
	require.Equal(t, 0, len(testEngine.engine.pendingState["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party2"].nodeToDelegateAmount))
	require.Equal(t, 1, len(testEngine.engine.pendingState["party1"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(testEngine.engine.pendingState["party2"].nodeToUndelegateAmount))
	require.Equal(t, 2, len(testEngine.engine.pendingState))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))
}

// test delegation when there is already pending undelegation and the delegation is countering exactly the undelegation
func testDelegateSuccessWithPreviousPendingUndelegateExactlyCovered(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	defaultSimpleDelegationState(testEngine, 12, 7)

	// setup pending undelegation
	err := testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(4))
	require.Nil(t, err)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(3))
	require.Nil(t, err)

	// show that the state before delegation matches expectation (i.e. that we have 2 for undelegation from party1 and party2 to node1 and node2 respectively)
	require.Equal(t, num.NewUint(4), testEngine.engine.pendingState["party1"].totalUndelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party1"].totalDelegation)
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party2"].totalUndelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party2"].totalDelegation)
	require.Equal(t, num.NewUint(4), testEngine.engine.pendingState["party1"].nodeToUndelegateAmount["node1"])
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party2"].nodeToUndelegateAmount["node2"])
	require.Equal(t, 1, len(testEngine.engine.pendingState["party1"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(testEngine.engine.pendingState["party2"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(testEngine.engine.pendingState))
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
	require.Equal(t, 0, len(testEngine.engine.pendingState))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))
}

/// undelegate
func testUndelegateInvalidNode(t *testing.T) {
	testEngine := getEngine(t)
	err := testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(10))
	assert.EqualError(t, err, ErrInvalidNodeID.Error())
}

// trying to undelegate more than the delegated amount when no undelegation or more than the delegated - undelegated if there are some
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

// trying to undelegate then incresae the undelegated amount until all is undelegated
func testUndelegateSuccessNoPreviousPending(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	defaultSimpleDelegationState(testEngine, 12, 7)

	// setup pending undelegation
	err := testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(2))
	require.Nil(t, err)

	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party1"].totalUndelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party1"].totalDelegation)
	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party2"].totalUndelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party2"].totalDelegation)
	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party1"].nodeToUndelegateAmount["node1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party2"].nodeToUndelegateAmount["node2"])
	require.Equal(t, 1, len(testEngine.engine.pendingState["party1"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(testEngine.engine.pendingState["party2"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(testEngine.engine.pendingState))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))

	// undelegate everything now
	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(4))
	require.Nil(t, err)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(1))
	require.Nil(t, err)

	// check that the state has updated correctly
	require.Equal(t, num.NewUint(6), testEngine.engine.pendingState["party1"].totalUndelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party1"].totalDelegation)
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party2"].totalUndelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party2"].totalDelegation)
	require.Equal(t, num.NewUint(6), testEngine.engine.pendingState["party1"].nodeToUndelegateAmount["node1"])
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party2"].nodeToUndelegateAmount["node2"])
	require.Equal(t, 1, len(testEngine.engine.pendingState["party1"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(testEngine.engine.pendingState["party2"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(testEngine.engine.pendingState))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState))

	// trying to further undelegate will get an error
	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(1))
	assert.Error(t, err, ErrIncorrectTokenAmountForUndelegation)

	err = testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(1))
	assert.Error(t, err, ErrIncorrectTokenAmountForUndelegation)
}

// delegate an amount that leave some delegation for the party
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

	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party1"].totalUndelegation)
	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party1"].totalDelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party2"].totalUndelegation)
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party2"].totalDelegation)
	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party1"].nodeToDelegateAmount["node2"])
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party2"].nodeToDelegateAmount["node2"])
	require.Equal(t, 0, len(testEngine.engine.pendingState["party1"].nodeToUndelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party2"].nodeToUndelegateAmount))
	require.Equal(t, 1, len(testEngine.engine.pendingState["party1"].nodeToDelegateAmount))
	require.Equal(t, 1, len(testEngine.engine.pendingState["party2"].nodeToDelegateAmount))
	require.Equal(t, 2, len(testEngine.engine.pendingState))
	require.Equal(t, 0, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 0, len(testEngine.engine.partyDelegationState))

}

// undelegate incrementally to get all pending delegates countered
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

	require.Equal(t, 0, len(testEngine.engine.pendingState))

}

// undelegate such that delegation for some party and node goes from delegate to undelegate
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

	// party1 had pending delegation of 2 for node1 so now it should have pending undelegation of 5
	require.Equal(t, num.NewUint(5), testEngine.engine.pendingState["party1"].totalUndelegation)
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party1"].totalDelegation)
	require.Equal(t, 1, len(testEngine.engine.pendingState["party1"].nodeToDelegateAmount))
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party1"].nodeToDelegateAmount["node2"])

	// party2 had pending delegation of 2 for node2 so now it should have pending undelegation of 2
	require.Equal(t, num.NewUint(2), testEngine.engine.pendingState["party2"].totalUndelegation)
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party2"].totalDelegation)
	require.Equal(t, 1, len(testEngine.engine.pendingState["party2"].nodeToDelegateAmount))
	require.Equal(t, num.NewUint(3), testEngine.engine.pendingState["party2"].nodeToDelegateAmount["node1"])
}

// preprocess delegation state from last epoch for changes in stake balance - such that there were no changes so no forced undelegation is expected
func testPreprocessForRewardingNoForcedUndelegationNeeded(t *testing.T) {
	testEngine := getEngine(t)

	setupDefaultDelegationState(testEngine, 12, 10)
	epochStart := time.Now()
	epochEnd := time.Now()
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart] = make(map[string]*num.Uint)
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart]["party1"] = num.NewUint(12)
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart]["party2"] = num.NewUint(10)

	// call preprocess to update the state based on the changes in staking account
	testEngine.engine.preprocessEpochForRewarding(context.Background(), epochStart, epochEnd)

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
// from a single available node
func testPreprocessForRewardingWithForceUndelegateSingleValidator(t *testing.T) {
	testEngine := getEngine(t)
	defaultSimpleDelegationState(testEngine, 12, 10)
	epochStart := time.Now()
	epochEnd := time.Now()
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart] = make(map[string]*num.Uint)
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart]["party1"] = num.NewUint(2)
	testEngine.stakingAccounts.partyToStakeForEpoch[epochStart]["party2"] = num.NewUint(0)

	// call preprocess to update the state based on the changes in staking account
	testEngine.engine.preprocessEpochForRewarding(context.Background(), epochStart, epochEnd)

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
// from a multiple validator with equal proportion available node - with is no remainder
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
	testEngine.engine.preprocessEpochForRewarding(context.Background(), epochStart, epochEnd)

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
// from a multiple validator with equal proportion available node - with remainder
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
	testEngine.engine.preprocessEpochForRewarding(context.Background(), epochStart, epochEnd)

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

// undelegate an empty slice of parties, no impact on state
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
	testEngine.engine.processPendingUndelegations([]string{})
	require.Equal(t, 0, len(testEngine.engine.pendingState))
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

// undelegate a party with no delegation, no impact on state
func testPendingUndelegationNothingToUndelegate(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// in this case party3 had delegate state which must have been cleared by the preprocessing step as the party withdrew from the staking account
	// but it still has an undelegation pending for execution - which will have no impact when executed
	testEngine.engine.processPendingUndelegations([]string{"party3"})

	// expect no change in delegation state and clearing of the pending state
	require.Equal(t, 0, len(testEngine.engine.pendingState))
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

// undelegate an more than the delegated balance of party - the whole balance for the party for the node is cleared
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

	testEngine.engine.processPendingUndelegations([]string{"party1", "party2"})
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

// undelegate less than the delegated balance of party - the difference between the balances is remained delegated
func testPendingUndelegationLTDelegateddBalance(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// trying to undelegate more than the node has delegated from the party should just undelegate everything the party has on the node
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(3))
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(1))

	testEngine.engine.processPendingUndelegations([]string{"party1", "party2"})
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

// undelegate the whole balance of a given party from all nodes
func testPendingUndelegationAllBalanceForParty(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// trying to undelegate more than the node has delegated from the party should just undelegate everything the party has on the node
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(6))
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(3))

	testEngine.engine.processPendingUndelegations([]string{"party1", "party2"})
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

// undelegate the whole balance of a given node
func testPendingUndelegationAllBalanceForNode(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// trying to undelegate more than the node has delegated from the party should just undelegate everything the party has on the node
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party1", "node1", num.NewUint(6))
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node1", num.NewUint(2))

	testEngine.engine.processPendingUndelegations([]string{"party1", "party2"})
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

// no pending delegations to process
func testPendingDelegationEmpty(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(12)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(7)

	testEngine.engine.processPendingDelegations([]string{}, num.NewUint(10))
	require.Equal(t, 0, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 0, len(testEngine.engine.partyDelegationState))
}

// delegation at the time of processing of the pending request has insufficient balance in the staking account
func testPendingDelegationInsufficientBalance(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(12)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(7)

	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(10))
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(8)
	testEngine.engine.processPendingDelegations([]string{}, num.NewUint(10))
	require.Equal(t, 0, len(testEngine.engine.nodeDelegationState))
	require.Equal(t, 0, len(testEngine.engine.partyDelegationState))
}

// the validator has all of its allowed allocation filled and it accepts no additional delegation - delegation is ignored
func testPendingDelegationValidatorAllocationMaxedOut(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// party1 has sufficient balance in their staking account to delegate 2 more
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))

	// expect the state hasn't changed and the delegation request has been ignored
	testEngine.engine.processPendingDelegations([]string{"party1", "party2"}, num.NewUint(8))
	require.Equal(t, num.NewUint(8), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(10), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
}

// delegation is adjusted to fit the max delegation per validator parameter
func testPendingDelegationAmountAdjusted(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// party1 has sufficient balance in their staking account to delegate 2 more
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))

	// the delegation amount has been adjusted to 1 and is added to the state
	testEngine.engine.processPendingDelegations([]string{"party1", "party2"}, num.NewUint(9))
	require.Equal(t, num.NewUint(9), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(7), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(11), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(7), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])

}

// process pending delegation successfully
func testPendingDelegationSuccess(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// party1 has sufficient balance in their staking account to delegate 2 more
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))

	// the delegation has been applied on the state
	testEngine.engine.processPendingDelegations([]string{"party1", "party2"}, num.NewUint(10))
	require.Equal(t, num.NewUint(10), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(8), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
	require.Equal(t, num.NewUint(12), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(8), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
}

// process pending delegations and undelegations
func testProcessPending(t *testing.T) {
	// setup committed delegated state
	testEngine := getEngine(t)
	setupDefaultDelegationState(testEngine, 12, 7)

	// party1 has sufficient balance in their staking account to delegate 2 more
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	testEngine.engine.UndelegateAtEndOfEpoch(context.Background(), "party2", "node2", num.NewUint(1))

	// the delegation has been applied on the state
	testEngine.netp.EXPECT().Get("validators.delegation.maxStakePerValidator").AnyTimes().Return("100", nil)

	testEngine.engine.processPending(context.Background())
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
	require.Equal(t, 0, len(testEngine.engine.pendingState))
}

func testGetValidatorsEmpty(t *testing.T) {
	testEngine := getEngine(t)
	validators := testEngine.engine.getValidatorData()
	require.Equal(t, 0, len(validators))
}

func testGetValidatorsSuccess(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(12)
	testEngine.stakingAccounts.partyToStake["party2"] = num.NewUint(7)
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	testEngine.engine.Delegate(context.Background(), "party2", "node2", num.NewUint(5))
	testEngine.netp.EXPECT().Get("validators.delegation.maxStakePerValidator").AnyTimes().Return("100", nil)
	testEngine.engine.processPending(context.Background())
	validators := testEngine.engine.getValidatorData()
	require.Equal(t, 2, len(validators))
	require.Equal(t, "node1", validators[0].NodeID)
	require.Equal(t, num.NewUint(2), validators[0].StakeByDelegators)
	require.Equal(t, 1, len(validators[0].Delegators))
	require.Equal(t, num.NewUint(2), validators[0].Delegators["party1"])
	require.Equal(t, "node2", validators[1].NodeID)
	require.Equal(t, num.NewUint(5), validators[1].StakeByDelegators)
	require.Equal(t, 1, len(validators[1].Delegators))
	require.Equal(t, num.NewUint(5), validators[1].Delegators["party2"])

}

// try to undelegate more than delegated
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

// undelegate all now, there are no committed delegations for the node, only pending and they are all cleared
func testUndelegateNowAllWithPendingOnly(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.topology.nodeToIsValidator["node1"] = true
	testEngine.topology.nodeToIsValidator["node2"] = true
	testEngine.stakingAccounts.partyToStake["party1"] = num.NewUint(30)
	testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(10))
	testEngine.engine.Delegate(context.Background(), "party1", "node2", num.NewUint(10))

	err := testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.Zero())
	require.Nil(t, err)

	require.Equal(t, 1, len(testEngine.engine.pendingState["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party1"].nodeToUndelegateAmount))
	require.Equal(t, num.NewUint(10), testEngine.engine.pendingState["party1"].totalDelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party1"].totalUndelegation)
	require.Equal(t, num.NewUint(10), testEngine.engine.pendingState["party1"].nodeToDelegateAmount["node2"])
}

// there's no pending delegation, remove all committed delegation
func testUndelegateNowAllWithCommittedOnly(t *testing.T) {
	testEngine := getEngine(t)
	// setup delegation state
	setupDefaultDelegationState(testEngine, 12, 7)

	// undelegate now all for party1 node1
	err := testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.Zero())
	require.Nil(t, err)

	require.Equal(t, 0, len(testEngine.engine.pendingState))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])

	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.nodeDelegationState["node1"].partyToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])

	// undelegate now all for party1 node2
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node2", num.Zero())
	require.Nil(t, err)
	require.Equal(t, 0, len(testEngine.engine.pendingState))
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState))
}

// there's both committed and pending delegation, take all from both
func testUndelegateNowAll(t *testing.T) {
	testEngine := getEngine(t)
	// setup delegation state
	setupDefaultDelegationState(testEngine, 12, 7)

	err := testEngine.engine.Delegate(context.Background(), "party1", "node1", num.NewUint(2))
	require.Nil(t, err)

	// undelegate now all for party1 node1 both committed and pending state should update
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.Zero())
	require.Nil(t, err)

	require.Equal(t, 0, len(testEngine.engine.pendingState))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])

	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.nodeDelegationState["node1"].partyToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])

	// undelegate now all for party1 node2
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node2", num.Zero())
	require.Nil(t, err)
	require.Equal(t, 0, len(testEngine.engine.pendingState))
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

	require.Equal(t, 2, len(testEngine.engine.pendingState["party1"].nodeToDelegateAmount))
	require.Equal(t, 0, len(testEngine.engine.pendingState["party1"].nodeToUndelegateAmount))
	require.Equal(t, num.NewUint(15), testEngine.engine.pendingState["party1"].totalDelegation)
	require.Equal(t, num.Zero(), testEngine.engine.pendingState["party1"].totalUndelegation)
	require.Equal(t, num.NewUint(5), testEngine.engine.pendingState["party1"].nodeToDelegateAmount["node1"])
	require.Equal(t, num.NewUint(10), testEngine.engine.pendingState["party1"].nodeToDelegateAmount["node2"])
}

func testUndelegateNowWithCommittedOnly(t *testing.T) {
	testEngine := getEngine(t)
	// setup delegation state
	setupDefaultDelegationState(testEngine, 12, 7)

	// undelegate now all for party1 node1
	err := testEngine.engine.UndelegateNow(context.Background(), "party1", "node1", num.NewUint(4))
	require.Nil(t, err)

	require.Equal(t, 0, len(testEngine.engine.pendingState))
	require.Equal(t, num.NewUint(6), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 2, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])

	require.Equal(t, num.NewUint(4), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState["node1"].partyToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
}

// undelegate now amount is fully covered in pending delegation, the committed state is unchanged
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
	require.Equal(t, 0, len(testEngine.engine.pendingState))

	// committed state should have stayed the same
	require.Equal(t, num.NewUint(10), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(8), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, num.NewUint(6), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party1"])
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])
}

// undelegate now takes all pending and some of the committed delegation
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
	require.Equal(t, 0, len(testEngine.engine.pendingState))

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

// undelegate now with an amount equals to the total delegated (pending + committed)
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
	require.Equal(t, 0, len(testEngine.engine.pendingState))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(4), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])

	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].totalDelegated)
	require.Equal(t, 1, len(testEngine.engine.nodeDelegationState["node1"].partyToAmount))
	require.Equal(t, num.NewUint(2), testEngine.engine.nodeDelegationState["node1"].partyToAmount["party2"])

	// undelegate now all for party1 node2
	err = testEngine.engine.UndelegateNow(context.Background(), "party1", "node2", num.NewUint(4))
	require.Nil(t, err)
	require.Equal(t, 0, len(testEngine.engine.pendingState))
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState))
	require.Equal(t, 2, len(testEngine.engine.nodeDelegationState))
}

func getEngine(t *testing.T) *testEngine {
	conf := NewDefaultConfig()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	logger := logging.NewTestLogger()
	stakingAccounts := newTestStakingAccount()
	netp := gmock.NewMockNetParams(ctrl)
	topology := newTestTopology()
	engine := New(logger, conf, broker, topology, stakingAccounts, netp)

	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	netp.EXPECT().Get("validators.delegation.minAmount").AnyTimes().Return("2", nil)

	return &testEngine{
		engine:          engine,
		ctrl:            ctrl,
		broker:          broker,
		stakingAccounts: stakingAccounts,
		topology:        topology,
		netp:            netp,
	}
}

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

func (t *TestStakingAccount) GetBalanceNow(party string) *num.Uint {
	ret := t.partyToStake[party]
	return ret
}

func (t *TestStakingAccount) GetBalanceForEpoch(party string, from, to time.Time) *num.Uint {
	ret, ok := t.partyToStakeForEpoch[from]
	if !ok {
		return nil
	}
	return ret[party]
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
