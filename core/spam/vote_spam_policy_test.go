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

package spam_test

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/spam"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/stretchr/testify/require"
)

type testTx struct {
	party    string
	proposal string
	command  txn.Command
}

func (*testTx) GetPoWNonce() uint64 { return 0 }
func (*testTx) GetPoWTID() string   { return "" }
func (*testTx) GetVersion() uint32  { return 2 }

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

func getVotingSpamPolicy(accounts map[string]*num.Uint) *spam.VoteSpamPolicy {
	logger := logging.NewTestLogger()
	testAccounts := testAccounts{balances: accounts}
	policy := spam.NewVoteSpamPolicy(netparams.SpamProtectionMinTokensForVoting, netparams.SpamProtectionMaxVotes, logger, testAccounts)
	minTokensForVoting, _ := num.UintFromString("100000000000000000000", 10)
	policy.UpdateUintParam(netparams.SpamProtectionMinTokensForVoting, minTokensForVoting)
	policy.UpdateIntParam(netparams.SpamProtectionMaxVotes, 3)
	return policy
}

// reject vote requests when the voter doesn't have sufficient balance at the beginning of the epoch.
func testPreRejectInsufficientBalance(t *testing.T) {
	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": num.NewUint(50)})
	policy.Reset(types.Epoch{Seq: 0})
	tx := &testTx{party: "party1", proposal: "proposal1"}
	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, spam.ErrInsufficientTokensForVoting, err)
}

// reject votes requests when the voter doesn't have sufficient balance with a factored min tokens.
func testPreRejectInsufficientBalanceWithFactor(t *testing.T) {
	// epoch 0 started party1 has enough balance without doubling, party2 has enough balance with doubling
	tokenMap := make(map[string]*num.Uint, 2)
	tokenMap["party1"] = sufficientTokensForVoting
	tokenMap["party2"] = sufficientTokens2ForVoting
	policy := getVotingSpamPolicy(tokenMap)

	policy.Reset(types.Epoch{Seq: 0})

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
	policy.EndOfBlock(1, time.Now(), time.Minute*30)

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
	// epoch 0 started party1 has enough balance without doubling, party2 has enough balance with doubling
	tokenMap := make(map[string]*num.Uint, 2)
	tokenMap["party1"] = sufficientTokensForVoting
	tokenMap["party2"] = sufficientTokens2ForVoting
	tokenMap["party3"] = sufficientTokens4ForVoting
	tokenMap["party4"] = sufficientTokens8ForVoting
	tokenMap["party5"] = maxSufficientTokensForVoting
	policy := getVotingSpamPolicy(tokenMap)

	policy.Reset(types.Epoch{Seq: 0})

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
			policy.EndOfBlock(uint64(i*10+j)+1, time.Now(), time.Minute*30)
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
	policy.Reset(types.Epoch{Seq: 0})

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
			policy.EndOfBlock(uint64(i*10+j)+1, time.Now(), time.Minute*30)
		}
	}
}

// reject vote requests from banned parties for as long as they are banned.
func testPreRejectBannedParty(t *testing.T) {
	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokens2ForVoting})

	// epoch 0 started party1 has enough balance
	policy.Reset(types.Epoch{Seq: 0})

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
	tm, _ := time.Parse("2006-01-02 15:04", "2022-12-12 04:35")
	policy.EndOfBlock(1, tm, time.Minute*30)

	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, "party is banned from submitting votes until the earlier between 2022-12-12 05:05:00 +0000 GMT and the beginning of the next epoch", err.Error())

	// advance 30 minutes - verify still banned until 30 minutes pass
	for i := 0; i < 3; i++ {
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, false, accept)
		require.Equal(t, "party is banned from submitting votes until the earlier between 2022-12-12 05:05:00 +0000 GMT and the beginning of the next epoch", err.Error())
		adjustment := 10 * time.Minute * time.Duration(i+1)
		policy.EndOfBlock(1, tm.Add(adjustment), time.Minute*30)
	}
	// should be released from ban now but should still fail as they have already voted 3 times
	accept, err = policy.PreBlockAccept(tx)
	require.Equal(t, "party has already voted the maximum number of times per proposal per epoch", err.Error())
	require.Equal(t, false, accept)
}

func testPreRejectTooManyVotesPerProposal(t *testing.T) {
	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
	// epoch 0 block 0
	policy.Reset(types.Epoch{Seq: 0})

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
	policy.EndOfBlock(1, time.Now(), time.Minute*30)

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
	policy.Reset(types.Epoch{Seq: 0})
	for i := 0; i < 3; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// pre rejected
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
	}
}

func testPreAccept(t *testing.T) {
	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
	// epoch 0 block 0
	policy.Reset(types.Epoch{Seq: 0})

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
	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
	policy.Reset(types.Epoch{Seq: 0})
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
	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
	policy.Reset(types.Epoch{Seq: 0})
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
	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
	policy.Reset(types.Epoch{Seq: 0})
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

	policy.EndOfBlock(1, time.Now(), time.Minute*30)
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
	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
	policy.Reset(types.Epoch{Seq: 0})
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

	// trigger ban of party1 for 30 minutes
	tm, _ := time.Parse("2006-01-02 15:04", "2022-12-12 04:35")
	policy.EndOfBlock(1, tm, time.Minute*30)

	policy.EndOfBlock(1, tm.Add(10*time.Minute), time.Minute*30)
	tx := &testTx{party: "party1", proposal: "proposal1"}
	accept, err := policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, "party is banned from submitting votes until the earlier between 2022-12-12 05:05:00 +0000 GMT and the beginning of the next epoch", err.Error())

	policy.EndOfBlock(1, tm.Add(20*time.Minute), time.Minute*30)
	accept, err = policy.PreBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, "party is banned from submitting votes until the earlier between 2022-12-12 05:05:00 +0000 GMT and the beginning of the next epoch", err.Error())

	policy.EndOfBlock(1, tm.Add(30*time.Minute), time.Minute*30)
	accept, err = policy.PostBlockAccept(tx)
	require.Equal(t, false, accept)
	require.Equal(t, "party has already voted the maximum number of times per proposal per epoch", err.Error())
}

func testVoteEndBlockReset(t *testing.T) {
	// set state
	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
	policy.Reset(types.Epoch{Seq: 0})

	// in each block we vote once
	var i uint64
	for ; i < 3; i++ {
		tx := &testTx{party: "party1", proposal: "proposal1"}
		accept, err := policy.PreBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)

		accept, err = policy.PostBlockAccept(tx)
		require.Equal(t, true, accept)
		require.Nil(t, err)
		policy.EndOfBlock(i, time.Now(), time.Minute*30)
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
	policy.EndOfBlock(3, time.Now(), time.Minute*30)

	bytes3, err := policy.Serialise()
	require.Nil(t, err)
	require.False(t, bytes.Equal(bytes3, bytes2))
}
