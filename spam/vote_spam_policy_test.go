package spam_test

import (
	"bytes"
	"strconv"
	"testing"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/spam"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

type testTx struct {
	party    string
	proposal string
	command  txn.Command
}

var (
	sufficientTokensForVoting, _    = num.UintFromString("100000000000000000000", 10)
	sufficientTokens2ForVoting, _   = num.UintFromString("200000000000000000000", 10)
	sufficientTokens4ForVoting, _   = num.UintFromString("400000000000000000000", 10)
	sufficientTokens8ForVoting, _   = num.UintFromString("800000000000000000000", 10)
	maxSufficientTokensForVoting, _ = num.UintFromString("1600000000000000000000", 10)
)

func TestVotingSpamProtection(t *testing.T) {
	t.Run("Pre reject vote from party with insufficient balance at the beginning of the epoch", testPreRejectInsufficientBalance)
	t.Run("Pre reject vote from party with insufficient balance at the beginning of the epoch vs factored min tokens", testPreRejectInsufficientBalanceWithFactor)
	t.Run("Double min tokens until the max", testFactoringOfMinTokens)
	t.Run("Pre reject vote from party that is banned for the epochs", testPreRejectBannedParty)
	t.Run("Pre reject vote from party that already had more than 3 votes for the epoch", testPreRejectTooManyVotesPerProposal)
	t.Run("Pre accept vote success", testPreAccept)
	t.Run("Post accept vote success", testPostAccept)
	t.Run("Post reject vote from party with too many votes in total all from current block", testPostRejectTooManyVotes)
	t.Run("Vote counts from the block carried over to next block", testCountersUpdated)
	t.Run("On epoch start voting counters are reset", testReset)
	t.Run("On end of block, block voting counters are reset and take a snapshot roundtrip", testVoteEndBlockReset)
}

func getVotingSpamPolicy() *spam.VoteSpamPolicy {
	logger := logging.NewTestLogger()
	policy := spam.NewVoteSpamPolicy(netparams.SpamProtectionMinTokensForVoting, netparams.SpamProtectionMaxVotes, logger)
	minTokensForVoting, _ := num.UintFromString("100000000000000000000", 10)
	policy.UpdateUintParam(netparams.SpamProtectionMinTokensForVoting, minTokensForVoting)
	policy.UpdateIntParam(netparams.SpamProtectionMaxVotes, 3)
	return policy
}

// reject vote requests when the voter doesn't have sufficient balance at the beginning of the epoch.
func testPreRejectInsufficientBalance(t *testing.T) {
	policy := getVotingSpamPolicy()
	policy.Reset(types.Epoch{Seq: 0}, map[string]*num.Uint{"party1": num.NewUint(50)})
	tx := &testTx{party: "party1", proposal: "proposal1"}
	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, spam.ErrInsufficientTokensForVoting, err)
}

// reject votes requests when the voter doesn't have sufficient balance with a factored min tokens.
func testPreRejectInsufficientBalanceWithFactor(t *testing.T) {
	policy := getVotingSpamPolicy()
	// epoch 0 started party1 has enough balance without doubling, party2 has enough balance with doubling
	tokenMap := make(map[string]*num.Uint, 2)
	tokenMap["party1"] = sufficientTokensForVoting
	tokenMap["party2"] = sufficientTokens2ForVoting
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)

	// make 30% of transactions post fail
	tx1 := &testTx{party: "party1", proposal: "proposal1"}
	tx2 := &testTx{party: "party2", proposal: "proposal2"}

	// party1 submits 5 votes for proposal 1 (with nothing earlier in the epoch)
	for i := 0; i < 5; i++ {
		accept, err := policy.PreBlockAccept(tx1)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}

	// party2 submits 4 votes for proposal 2 (with nothing earlier in the epoch)
	for i := 0; i < 5; i++ {
		accept, err := policy.PreBlockAccept(tx2)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}

	// party1 gets 3 post accepted and 2 post rejected
	for i := 0; i < 3; i++ {
		accept, err := policy.PostBlockAccept(tx1)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}
	for i := 0; i < 2; i++ {
		accept, err := policy.PostBlockAccept(tx1)
		require.Equal(t, false, accept)
		require.Equal(t, spam.ErrTooManyVotes, err)
	}

	// party2 gets 3 post accepted and 1 post rejected
	for i := 0; i < 3; i++ {
		accept, err := policy.PostBlockAccept(tx2)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}

	accept, err := policy.PostBlockAccept(tx2)
	require.Equal(t, false, accept)
	require.Equal(t, spam.ErrTooManyVotes, err)

	// end the block for doubling of min amount to take place
	policy.EndOfBlock(1)

	// in the next block party1 should not have enough balance to vote while party2 still has, but has no more votes
	accept, err = policy.PreBlockAccept(tx1)
	require.Equal(t, false, accept)
	require.Equal(t, spam.ErrInsufficientTokensForVoting, err)

	accept, err = policy.PreBlockAccept(tx2)
	require.Equal(t, false, accept)
	require.Equal(t, spam.ErrTooManyVotes, err)
}

// attack for a number of blocks until the min tokens reach 1600.
func testFactoringOfMinTokens(t *testing.T) {
	policy := getVotingSpamPolicy()
	// epoch 0 started party1 has enough balance without doubling, party2 has enough balance with doubling
	tokenMap := make(map[string]*num.Uint, 2)
	tokenMap["party1"] = sufficientTokensForVoting
	tokenMap["party2"] = sufficientTokens2ForVoting
	tokenMap["party3"] = sufficientTokens4ForVoting
	tokenMap["party4"] = sufficientTokens8ForVoting
	tokenMap["party5"] = maxSufficientTokensForVoting
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)

	// party x submits 5 votes for proposal 1 (with nothing earlier in the epoch)
	for i := 0; i < 4; i++ {
		tx := &testTx{party: "party" + strconv.Itoa(i+1), proposal: "proposal" + strconv.Itoa(i+1)}
		// pre accepted
		for i := 0; i < 5; i++ {
			accept, err := policy.PreBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}

		// post 3 accepted 2 rejected
		for i := 0; i < 3; i++ {
			accept, err := policy.PostBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}
		for i := 0; i < 2; i++ {
			accept, err := policy.PostBlockAccept(tx)
			require.Equal(t, false, accept)
			require.Equal(t, spam.ErrTooManyVotes, err)
		}

		for j := 0; j < 11; j++ {
			// the first end of block will double the amount, the following 10 will have no impact on doubling
			policy.EndOfBlock(uint64(i*10+j) + 1)
		}
	}

	// at this point we expect the min tokens to be the max so all but party5 shall be pre rejected
	for i := 1; i < 5; i++ {
		tx := &testTx{party: "party" + strconv.Itoa(i), proposal: "proposal" + strconv.Itoa(i)}
		// pre rejected
		for i := 0; i < 5; i++ {
			accept, err := policy.PreBlockAccept(tx)
			require.Equal(t, false, accept)
			require.Equal(t, spam.ErrInsufficientTokensForVoting, err)
		}
	}

	tx := &testTx{party: "party5", proposal: "proposal5"}
	for i := 0; i < 5; i++ {
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}

	for i := 0; i < 3; i++ {
		accept, err := policy.PostBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}
	for i := 0; i < 2; i++ {
		accept, err := policy.PostBlockAccept(tx)
		require.Equal(t, false, accept)
		require.Equal(t, spam.ErrTooManyVotes, err)
	}

	// advance to the next epoch so we reset the balances and all should be able to succeed with their token balances
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)

	for i := 0; i < 4; i++ {
		tx := &testTx{party: "party" + strconv.Itoa(i+1), proposal: "proposal" + strconv.Itoa(i+1)}
		// pre accepted
		for i := 0; i < 5; i++ {
			accept, err := policy.PreBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}

		// post 3 accepted 2 rejected
		for i := 0; i < 3; i++ {
			accept, err := policy.PostBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}
		for i := 0; i < 2; i++ {
			accept, err := policy.PostBlockAccept(tx)
			require.Equal(t, false, accept)
			require.Equal(t, spam.ErrTooManyVotes, err)
		}

		for j := 0; j < 11; j++ {
			// the first end of block will double the amount, the following 10 will have no impact on doubling
			policy.EndOfBlock(uint64(i*10+j) + 1)
		}
	}
}

// reject vote requests from banned parties for as long as they are banned.
func testPreRejectBannedParty(t *testing.T) {
	policy := getVotingSpamPolicy()

	// epoch 0 started party1 has enough balance
	policy.Reset(types.Epoch{Seq: 0}, map[string]*num.Uint{"party1": sufficientTokensForVoting})

	// trigger banning of party1 by causing it to post reject 3/6 of the requests to vote
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
		require.Equal(t, spam.ErrTooManyVotes, err)
	}

	// end the block for banning to take place
	policy.EndOfBlock(1)

	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, spam.ErrPartyIsBannedFromVoting, err)

	// advance epochs - verify still banned until epoch 4 (including)
	for i := 0; i < 4; i++ {
		policy.Reset(types.Epoch{Seq: uint64(i + 1)}, map[string]*num.Uint{"party1": sufficientTokensForVoting})
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, false, accept)
		require.Equal(t, spam.ErrPartyIsBannedFromVoting, err)
	}
	// should be released from ban on epoch 5
	policy.Reset(types.Epoch{Seq: 5}, map[string]*num.Uint{"party1": sufficientTokensForVoting})
	accept, err = policy.PreBlockAccept(tx)
	require.Equal(t, true, accept)
	require.Nil(t, err)
}

func testPreRejectTooManyVotesPerProposal(t *testing.T) {
	policy := getVotingSpamPolicy()
	// epoch 0 block 0
	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientTokensForVoting
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)

	// vote 5 times for each proposal all pre accepted, 3 for each post accepted
	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// pre accepted
		for i := 0; i < 5; i++ {
			accept, err := policy.PreBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}

		// post 3 accepted 1 rejected
		for i := 0; i < 3; i++ {
			accept, err := policy.PostBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}
	}

	// end block 0
	policy.EndOfBlock(1)

	// try to submit
	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// pre rejected
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, false, accept)
		require.Equal(t, spam.ErrTooManyVotes, err)
	}

	tx := &testTx{party: "party1", proposal: "proposal3"}
	// pre accepted
	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, true, accept)
	require.Nil(t, err)

	// advance to next epoch to reset limits
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)
	for i := 0; i < 3; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// pre rejected
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}
}

func testPreAccept(t *testing.T) {
	policy := getVotingSpamPolicy()
	// epoch 0 block 0
	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientTokensForVoting
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)

	// vote 5 times for each proposal all pre accepted, 3 for each post accepted
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

func testPostAccept(t *testing.T) {
	policy := getVotingSpamPolicy()
	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientTokensForVoting
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)
	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// pre accepted
		for i := 0; i < 3; i++ {
			accept, err := policy.PreBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}
	}

	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// post accepted
		for i := 0; i < 3; i++ {
			accept, err := policy.PostBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}
	}
}

func testPostRejectTooManyVotes(t *testing.T) {
	policy := getVotingSpamPolicy()
	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientTokensForVoting
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)
	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// pre accepted
		for i := 0; i < 5; i++ {
			accept, err := policy.PreBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}
	}

	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
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
			require.Equal(t, spam.ErrTooManyVotes, err)
		}
	}
}

func testCountersUpdated(t *testing.T) {
	policy := getVotingSpamPolicy()
	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientTokensForVoting
	policy.Reset(types.Epoch{Seq: 0}, tokenMap)
	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// pre accepted
		for i := 0; i < 2; i++ {
			accept, err := policy.PreBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}
	}

	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// post accepted
		for i := 0; i < 2; i++ {
			accept, err := policy.PostBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}
	}

	policy.EndOfBlock(1)
	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// pre accepted
		for i := 0; i < 2; i++ {
			accept, err := policy.PreBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}
	}
	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// post accepted
		accept, err := policy.PostBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)

		// post rejected
		accept, err = policy.PostBlockAccept(tx)
		require.Equal(t, false, accept)
		require.Equal(t, spam.ErrTooManyVotes, err)
	}
}

func testReset(t *testing.T) {
	// set state
	policy := getVotingSpamPolicy()

	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientTokensForVoting

	policy.Reset(types.Epoch{Seq: 0}, tokenMap)
	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// pre accepted
		for i := 0; i < 6; i++ {
			accept, err := policy.PreBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}
	}

	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// post accepted
		for i := 0; i < 3; i++ {
			accept, err := policy.PostBlockAccept(tx)
			require.Equal(t, true, accept)
			require.Nil(t, err)
		}

		for i := 0; i < 3; i++ {
			accept, err := policy.PostBlockAccept(tx)
			require.Equal(t, false, accept)
			require.Equal(t, spam.ErrTooManyVotes, err)
		}
	}

	// trigger ban of party1 until epoch 4
	policy.EndOfBlock(1)
	// verify reset at the start of new epoch, party1 should still be banned for the epoch until epoch 4
	policy.Reset(types.Epoch{Seq: 1}, tokenMap)
	tx := &testTx{party: "party1", proposal: "proposal1"}
	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, spam.ErrPartyIsBannedFromVoting, err)

	policy.Reset(types.Epoch{Seq: 2}, tokenMap)
	accept, err = policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, spam.ErrPartyIsBannedFromVoting, err)

	policy.Reset(types.Epoch{Seq: 3}, tokenMap)
	accept, err = policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, spam.ErrPartyIsBannedFromVoting, err)

	policy.Reset(types.Epoch{Seq: 4}, tokenMap)
	accept, err = policy.PostBlockAccept(tx)
	require.Equal(t, true, accept)
	require.Nil(t, err)
}

func testVoteEndBlockReset(t *testing.T) {
	// set state
	policy := getVotingSpamPolicy()

	tokenMap := make(map[string]*num.Uint, 1)
	tokenMap["party1"] = sufficientTokensForVoting

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
	require.Equal(t, spam.ErrTooManyVotes, err)

	bytes1, err := policy.Serialise()
	require.Nil(t, err)
	var votePayload snapshot.Payload
	proto.Unmarshal(bytes1, &votePayload)
	payload := types.PayloadFromProto(&votePayload)
	policy.Deserialise(payload)
	bytes2, err := policy.Serialise()
	require.Nil(t, err)
	require.True(t, bytes.Equal(bytes1, bytes2))
	policy.EndOfBlock(3)

	bytes3, err := policy.Serialise()
	require.Nil(t, err)
	require.False(t, bytes.Equal(bytes3, bytes2))
}
