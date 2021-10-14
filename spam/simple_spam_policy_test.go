package spam_test

import (
	"bytes"
	"errors"
	"strconv"
	"testing"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/spam"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

var insufficientPropTokens, _ = num.UintFromString("50000000000000000000000", 10)

var sufficientPropTokens, _ = num.UintFromString("100000000000000000000000", 10)

func TestSimpleSpamProtection(t *testing.T) {
	t.Run("Pre reject command from party with insufficient balance at the beginning of the epoch", testCommandPreRejectInsufficientBalance)
	t.Run("Pre reject command from party that is banned for the epochs", testCommandPreRejectBannedParty)
	t.Run("Pre reject command from party that already had more than 3 proposal for the epoch", testCommandPreRejectTooManyProposals)
	t.Run("Pre accept command success", testCommandPreAccept)
	t.Run("Post accept command success", testCommandPostAccept)
	t.Run("Post reject command from party with too many proposals in total all from current block", testCommandPostRejectTooManyProposals)
	t.Run("command counts from the block successfully applied on state", testCommandCountersUpdated)
	t.Run("Start of epoch resets counters", testCommandReset)
	t.Run("On end of block, block proposal counters are reset and take a snapshot roundtrip", testProposalEndBlockReset)
}

func getCommandSpamPolicy() *spam.SimpleSpamPolicy {
	logger := logging.NewTestLogger()
	policy := spam.NewSimpleSpamPolicy("simple", netparams.SpamProtectionMinTokensForProposal, netparams.SpamProtectionMaxProposals, logger)
	minTokensForProposal, _ := num.UintFromString("100000000000000000000000", 10)
	policy.UpdateUintParam(netparams.SpamProtectionMinTokensForProposal, minTokensForProposal)
	policy.UpdateIntParam(netparams.SpamProtectionMaxProposals, 3)
	return policy
}

func testProposalEndBlockReset(t *testing.T) {
	// set state
	policy := getCommandSpamPolicy()

	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientPropTokens

	policy.Reset(types.Epoch{Seq: 0}, tokenMap)

	// in each block we vote once
	var i uint64 = 0
	for ; i < 3; i++ {
		tx := &testTx{party: "party1", proposal: "proposal1"}
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)

		accept, err = policy.PostBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
		policy.EndOfBlock(i)
	}

	tx := &testTx{party: "party1", proposal: "proposal1"}
	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, errors.New("party has already proposed the maximum number of simple requests per epoch"), err)

	bytes1, err := policy.Serialise()
	require.Nil(t, err)
	var proposalPayload snapshot.Payload
	proto.Unmarshal(bytes1, &proposalPayload)
	payload := types.PayloadFromProto(&proposalPayload)
	policy.Deserialise(payload)
	bytes2, err := policy.Serialise()
	require.Nil(t, err)
	require.True(t, bytes.Equal(bytes1, bytes2))

	policy.Reset(types.Epoch{Seq: 1}, tokenMap)

	bytes3, err := policy.Serialise()
	require.Nil(t, err)
	require.False(t, bytes.Equal(bytes3, bytes2))
}

// reject proposal when the proposer doesn't have sufficient balance at the beginning of the epoch.
func testCommandPreRejectInsufficientBalance(t *testing.T) {
	policy := getCommandSpamPolicy()

	policy.Reset(types.Epoch{Seq: 0}, map[string]*num.Uint{"party1": insufficientPropTokens})
	tx := &testTx{party: "party1", proposal: "proposal1"}
	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, errors.New("party has insufficient tokens to submit simple request in this epoch"), err)
}

// reject proposal requests from banned parties for as long as they are banned.
func testCommandPreRejectBannedParty(t *testing.T) {
	policy := getCommandSpamPolicy()

	// epoch 0 started party1 has enough balance
	policy.Reset(types.Epoch{Seq: 0}, map[string]*num.Uint{"party1": sufficientPropTokens})

	// trigger banning of party1 by causing it to post reject 3/6 of the proposal
	tx := &testTx{party: "party1", proposal: "proposal1"}
	for i := 0; i < 6; i++ {
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}
	for i := 0; i < 3; i++ {
		accept, err := policy.PostBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}
	for i := 0; i < 3; i++ {
		accept, err := policy.PostBlockAccept(tx)
		require.Equal(t, false, accept)
		require.Equal(t, errors.New("party has already proposed the maximum number of simple requests per epoch"), err)
	}

	// end the block for banning to take place
	policy.EndOfBlock(1)

	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, errors.New("party is banned from submitting simple in the current epoch"), err)

	// advance epochs - verify still banned until epoch 4 (including)
	for i := 0; i < 4; i++ {
		policy.Reset(types.Epoch{Seq: uint64(i + 1)}, map[string]*num.Uint{"party1": sufficientPropTokens})
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, false, accept)
		require.Equal(t, errors.New("party is banned from submitting simple in the current epoch"), err)
	}
	// should be released from ban on epoch 5
	policy.Reset(types.Epoch{Seq: 5}, map[string]*num.Uint{"party1": sufficientPropTokens})
	accept, err = policy.PreBlockAccept(tx)
	require.Equal(t, true, accept)
	require.Nil(t, err)
}

func testCommandPreRejectTooManyProposals(t *testing.T) {
	policy := getCommandSpamPolicy()
	// epoch 0 block 0
	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientPropTokens
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)

	// propose 4 proposals, all preaccepted 3 post accepted
	tx := &testTx{party: "party1", proposal: "proposal1"}
	// pre accepted
	for i := 0; i < 4; i++ {
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}

	// 3 post accepted
	for i := 0; i < 3; i++ {
		accept, err := policy.PostBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}

	// end block 0
	policy.EndOfBlock(1)

	// try to submit proposal - pre rejected because it already have 3 proposals
	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, errors.New("party has already proposed the maximum number of simple requests per epoch"), err)

	// advance to next epoch to reset limits
	policy.Reset(types.Epoch{Seq: 1}, tokenMap)
	for i := 0; i < 3; i++ {
		tx := &testTx{party: "party1", proposal: "proposal1"}
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}
}

func testCommandPreAccept(t *testing.T) {
	policy := getCommandSpamPolicy()
	// epoch 0 block 0
	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientPropTokens
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)

	// propose 5 times all pre accepted, 3 for each post accepted
	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// pre accepted
		for i := 0; i < 5; i++ {
			accept, err := policy.PreBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}
	}
}

func testCommandPostAccept(t *testing.T) {
	policy := getCommandSpamPolicy()
	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientPropTokens
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)
	tx := &testTx{party: "party1", proposal: "proposal1"}
	// pre accepted
	for i := 0; i < 3; i++ {
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}

	// post accepted
	for i := 0; i < 3; i++ {
		accept, err := policy.PostBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}
}

func testCommandPostRejectTooManyProposals(t *testing.T) {
	policy := getCommandSpamPolicy()
	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientPropTokens
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)
	tx := &testTx{party: "party1", proposal: "proposal1"}
	// pre accepted
	for i := 0; i < 5; i++ {
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}

	// post accepted
	for i := 0; i < 3; i++ {
		accept, err := policy.PostBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}

	// post rejected
	for i := 0; i < 2; i++ {
		accept, err := policy.PostBlockAccept(tx)
		require.Equal(t, false, accept)
		require.Equal(t, errors.New("party has already proposed the maximum number of simple requests per epoch"), err)
	}
}

func testCommandCountersUpdated(t *testing.T) {
	policy := getCommandSpamPolicy()
	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientPropTokens
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)

	tx := &testTx{party: "party1", proposal: "proposal"}
	// pre accepted
	for i := 0; i < 3; i++ {
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}

	// post accepted
	for i := 0; i < 3; i++ {
		accept, err := policy.PostBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}

	policy.EndOfBlock(1)
	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, errors.New("party has already proposed the maximum number of simple requests per epoch"), err)
}

func testCommandReset(t *testing.T) {
	// set state
	policy := getCommandSpamPolicy()

	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientPropTokens

	policy.Reset(types.Epoch{Seq: 0}, tokenMap)
	tx := &testTx{party: "party1", proposal: "proposal1"}
	// pre accepted
	for i := 0; i < 6; i++ {
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}
	// post accepted
	for i := 0; i < 3; i++ {
		accept, err := policy.PostBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}

	for i := 0; i < 3; i++ {
		accept, err := policy.PostBlockAccept(tx)
		require.Equal(t, false, accept)
		require.Equal(t, errors.New("party has already proposed the maximum number of simple requests per epoch"), err)
	}

	// trigger ban of party1 until epoch 4
	policy.EndOfBlock(1)
	// verify reset at the start of new epoch, party1 should still be banned for the epoch until epoch 4
	policy.Reset(types.Epoch{Seq: 1}, tokenMap)
	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, errors.New("party is banned from submitting simple in the current epoch"), err)

	// advance epochs until the ban is lifted fro party1
	policy.Reset(types.Epoch{Seq: 2}, tokenMap)
	accept, err = policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, errors.New("party is banned from submitting simple in the current epoch"), err)

	policy.Reset(types.Epoch{Seq: 3}, tokenMap)
	accept, err = policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, errors.New("party is banned from submitting simple in the current epoch"), err)

	policy.Reset(types.Epoch{Seq: 4}, tokenMap)
	accept, err = policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, errors.New("party is banned from submitting simple in the current epoch"), err)

	policy.Reset(types.Epoch{Seq: 5}, tokenMap)
	accept, err = policy.PreBlockAccept(tx)
	require.Equal(t, true, accept)
	require.Nil(t, err)
}
