package nullchain_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	vgrand "code.vegaprotocol.io/shared/libs/rand"

	"code.vegaprotocol.io/vega/blockchain/nullchain"
	"code.vegaprotocol.io/vega/blockchain/nullchain/mocks"
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/proto/tendermint/types"
)

func TestNullChain(t *testing.T) {
	t.Run("test transactions create block", testTransactionsCreateBlock)
	t.Run("test timeforwarding creates blocks", testTimeForwardingCreatesBlocks)
	t.Run("test timeforwarding less than a block does nothing", testTimeForwardingLessThanABlockDoesNothing)
}

func testTransactionsCreateBlock(t *testing.T) {
	ctx := context.Background()
	testChain := getTestNullChain(t, 2, time.Second)
	defer testChain.ctrl.Finish()

	// Expected BeginBlock to be called with time shuffled forward by a block
	now, _ := testChain.chain.GetGenesisTime(ctx)
	r := abci.RequestBeginBlock{Header: types.Header{Time: now.Add(time.Second)}}

	// One round of block processing calls
	testChain.app.EXPECT().BeginBlock(r).Times(1)
	testChain.app.EXPECT().EndBlock(gomock.Any()).Times(1)
	testChain.app.EXPECT().Commit().Times(1)

	// Expect only two of the three transactions to be delivered
	testChain.app.EXPECT().DeliverTx(gomock.Any()).Times(2)

	// Send in three transactions, two gets delivered in the block, one left over
	testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))
	testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))
	testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))

	count, err := testChain.chain.GetUnconfirmedTxCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func testTimeForwardingCreatesBlocks(t *testing.T) {
	ctx := context.Background()
	testChain := getTestNullChain(t, 10, 2*time.Second)
	defer testChain.ctrl.Finish()

	// each block is 2 seconds (we should snap back to 10 blocks)
	step := 21 * time.Second
	now, _ := testChain.chain.GetGenesisTime(ctx)
	beginBlockTime := now
	height := 0

	// Fill in a partial blocks worth of transactions
	testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))
	testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))
	testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))

	// One round of block processing calls
	testChain.app.EXPECT().BeginBlock(gomock.Any()).Times(10).Do(func(r abci.RequestBeginBlock) {
		beginBlockTime = r.Header.Time
	})
	testChain.app.EXPECT().EndBlock(gomock.Any()).Times(10).Do(func(r abci.RequestEndBlock) {
		height = int(r.Height)
	})
	testChain.app.EXPECT().Commit().Times(10)
	testChain.app.EXPECT().DeliverTx(gomock.Any()).Times(3)

	testChain.chain.ForwardTime(step)

	assert.True(t, beginBlockTime.Equal(now.Add(20*time.Second)))
	assert.Equal(t, 10, height)

	count, err := testChain.chain.GetUnconfirmedTxCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testTimeForwardingLessThanABlockDoesNothing(t *testing.T) {
	ctx := context.Background()
	testChain := getTestNullChain(t, 10, 2*time.Second)
	defer testChain.ctrl.Finish()

	// half a block duration
	step := time.Second

	// Fill in a partial blocks worth of transactions
	testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))
	testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))
	testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))

	testChain.chain.ForwardTime(step)

	count, err := testChain.chain.GetUnconfirmedTxCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

type testNullBlockChain struct {
	chain *nullchain.NullBlockchain
	ctrl  *gomock.Controller
	app   *mocks.MockApplicationService
}

func getTestNullChain(t *testing.T, txnPerBlock uint64, d time.Duration) *testNullBlockChain {
	t.Helper()

	ctrl := gomock.NewController(t)

	app := mocks.NewMockApplicationService(ctrl)

	cfg := nullchain.NewDefaultConfig()
	cfg.GenesisFile = newGenesisFile(t)
	cfg.BlockDuration = encoding.Duration{Duration: d}
	cfg.TransactionsPerBlock = txnPerBlock
	cfg.Level = encoding.LogLevel{Level: logging.DebugLevel}

	n := nullchain.NewClient(logging.NewTestLogger(), cfg, app)
	require.NotNil(t, n)

	app.EXPECT().InitChain(gomock.Any()).Times(1)
	app.EXPECT().BeginBlock(gomock.Any()).Times(1)

	err := n.Start()
	require.NoError(t, err)

	return &testNullBlockChain{
		chain: n,
		ctrl:  ctrl,
		app:   app,
	}
}

func newGenesisFile(t *testing.T) string {
	t.Helper()
	data := "{ \"appstate\": { \"stuff\": \"stuff\" }}"

	filePath := filepath.Join(t.TempDir(), "genesis.json")
	if err := vgfs.WriteFile(filePath, []byte(data)); err != nil {
		t.Fatalf("couldn't write file: %v", err)
	}
	return filePath
}
