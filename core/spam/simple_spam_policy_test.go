// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package spam_test

import (
	"strconv"
	"testing"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/spam"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/stretchr/testify/require"
)

var insufficientPropTokens, _ = num.UintFromString("50000000000000000000000", 10)

var sufficientPropTokens, _ = num.UintFromString("100000000000000000000000", 10)

func getCommandSpamPolicy(accounts map[string]*num.Uint) *spam.SimpleSpamPolicy {
	testAccounts := testAccounts{balances: accounts}
	logger := logging.NewTestLogger()
	policy := spam.NewSimpleSpamPolicy("simple", netparams.SpamProtectionMinTokensForProposal, netparams.SpamProtectionMaxProposals, logger, testAccounts)
	minTokensForProposal, _ := num.UintFromString("100000000000000000000000", 10)
	policy.UpdateUintParam(netparams.SpamProtectionMinTokensForProposal, minTokensForProposal)
	policy.UpdateIntParam(netparams.SpamProtectionMaxProposals, 3)
	return policy
}

func TestInsufficientPropTokens(t *testing.T) {
	policy := getCommandSpamPolicy(map[string]*num.Uint{"party1": insufficientPropTokens})
	// epoch 0 block 0
	policy.Reset(types.Epoch{Seq: 0})
	tx := &testTx{party: "party1", proposal: "proposal1"}
	err := policy.PreBlockAccept(tx)
	require.Equal(t, "party has insufficient associated governance tokens in their staking account to submit simple request", err.Error())
}

func TestCommandPreAccept(t *testing.T) {
	policy := getCommandSpamPolicy(map[string]*num.Uint{"party1": sufficientPropTokens})
	// epoch 0 block 0
	policy.Reset(types.Epoch{Seq: 0})

	// propose 5 times all pre accepted, 3 for each post accepted
	for i := 0; i < 2; i++ {
		tx := &testTx{party: "party1", proposal: "proposal" + strconv.Itoa(i+1)}
		// pre accepted
		for i := 0; i < 5; i++ {
			err := policy.PreBlockAccept(tx)
			require.NoError(t, err)
		}
	}
}

func TestEndPrepareBlock(t *testing.T) {
	policy := getCommandSpamPolicy(map[string]*num.Uint{"party1": sufficientPropTokens})
	policy.Reset(types.Epoch{Seq: 0})

	tx1 := &testTx{party: "party1", proposal: "proposal1"}
	tx2 := &testTx{party: "party1", proposal: "proposal2"}
	tx3 := &testTx{party: "party1", proposal: "proposal3"}

	// prepare a block
	require.NoError(t, policy.CheckBlockTx(tx1))
	require.NoError(t, policy.CheckBlockTx(tx2))
	require.NoError(t, policy.CheckBlockTx(tx3))

	// end the proposal preparation to rollback any block changes
	policy.RollbackProposal()

	s := policy.GetSpamStats(tx1.party)
	require.Equal(t, uint64(0), s.CountForEpoch)

	// assume block was proposed, now check from process proposal
	policy.UpdateTx(tx1)
	policy.UpdateTx(tx2)
	policy.UpdateTx(tx3)

	s = policy.GetSpamStats(tx1.party)
	require.Equal(t, uint64(3), s.CountForEpoch)
}

func TestCheckBlockTx(t *testing.T) {
	policy := getCommandSpamPolicy(map[string]*num.Uint{"party1": sufficientPropTokens})
	policy.Reset(types.Epoch{Seq: 0})

	tx1 := &testTx{party: "party1", proposal: "proposal1"}
	tx2 := &testTx{party: "party1", proposal: "proposal2"}
	tx3 := &testTx{party: "party1", proposal: "proposal3"}
	tx4 := &testTx{party: "party1", proposal: "proposal4"}

	require.NoError(t, policy.CheckBlockTx(tx1))
	require.NoError(t, policy.CheckBlockTx(tx2))
	require.NoError(t, policy.CheckBlockTx(tx3))
	require.Error(t, policy.CheckBlockTx(tx4))

	// rollback the proposal
	policy.RollbackProposal()

	// as the state has nothing expect pre block accept of all 4 txs
	require.NoError(t, policy.PreBlockAccept(tx1))
	require.NoError(t, policy.PreBlockAccept(tx2))
	require.NoError(t, policy.PreBlockAccept(tx3))
	require.NoError(t, policy.PreBlockAccept(tx4))

	// now a block is made with the first 3 txs
	policy.UpdateTx(tx1)
	policy.UpdateTx(tx2)
	policy.UpdateTx(tx3)

	stats := policy.GetSpamStats(tx1.party)
	require.Equal(t, uint64(3), stats.CountForEpoch)

	// now that there's been 3 proposals already, the 4th should be pre-rejected
	require.Error(t, policy.PreBlockAccept(tx4))

	// start a new epoch to reset counters
	policy.Reset(types.Epoch{Seq: 0})

	// check that the new proposal is pre-block accepted
	require.NoError(t, policy.PreBlockAccept(tx4))

	stats = policy.GetSpamStats(tx1.party)
	require.Equal(t, uint64(0), stats.CountForEpoch)
}
