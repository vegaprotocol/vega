package nodewallet_test

import (
	"context"
	"encoding/hex"
	"testing"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/nodewallet/mocks"
	"code.vegaprotocol.io/vega/txn"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type testCommander struct {
	*nodewallet.Commander
	ctx    context.Context
	cfunc  context.CancelFunc
	ctrl   *gomock.Controller
	chain  *mocks.MockChain
	bstats *mocks.MockBlockchainStats
	wal    nodewallet.Wallet
}

type stubWallet struct {
	key    []byte
	chain  string
	signed []byte
	err    error
	name   string
}

func getTestCommander(t *testing.T) *testCommander {
	ctx, cfunc := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	chain := mocks.NewMockChain(ctrl)
	bstats := mocks.NewMockBlockchainStats(ctrl)
	wal := &stubWallet{name: "some_name.1234", chain: string(nodewallet.Vega)}
	cmd, err := nodewallet.NewCommander(logging.NewTestLogger(), chain, wal, bstats)
	assert.NoError(t, err)
	return &testCommander{
		Commander: cmd,
		ctx:       ctx,
		cfunc:     cfunc,
		ctrl:      ctrl,
		chain:     chain,
		bstats:    bstats,
		wal:       wal,
	}
}

func TestCommand(t *testing.T) {
	t.Run("Signed command - success", testSignedCommandSuccess)
	t.Run("Signed command - failure", testSignedCommandFailure)
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
	payload := &commandspb.NodeVote{
		PubKey:    []byte("my-pub-key"),
		Reference: "test",
	}
	ctx := context.Background()

	commander.bstats.EXPECT().Height().Times(1).Return(uint64(42))
	commander.chain.EXPECT().SubmitTransactionV2(
		ctx, gomock.Any(), gomock.Any()).Times(1).Return(nil)

	ok := make(chan bool)
	commander.Command(ctx, cmd, payload, func(b bool) {
		ok <- b
	})
	assert.True(t, <-ok)
}

func testSignedCommandFailure(t *testing.T) {
	commander := getTestCommander(t)
	defer commander.Finish()

	cmd := txn.NodeVoteCommand
	payload := &commandspb.NodeVote{
		PubKey:    []byte("my-pub-key"),
		Reference: "test",
	}
	ctx := context.Background()

	commander.bstats.EXPECT().Height().Times(1).Return(uint64(42))
	commander.chain.EXPECT().SubmitTransactionV2(
		ctx, gomock.Any(), gomock.Any()).Times(1).Return(errors.New("bad bad"))

	ok := make(chan bool)
	commander.Command(ctx, cmd, payload, func(b bool) {
		ok <- b
	})
	assert.False(t, <-ok)
}

func (t *testCommander) Finish() {
	t.cfunc()
	t.ctrl.Finish()
}

func (s stubWallet) Name() string {
	return s.name
}

func (s stubWallet) Chain() string {
	return s.chain
}

func (s stubWallet) Algo() string {
	return "vega/ed25519"
}

func (s stubWallet) Version() uint32 {
	return 1
}

func (s stubWallet) PubKeyOrAddress() crypto.PublicKeyOrAddress {
	return crypto.NewPublicKeyOrAddress(hex.EncodeToString(s.key), s.key)
}

func (s stubWallet) Sign(_ []byte) ([]byte, error) {
	return s.signed, s.err
}
