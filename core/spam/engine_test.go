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
	"bytes"
	"context"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/spam"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/stretchr/testify/require"
)

func (t *testTx) Command() txn.Command {
	return t.command
}

func (t *testTx) Unmarshal(cmd interface{}) error {
	cmd.(*commandspb.VoteSubmission).ProposalId = t.proposal
	return nil
}

func (t *testTx) PubKey() []byte {
	return nil
}

func (t *testTx) PubKeyHex() string {
	return ""
}

func (t *testTx) Party() string {
	return t.party
}

func (t *testTx) Hash() []byte {
	return nil
}

func (t *testTx) Signature() []byte {
	return nil
}

func (t *testTx) Validate() error {
	return nil
}

func (t *testTx) BlockHeight() uint64 {
	return 0
}

func (t *testTx) GetCmd() interface{} {
	return nil
}

func TestEngine(t *testing.T) {
	t.Run("pre block goes is handled by the appropriate spam policy", testPreBlockAccept)
	t.Run("post block goes is handled by the appropriate spam policy", testPostBlockAccept)
	t.Run("end prepare and end process is applied to all policies", testEndOfBlock)
	t.Run("reset is applied to all policies", testEngineReset)
}

func testEngineReset(t *testing.T) {
	testEngine := getEngine(t, map[string]*num.Uint{"party1": sufficientPropTokens})
	engine := testEngine.engine
	engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 0})

	tx1 := &testTx{party: "party1", proposal: "proposal1", command: txn.ProposeCommand}
	tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}

	// pre accept
	for i := 0; i < 3; i++ {
		require.NoError(t, engine.PreBlockAccept(tx1))
		require.NoError(t, engine.PreBlockAccept(tx2))
	}

	// post accept
	for i := 0; i < 3; i++ {
		err := engine.CheckBlockTx(tx1)
		require.NoError(t, err)
		err = engine.CheckBlockTx(tx2)
		require.NoError(t, err)
	}
	// move to next block, we've voted/proposed everything already so shouldn't be allowed to make more
	engine.EndPrepareProposal()
	engine.ProcessProposal([]abci.Tx{tx1, tx1, tx1, tx2, tx2, tx2})
	engine.BeginBlock([]abci.Tx{tx1, tx1, tx1, tx2, tx2, tx2})

	proposalState, _, err := engine.GetState("proposal")
	require.Nil(t, err)
	voteState, _, err := engine.GetState((&types.PayloadVoteSpamPolicy{}).Key())
	require.Nil(t, err)

	keys := engine.Keys()
	snap := make(map[string][]byte, len(keys))
	for _, k := range keys {
		data, _, err := engine.GetState(k)
		require.NoError(t, err)
		snap[k] = data
	}

	snapEngine := getEngine(t, map[string]*num.Uint{"party1": sufficientPropTokens})
	for _, bytes := range snap {
		var p snapshot.Payload
		proto.Unmarshal(bytes, &p)
		payload := types.PayloadFromProto(&p)
		snapEngine.engine.LoadState(context.Background(), payload)
	}

	// restore the epoch we were on
	snapEngine.engine.OnEpochRestore(context.Background(), types.Epoch{Seq: 0})

	proposalState2, _, err := snapEngine.engine.GetState("proposal")
	require.Nil(t, err)
	require.True(t, bytes.Equal(proposalState, proposalState2))

	voteState2, _, err := snapEngine.engine.GetState((&types.PayloadVoteSpamPolicy{}).Key())
	require.Nil(t, err)
	require.True(t, bytes.Equal(voteState, voteState2))

	require.Error(t, errors.New("party has already submitted the maximum number of proposal requests per epoch"), snapEngine.engine.PreBlockAccept(tx1))
	require.Equal(t, spam.ErrTooManyVotes, snapEngine.engine.PreBlockAccept(tx2))

	// Notify an epoch event for the *same* epoch and a reset should not happen
	snapEngine.engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 0})
	proposalStateNoReset, _, err := snapEngine.engine.GetState("proposal")
	require.Nil(t, err)
	require.True(t, bytes.Equal(proposalStateNoReset, proposalState2))

	// move to next epoch
	snapEngine.engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 1})

	// expect to be able to submit 3 more votes/proposals successfully
	for i := 0; i < 3; i++ {
		require.NoError(t, snapEngine.engine.PreBlockAccept(tx1))
		require.NoError(t, snapEngine.engine.PreBlockAccept(tx2))
	}

	proposalState3, _, err := snapEngine.engine.GetState("proposal")
	require.Nil(t, err)
	require.False(t, bytes.Equal(proposalState3, proposalState2))

	voteState3, _, err := snapEngine.engine.GetState((&types.PayloadVoteSpamPolicy{}).Key())
	require.Nil(t, err)
	require.False(t, bytes.Equal(voteState3, voteState2))
}

func testPreBlockAccept(t *testing.T) {
	testEngine := getEngine(t, map[string]*num.Uint{"party1": sufficientPropTokens})
	engine := testEngine.engine
	engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 0})

	tx1 := &testTx{party: "party1", proposal: "proposal1", command: txn.ProposeCommand}
	require.NoError(t, engine.PreBlockAccept(tx1))

	tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}
	require.NoError(t, engine.PreBlockAccept(tx2))

	tx1 = &testTx{party: "party2", proposal: "proposal1", command: txn.ProposeCommand}
	require.Equal(t, errors.New("party has insufficient associated governance tokens in their staking account to submit proposal request"), engine.PreBlockAccept(tx1))

	tx2 = &testTx{party: "party2", proposal: "proposal1", command: txn.VoteCommand}
	require.Equal(t, spam.ErrInsufficientTokensForVoting, engine.PreBlockAccept(tx2))
}

func testPostBlockAccept(t *testing.T) {
	testEngine := getEngine(t, map[string]*num.Uint{"party1": sufficientPropTokens})
	engine := testEngine.engine

	engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 0})

	tx1 := &testTx{party: "party1", proposal: "proposal1", command: txn.ProposeCommand}
	tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}
	for i := 0; i < 3; i++ {
		err := engine.CheckBlockTx(tx1)
		require.NoError(t, err)

		err = engine.CheckBlockTx(tx2)
		require.NoError(t, err)
	}
	engine.EndPrepareProposal()
	engine.ProcessProposal([]abci.Tx{tx1, tx1, tx1, tx2, tx2, tx2})
	engine.BeginBlock([]abci.Tx{tx1, tx1, tx1, tx2, tx2, tx2})

	tx1 = &testTx{party: "party1", proposal: "proposal1", command: txn.ProposeCommand}
	require.Error(t, errors.New("party has already submitted the maximum number of proposal requests per epoch"), engine.CheckBlockTx(tx1))

	tx2 = &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}
	require.Error(t, spam.ErrTooManyVotes, engine.CheckBlockTx(tx2))
}

func testEndOfBlock(t *testing.T) {
	testEngine := getEngine(t, map[string]*num.Uint{"party1": sufficientPropTokens})
	engine := testEngine.engine

	engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 0})

	tx1 := &testTx{party: "party1", proposal: "proposal1", command: txn.ProposeCommand}
	tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}

	for i := 0; i < 3; i++ {
		require.NoError(t, engine.CheckBlockTx(tx1))
		require.NoError(t, engine.CheckBlockTx(tx2))
	}
	engine.EndPrepareProposal()
	engine.ProcessProposal([]abci.Tx{tx1, tx1, tx1, tx2, tx2, tx2})
	engine.BeginBlock([]abci.Tx{tx1, tx1, tx1, tx2, tx2, tx2})

	for i := 0; i < 3; i++ {
		tx1 := &testTx{party: "party1", proposal: "proposal1", command: txn.ProposeCommand}
		require.Error(t, engine.CheckBlockTx(tx1))

		tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}
		require.Error(t, engine.CheckBlockTx(tx2))
	}
	// proposal is rejected
	engine.EndPrepareProposal()

	require.Error(t, errors.New("party has already submitted the maximum number of proposal requests per epoch"), engine.PreBlockAccept(tx1))
	require.Equal(t, spam.ErrTooManyVotes, engine.PreBlockAccept(tx2))
}

type testEngine struct {
	engine      *spam.Engine
	epochEngine *TestEpochEngine
	accounts    *testAccounts
}

type testAccounts struct {
	balances map[string]*num.Uint
}

func (t testAccounts) GetAvailableBalance(party string) (*num.Uint, error) {
	balance, ok := t.balances[party]
	if !ok {
		return nil, errors.New("no balance for party")
	}
	return balance, nil
}

func getEngine(t *testing.T, balances map[string]*num.Uint) *testEngine {
	t.Helper()
	conf := spam.NewDefaultConfig()
	logger := logging.NewTestLogger()
	epochEngine := &TestEpochEngine{
		callbacks: []func(context.Context, types.Epoch){},
		restore:   []func(context.Context, types.Epoch){},
	}
	accounts := &testAccounts{balances: balances}

	engine := spam.New(logger, conf, epochEngine, accounts)

	minTokensForVoting, _ := num.DecimalFromString("100000000000000000000")
	minTokensForProposal, _ := num.DecimalFromString("100000000000000000000000")
	engine.OnMaxProposalsChanged(context.Background(), 3)
	engine.OnMaxVotesChanged(context.Background(), 3)
	engine.OnMinTokensForVotingChanged(context.Background(), minTokensForVoting)
	engine.OnMinTokensForProposalChanged(context.Background(), minTokensForProposal)

	return &testEngine{
		engine:      engine,
		epochEngine: epochEngine,
		accounts:    accounts,
	}
}

type TestEpochEngine struct {
	callbacks []func(context.Context, types.Epoch)
	restore   []func(context.Context, types.Epoch)
}

func (e *TestEpochEngine) NotifyOnEpoch(f func(context.Context, types.Epoch), r func(context.Context, types.Epoch)) {
	e.callbacks = append(e.callbacks, f)
	e.restore = append(e.restore, r)
}
