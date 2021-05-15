package nodewallet_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/nodewallet/mocks"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/txn"

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
	cmd, err := nodewallet.NewCommander(chain, wal)
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

	cmd := txn.NodeVoteCommand
	payload := &commandspb.NodeVote{}
	ctx := context.Background()

	commander.chain.EXPECT().SubmitTransaction(ctx, gomock.Any(), gomock.Any()).Times(1)
	assert.NoError(t, commander.Command(ctx, cmd, payload))
}

func (t *testCommander) Finish() {
	t.cfunc()
	t.ctrl.Finish()
}

func (s stubWallet) Chain() string {
	return s.chain
}

func (s stubWallet) Algo() string {
	return "vega/ed25519"
}

func (s stubWallet) Version() uint64 {
	return 1
}

func (s stubWallet) PubKeyOrAddress() []byte {
	return s.key
}

func (s stubWallet) Sign(_ []byte) ([]byte, error) {
	return s.signed, s.err
}
