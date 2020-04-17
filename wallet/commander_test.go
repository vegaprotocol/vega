package wallet_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/blockchain"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/wallet"
	"code.vegaprotocol.io/vega/wallet/crypto"
	"code.vegaprotocol.io/vega/wallet/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testCommander struct {
	*wallet.Commander
	ctx   context.Context
	cfunc context.CancelFunc
	ctrl  *gomock.Controller
	chain *mocks.MockChain
}

func getTestCommander(t *testing.T) *testCommander {
	ctx, cfunc := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	chain := mocks.NewMockChain(ctrl)
	return &testCommander{
		Commander: wallet.NewCommander(ctx, chain),
		ctx:       ctx,
		cfunc:     cfunc,
		ctrl:      ctrl,
		chain:     chain,
	}
}

func TestCommand(t *testing.T) {
	t.Run("Unsigned command - success", testUnsignedCommandSuccess)
	t.Run("Unsigned command - Fail", testUnsignedCommandFail)
	t.Run("Signed command - success", testSignedCommandSuccess)
	t.Run("Signed command - signature not required", testSignedUnsignedSuccess)
	t.Run("SetChain - dummy test for completeness", testSetChain)
}

func testSetChain(t *testing.T) {
	commander := getTestCommander(t)
	defer commander.Finish()
	commander.SetChain(&blockchain.Client{})
}

func testUnsignedCommandSuccess(t *testing.T) {
	commander := getTestCommander(t)
	defer commander.Finish()

	cmd := blockchain.RegisterNodeCommand
	payload := &types.NodeRegistration{}
	commander.chain.EXPECT().SubmitNodeRegistration(commander.ctx, gomock.Any()).Times(1)
	assert.NoError(t, commander.Command(nil, cmd, payload))
}

func testUnsignedCommandFail(t *testing.T) {
	commander := getTestCommander(t)
	defer commander.Finish()

	cmd := blockchain.NodeVoteCommand
	payload := &types.NodeVote{}
	err := commander.Command(nil, cmd, payload)
	assert.Error(t, err)
	assert.Equal(t, wallet.ErrCommandMustBeSigned, err)
}

func testSignedCommandSuccess(t *testing.T) {
	commander := getTestCommander(t)
	defer commander.Finish()

	cmd := blockchain.NodeVoteCommand
	payload := &types.NodeVote{}
	key := wallet.NewKeypair(crypto.NewEd25519(), []byte{1, 2, 3, 255}, []byte{253, 3, 2, 1})
	commander.chain.EXPECT().SubmitTransaction(commander.ctx, gomock.Any()).Times(1)
	assert.NoError(t, commander.Command(&key, cmd, payload))
}

func testSignedUnsignedSuccess(t *testing.T) {
	commander := getTestCommander(t)
	defer commander.Finish()

	// this command doesn't require a signature, let's sign it anyway
	cmd := blockchain.NotifyTraderAccountCommand
	payload := &types.NotifyTraderAccount{}
	key := wallet.NewKeypair(crypto.NewEd25519(), []byte{1, 2, 3, 255}, []byte{253, 3, 2, 1})
	commander.chain.EXPECT().SubmitTransaction(commander.ctx, gomock.Any()).Times(1)
	assert.NoError(t, commander.Command(&key, cmd, payload))
}

func (t *testCommander) Finish() {
	t.cfunc()
	t.ctrl.Finish()
}
