package nodewallet_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/nodewallet/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testCommander struct {
	*nodewallet.Commander
	ctx   context.Context
	cfunc context.CancelFunc
	ctrl  *gomock.Controller
	chain *mocks.MockChain
	wal   nodewallet.Wallet
}

type stubWallet struct {
	key    []byte
	chain  string
	signed []byte
	err    error
}

func getTestCommander(t *testing.T) *testCommander {
	ctx, cfunc := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	chain := mocks.NewMockChain(ctrl)
	wal := &stubWallet{chain: string(nodewallet.Vega)}
	cmd, err := nodewallet.NewCommander(ctx, chain, wal)
	assert.NoError(t, err)
	return &testCommander{
		Commander: cmd,
		ctx:       ctx,
		cfunc:     cfunc,
		ctrl:      ctrl,
		chain:     chain,
		wal:       wal,
	}
}

func TestCommand(t *testing.T) {
	t.Run("Signed command - success", testSignedCommandSuccess)
	t.Run("Signed command - signature not required", testSignedUnsignedSuccess)
	t.Run("SetChain - dummy test for completeness", testSetChain)
}

func testSetChain(t *testing.T) {
	commander := getTestCommander(t)
	defer commander.Finish()
	commander.SetChain(&blockchain.Client{})
}

func testSignedCommandSuccess(t *testing.T) {
	commander := getTestCommander(t)
	defer commander.Finish()

	cmd := blockchain.NodeVoteCommand
	payload := &types.NodeVote{}
	commander.chain.EXPECT().SubmitTransaction(commander.ctx, gomock.Any()).Times(1)
	assert.NoError(t, commander.Command(cmd, payload))
}

func testSignedUnsignedSuccess(t *testing.T) {
	commander := getTestCommander(t)
	defer commander.Finish()

	// this command doesn't require a signature, let's sign it anyway
	cmd := blockchain.NotifyTraderAccountCommand
	commander.chain.EXPECT().SubmitTransaction(commander.ctx, gomock.Any()).Times(1)
	payload := &types.NotifyTraderAccount{}
	assert.NoError(t, commander.Command(cmd, payload))
}

func (t *testCommander) Finish() {
	t.cfunc()
	t.ctrl.Finish()
}

func (s stubWallet) Chain() string {
	return s.chain
}

func (s stubWallet) PubKeyOrAddress() []byte {
	return s.key
}

func (s stubWallet) Sign(_ []byte) ([]byte, error) {
	return s.signed, s.err
}
