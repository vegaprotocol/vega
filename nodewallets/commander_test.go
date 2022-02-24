package nodewallets_test

import (
	"context"
	"testing"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/vega/blockchain"
	vgtesting "code.vegaprotocol.io/vega/libs/testing"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/nodewallets/mocks"
	vgnw "code.vegaprotocol.io/vega/nodewallets/vega"
	"code.vegaprotocol.io/vega/txn"
	"github.com/stretchr/testify/require"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type testCommander struct {
	*nodewallets.Commander
	ctx    context.Context
	cfunc  context.CancelFunc
	ctrl   *gomock.Controller
	chain  *mocks.MockChain
	bstats *mocks.MockBlockchainStats
	wallet *vgnw.Wallet
}

func getTestCommander(t *testing.T) *testCommander {
	t.Helper()
	ctx, cfunc := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	chain := mocks.NewMockChain(ctrl)
	bstats := mocks.NewMockBlockchainStats(ctrl)

	vegaPaths, _ := vgtesting.NewVegaPaths()
	registryPass := vgrand.RandomStr(10)
	walletPass := vgrand.RandomStr(10)

	_, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletPass, false)
	require.NoError(t, err)
	wallet, err := nodewallets.GetVegaWallet(vegaPaths, registryPass)
	require.NoError(t, err)
	require.NotNil(t, wallet)

	cmd, err := nodewallets.NewCommander(
		nodewallets.NewDefaultConfig(), logging.NewTestLogger(), chain, wallet, bstats)
	require.NoError(t, err)

	return &testCommander{
		Commander: cmd,
		ctx:       ctx,
		cfunc:     cfunc,
		ctrl:      ctrl,
		chain:     chain,
		bstats:    bstats,
		wallet:    wallet,
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
	commander.chain.EXPECT().SubmitTransactionAsync(
		gomock.Any(), gomock.Any()).Times(1).Return(&tmctypes.ResultBroadcastTx{}, nil)

	ok := make(chan error)
	commander.Command(ctx, cmd, payload, func(err error) {
		ok <- err
	})
	assert.NoError(t, <-ok)
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
	commander.chain.EXPECT().SubmitTransactionAsync(
		gomock.Any(), gomock.Any()).Times(1).Return(&tmctypes.ResultBroadcastTx{}, errors.New("bad bad"))

	ok := make(chan error)
	commander.Command(ctx, cmd, payload, func(err error) {
		ok <- err
	})
	assert.Error(t, <-ok)
}

func (t *testCommander) Finish() {
	t.cfunc()
	t.ctrl.Finish()
}
