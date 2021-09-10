package spam

import (
	"context"
	"testing"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
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

func TestEngine(t *testing.T) {
	t.Run("", testPreBlockAccept)
	t.Run("", testPostBlockAccept)
	t.Run("", testEndOfBlock)
	t.Run("", testReset)
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
	require.Equal(t, ErrInsufficientTokensForProposal, err)

	tx2 = &testTx{party: "party2", proposal: "proposal1", command: txn.VoteCommand}
	_, err = engine.PreBlockAccept(tx2)
	require.Equal(t, ErrInsufficientTokensForVoting, err)
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
	require.Equal(t, ErrTooManyProposals, err)

	tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}
	_, err = engine.PostBlockAccept(tx2)
	require.Equal(t, ErrTooManyVotes, err)
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
	require.Equal(t, ErrTooManyProposals, err)

	tx2 := &testTx{party: "party1", proposal: "proposal1", command: txn.VoteCommand}
	_, err = engine.PreBlockAccept(tx2)
	require.Equal(t, ErrTooManyVotes, err)
}

type testEngine struct {
	engine      *Engine
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
	conf := NewDefaultConfig()
	logger := logging.NewTestLogger()
	epochEngine := &TestEpochEngine{callbacks: []func(context.Context, types.Epoch){}}
	accounts := &testAccounts{balances: map[string]*num.Uint{}}

	engine := New(logger, conf, epochEngine, accounts)

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
