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

// import (
// 	"bytes"
// 	"strconv"
// 	"testing"

// 	"code.vegaprotocol.io/vega/core/netparams"
// 	"code.vegaprotocol.io/vega/core/spam"
// 	"code.vegaprotocol.io/vega/core/txn"
// 	"code.vegaprotocol.io/vega/core/types"
// 	"code.vegaprotocol.io/vega/libs/num"
// 	"code.vegaprotocol.io/vega/libs/proto"
// 	"code.vegaprotocol.io/vega/logging"
// 	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
// 	"github.com/stretchr/testify/require"
// )

// type testTx struct {
// 	party    string
// 	proposal string
// 	command  txn.Command
// }

// func (*testTx) GetLength() int      { return 0 }
// func (*testTx) GetPoWNonce() uint64 { return 0 }
// func (*testTx) GetPoWTID() string   { return "" }
// func (*testTx) GetVersion() uint32  { return 2 }

// var sufficientTokensForVoting, _ = num.UintFromString("100000000000000000000", 10)

// func TestVotingSpamProtection(t *testing.T) {
// 	t.Run("Pre reject vote from party with insufficient balance at the beginning of the epoch", testPreRejectInsufficientBalance)
// 	t.Run("Pre reject vote from party that already had more than 3 votes for the epoch", testPreRejectTooManyVotesPerProposal)
// 	t.Run("Pre accept vote success", testPreAccept)
// 	t.Run("Post accept vote success", testPostAccept)
// 	t.Run("Vote counts from the block carried over to next block", testCountersUpdated)
// 	t.Run("On epoch start voting counters are reset", testReset)
// 	t.Run("On end of block, block voting counters are reset and take a snapshot roundtrip", testVoteEndBlockReset)
// }

// func getVotingSpamPolicy(accounts map[string]*num.Uint) *spam.VoteSpamPolicy {
// 	logger := logging.NewTestLogger()
// 	testAccounts := testAccounts{balances: accounts}
// 	policy := spam.NewVoteSpamPolicy(netparams.SpamProtectionMinTokensForVoting, netparams.SpamProtectionMaxVotes, logger, testAccounts)
// 	minTokensForVoting, _ := num.UintFromString("100000000000000000000", 10)
// 	policy.UpdateUintParam(netparams.SpamProtectionMinTokensForVoting, minTokensForVoting)
// 	policy.UpdateIntParam(netparams.SpamProtectionMaxVotes, 3)
// 	return policy
// }

// // reject vote requests when the voter doesn't have sufficient balance at the beginning of the epoch.
// func testPreRejectInsufficientBalance(t *testing.T) {
// 	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": num.NewUint(50)})
// 	policy.Reset(types.Epoch{Seq: 0})
// 	tx := &testTx{party: "party1", proposal: "proposal1"}
// 	require.Equal(t, spam.ErrInsufficientTokensForVoting, policy.PreBlockAccept(tx))
// }

// func testPreRejectTooManyVotesPerProposal(t *testing.T) {
// 	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
// 	// epoch 0 block 0
// 	policy.Reset(types.Epoch{Seq: 0})

// 	// vote 5 times for each proposal all pre accepted, 3 for each post accepted
// 	for i := 0; i < 2; i++ {
// 		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
// 		// pre accepted
// 		for i := 0; i < 5; i++ {
// 			require.NoError(t, policy.PreBlockAccept(tx))
// 		}

// 		// prepare block check
// 		for i := 0; i < 5; i++ {
// 			if i < 3 {
// 				require.NoError(t, policy.CheckBlockTx(tx))
// 			} else {
// 				require.Error(t, policy.CheckBlockTx(tx))
// 			}
// 		}
// 	}

// 	// end block 0
// 	policy.EndProcessBlock()

// 	// try to submit
// 	for i := 0; i < 2; i++ {
// 		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
// 		require.Equal(t, spam.ErrTooManyVotes.Error(), policy.PreBlockAccept(tx).Error())
// 	}

// 	tx := &testTx{party: "party1", proposal: "proposal3"}
// 	// pre accepted
// 	require.NoError(t, policy.PreBlockAccept(tx))

// 	// advance to next epoch to reset limits
// 	policy.Reset(types.Epoch{Seq: 0})
// 	for i := 0; i < 3; i++ {
// 		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
// 		require.NoError(t, policy.PreBlockAccept(tx))
// 	}
// }

// func testPreAccept(t *testing.T) {
// 	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
// 	// epoch 0 block 0
// 	policy.Reset(types.Epoch{Seq: 0})

// 	// vote 5 times for each proposal all pre accepted, 3 for each post accepted
// 	for i := 0; i < 2; i++ {
// 		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
// 		// pre accepted
// 		for i := 0; i < 5; i++ {
// 			require.Nil(t, policy.PreBlockAccept(tx))
// 		}
// 	}
// }

// func testPostAccept(t *testing.T) {
// 	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
// 	policy.Reset(types.Epoch{Seq: 0})

// 	policy.Reset(types.Epoch{Seq: 0})

// 	tx1 := &testTx{party: "party1", proposal: "proposal1"}
// 	tx2 := &testTx{party: "party1", proposal: "proposal1"}
// 	tx3 := &testTx{party: "party1", proposal: "proposal1"}
// 	tx4 := &testTx{party: "party1", proposal: "proposal1"}

// 	require.NoError(t, policy.CheckBlockTx(tx1))
// 	require.NoError(t, policy.CheckBlockTx(tx2))
// 	require.NoError(t, policy.CheckBlockTx(tx3))
// 	require.Error(t, policy.CheckBlockTx(tx4))

// 	// rollback the proposal
// 	policy.EndProcessBlock()

// 	// as the state has nothing expect pre block accept of all 4 txs
// 	require.NoError(t, policy.PreBlockAccept(tx1))
// 	require.NoError(t, policy.PreBlockAccept(tx2))
// 	require.NoError(t, policy.PreBlockAccept(tx3))
// 	require.NoError(t, policy.PreBlockAccept(tx4))

// 	// now a block is made with the first 3 txs
// 	require.NoError(t, policy.CheckBlockTx(tx1))
// 	require.NoError(t, policy.CheckBlockTx(tx2))
// 	require.NoError(t, policy.CheckBlockTx(tx3))

// 	// and the block is confirmed
// 	policy.EndProcessBlock()

// 	stats := policy.GetVoteSpamStats(tx1.party).GetStatistics()[0]
// 	require.Equal(t, uint64(3), stats.CountForEpoch)

// 	// now that there's been 3 proposals already, the 4th should be pre-rejected
// 	require.Error(t, policy.PreBlockAccept(tx4))

// 	// start a new epoch to reset counters
// 	policy.Reset(types.Epoch{Seq: 0})

// 	// check that the new proposal is pre-block accepted
// 	require.NoError(t, policy.PreBlockAccept(tx4))

// 	require.Equal(t, 0, len(policy.GetVoteSpamStats(tx1.party).GetStatistics()))
// }

// func testCountersUpdated(t *testing.T) {
// 	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
// 	policy.Reset(types.Epoch{Seq: 0})

// 	for i := 0; i < 2; i++ {
// 		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
// 		// post accepted
// 		for i := 0; i < 2; i++ {
// 			require.NoError(t, policy.CheckBlockTx(tx))
// 		}
// 	}
// 	policy.EndProcessBlock()
// 	for i := 0; i < 2; i++ {
// 		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
// 		// pre accepted
// 		for i := 0; i < 2; i++ {
// 			require.NoError(t, policy.PreBlockAccept(tx))
// 		}
// 	}
// 	for i := 0; i < 2; i++ {
// 		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
// 		// post accepted
// 		require.NoError(t, policy.CheckBlockTx(tx))

// 		// post rejected
// 		require.Error(t, spam.ErrTooManyVotes, policy.CheckBlockTx(tx))
// 	}
// }

// func testReset(t *testing.T) {
// 	// set state
// 	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
// 	policy.Reset(types.Epoch{Seq: 0})
// 	for i := 0; i < 2; i++ {
// 		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
// 		// pre accepted
// 		for i := 0; i < 6; i++ {
// 			require.NoError(t, policy.PreBlockAccept(tx))
// 		}
// 	}

// 	for i := 0; i < 2; i++ {
// 		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
// 		// post accepted
// 		for i := 0; i < 3; i++ {
// 			require.NoError(t, policy.CheckBlockTx(tx))
// 		}

// 		for i := 0; i < 3; i++ {
// 			require.Equal(t, spam.ErrTooManyVotes, policy.CheckBlockTx(tx))
// 		}
// 	}
// 	policy.EndProcessBlock()
// 	tx := &testTx{party: "party1", proposal: "proposal1"}
// 	require.Error(t, policy.PreBlockAccept(tx))

// 	policy.Reset(types.Epoch{Seq: 1})
// 	require.NoError(t, policy.PreBlockAccept(tx))
// }

// func testVoteEndBlockReset(t *testing.T) {
// 	// set state
// 	policy := getVotingSpamPolicy(map[string]*num.Uint{"party1": sufficientTokensForVoting})
// 	policy.Reset(types.Epoch{Seq: 0})

// 	// in each block we vote once
// 	var i uint64
// 	for ; i < 3; i++ {
// 		tx := &testTx{party: "party1", proposal: "proposal1"}
// 		require.NoError(t, policy.PreBlockAccept(tx))
// 		require.NoError(t, policy.CheckBlockTx(tx))
// 		policy.EndProcessBlock()
// 	}

// 	tx := &testTx{party: "party1", proposal: "proposal1"}
// 	require.Error(t, spam.ErrTooManyVotes, policy.PreBlockAccept(tx))

// 	bytes1, err := policy.Serialise()
// 	require.Nil(t, err)
// 	var votePayload snapshot.Payload
// 	proto.Unmarshal(bytes1, &votePayload)
// 	payload := types.PayloadFromProto(&votePayload)
// 	policy.Deserialise(payload)
// 	bytes2, err := policy.Serialise()
// 	require.Nil(t, err)
// 	require.True(t, bytes.Equal(bytes1, bytes2))

// 	tx2 := &testTx{party: "party1", proposal: "proposal2"}
// 	require.NoError(t, policy.CheckBlockTx(tx2))
// 	policy.EndPrepareBlock()

// 	// verify that changes made during prepare proposal are properly rolled back and not affecting the state
// 	bytes3, err := policy.Serialise()
// 	require.NoError(t, err)
// 	require.True(t, bytes.Equal(bytes3, bytes2))

// 	// now the block has been processed, verify that the state has changed
// 	require.NoError(t, policy.CheckBlockTx(tx2))
// 	policy.EndProcessBlock()
// 	bytes4, err := policy.Serialise()
// 	require.NoError(t, err)
// 	require.False(t, bytes.Equal(bytes4, bytes3))
// }
