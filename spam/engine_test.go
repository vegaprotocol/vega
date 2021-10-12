package spam_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/spam"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/protobuf/proto"
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
	t.Run("end of block is applied to all policies", testEndOfBlock)
	t.Run("reset is applied to all policies", testEngineReset)
}

func testEngineReset(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	testEngine.accounts.balances = map[string]*num.Uint{"party1": sufficientPropTokens}
	engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 0})

	tx1 := &testTx{party: "party1", proposal: "proposal1", command: txn.ProposeCommand}
	tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}

	// pre accept
	for i := 0; i < 3; i++ {
		accept, _ := engine.PreBlockAccept(tx1)
		require.Equal(t, true, accept)

		accept, _ = engine.PreBlockAccept(tx2)
		require.Equal(t, true, accept)
	}

	// post accept
	for i := 0; i < 3; i++ {
		accept, _ := engine.PostBlockAccept(tx1)
		require.Equal(t, true, accept)
		accept, _ = engine.PostBlockAccept(tx2)
		require.Equal(t, true, accept)
	}

	// move to next block, we've voted/proposed everything already so shouldn't be allowed to make more
	engine.EndOfBlock(1)

	proposalHash, err := engine.GetHash("proposal")
	require.Nil(t, err)
	voteHash, err := engine.GetHash((&types.PayloadDelegationActive{}).Key())
	require.Nil(t, err)

	snap, err := engine.Snapshot()
	require.Nil(t, err)
	for _, bytes := range snap {
		var p snapshot.Payload
		proto.Unmarshal(bytes, &p)
		payload := types.PayloadFromProto(&p)
		engine.LoadState(context.Background(), payload)
	}

	proposalHash2, err := engine.GetHash("proposal")
	require.Nil(t, err)
	require.True(t, bytes.Equal(proposalHash, proposalHash2))

	voteHash2, err := engine.GetHash((&types.PayloadDelegationActive{}).Key())
	require.Nil(t, err)
	require.True(t, bytes.Equal(voteHash, voteHash2))

	accept, err := engine.PreBlockAccept(tx1)
	require.Equal(t, false, accept)
	require.Equal(t, errors.New("party has already proposed the maximum number of proposal requests per epoch"), err)

	accept, err = engine.PreBlockAccept(tx2)
	require.Equal(t, false, accept)
	require.Equal(t, spam.ErrTooManyVotes, err)

	// move to next epoch
	engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 1})

	// expect to be able to submit 3 more votes/proposals successfully
	for i := 0; i < 3; i++ {
		accept, _ := engine.PreBlockAccept(tx1)
		require.Equal(t, true, accept)

		accept, _ = engine.PreBlockAccept(tx2)
		require.Equal(t, true, accept)
	}

	proposalHash3, err := engine.GetHash("proposal")
	require.Nil(t, err)
	require.False(t, bytes.Equal(proposalHash3, proposalHash2))

	voteHash3, err := engine.GetHash((&types.PayloadDelegationActive{}).Key())
	require.Nil(t, err)
	require.False(t, bytes.Equal(voteHash3, voteHash2))

}

func testPreBlockAccept(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	testEngine.accounts.balances = map[string]*num.Uint{"party1": sufficientPropTokens}

	engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 0})

	tx1 := &testTx{party: "party1", proposal: "proposal1", command: txn.ProposeCommand}
	accept, _ := engine.PreBlockAccept(tx1)
	require.Equal(t, true, accept)

	tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}
	accept, _ = engine.PreBlockAccept(tx2)
	require.Equal(t, true, accept)

	tx1 = &testTx{party: "party2", proposal: "proposal1", command: txn.ProposeCommand}
	_, err := engine.PreBlockAccept(tx1)
	require.Equal(t, errors.New("party has insufficient tokens to submit proposal request in this epoch"), err)

	tx2 = &testTx{party: "party2", proposal: "proposal1", command: txn.VoteCommand}
	_, err = engine.PreBlockAccept(tx2)
	require.Equal(t, spam.ErrInsufficientTokensForVoting, err)
}

func testPostBlockAccept(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	testEngine.accounts.balances = map[string]*num.Uint{"party1": sufficientPropTokens}

	engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 0})

	for i := 0; i < 3; i++ {
		tx1 := &testTx{party: "party1", proposal: "proposal1", command: txn.ProposeCommand}
		accept, _ := engine.PostBlockAccept(tx1)
		require.Equal(t, true, accept)

		tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}
		accept, _ = engine.PostBlockAccept(tx2)
		require.Equal(t, true, accept)
	}

	tx1 := &testTx{party: "party1", proposal: "proposal1", command: txn.ProposeCommand}
	_, err := engine.PostBlockAccept(tx1)
	require.Equal(t, errors.New("party has already proposed the maximum number of proposal requests per epoch"), err)

	tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}
	_, err = engine.PostBlockAccept(tx2)
	require.Equal(t, spam.ErrTooManyVotes, err)
}

func testEndOfBlock(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	testEngine.accounts.balances = map[string]*num.Uint{"party1": sufficientPropTokens}

	engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 0})

	for i := 0; i < 3; i++ {
		tx1 := &testTx{party: "party1", proposal: "proposal1", command: txn.ProposeCommand}
		accept, _ := engine.PostBlockAccept(tx1)
		require.Equal(t, true, accept)

		tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}
		accept, _ = engine.PostBlockAccept(tx2)
		require.Equal(t, true, accept)
	}
	engine.EndOfBlock(1)
	tx1 := &testTx{party: "party1", proposal: "proposal1", command: txn.ProposeCommand}
	_, err := engine.PreBlockAccept(tx1)
	require.Equal(t, errors.New("party has already proposed the maximum number of proposal requests per epoch"), err)

	tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}
	_, err = engine.PreBlockAccept(tx2)
	require.Equal(t, spam.ErrTooManyVotes, err)
}

type testEngine struct {
	engine      *spam.Engine
	epochEngine *TestEpochEngine
	accounts    *testAccounts
}

type testAccounts struct {
	balances map[string]*num.Uint
}

func (t testAccounts) GetAllAvailableBalances() map[string]*num.Uint {
	return t.balances
}

func getEngine(t *testing.T) *testEngine {
	conf := spam.NewDefaultConfig()
	logger := logging.NewTestLogger()
	epochEngine := &TestEpochEngine{callbacks: []func(context.Context, types.Epoch){}}
	accounts := &testAccounts{balances: map[string]*num.Uint{}}

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
}

func (e *TestEpochEngine) NotifyOnEpoch(f func(context.Context, types.Epoch)) {
	e.callbacks = append(e.callbacks, f)
}
