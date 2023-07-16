// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package nullchain_test

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/blockchain"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	vgrand "code.vegaprotocol.io/vega/libs/rand"

	"code.vegaprotocol.io/vega/core/blockchain/nullchain"
	"code.vegaprotocol.io/vega/core/blockchain/nullchain/mocks"
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	chainID     = "somechainid"
	genesisTime = "2021-11-25T10:22:23.03277423Z"
)

func TestNullChain(t *testing.T) {
	t.Run("test basics", testBasics)
	t.Run("test transactions create block", testTransactionsCreateBlock)
	t.Run("test timeforwarding creates blocks", testTimeForwardingCreatesBlocks)
	t.Run("test timeforwarding less than a block does nothing", testTimeForwardingLessThanABlockDoesNothing)
	t.Run("test timeforwarding request conversion", testTimeForwardingRequestConversion)
	t.Run("test replay from genesis", testReplayFromGenesis)
	t.Run("test replay with snapshot restore", testReplayWithSnapshotRestore)
	t.Run("test replay with a block that panics", testReplayPanicBlock)
}

func testBasics(t *testing.T) {
	ctx := context.Background()
	testChain := getTestNullChain(t, 2, time.Second)
	defer testChain.ctrl.Finish()

	// Check genesis time from genesis file has filtered through
	gt, _ := time.Parse(time.RFC3339Nano, genesisTime)
	getGT, err := testChain.chain.GetGenesisTime(ctx)
	assert.NoError(t, err)
	assert.Equal(t, gt, getGT)

	// Check chainID time from genesis file has filtered through
	id, err := testChain.chain.GetChainID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, chainID, id)
}

func testTransactionsCreateBlock(t *testing.T) {
	ctx := context.Background()
	testChain := getTestNullChain(t, 2, time.Second)
	defer testChain.ctrl.Finish()

	// Expected BeginBlock to be called with time shuffled forward by a block
	now, _ := testChain.chain.GetGenesisTime(ctx)
	// r := abci.RequestBeginBlock{Header: types.Header{Time: now, ChainID: chainID, Height: 1}}

	// One round of block processing calls
	testChain.app.EXPECT().FinalizeBlock(gomock.Any(), gomock.Any()).Do(func(_ context.Context, rr *abci.RequestFinalizeBlock) {
		require.Equal(t, now, rr.Time)
		require.Equal(t, int64(1), rr.Height)
	}).Times(1)
	testChain.app.EXPECT().Commit(gomock.Any(), gomock.Any()).Times(1)
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

	testChain.app.EXPECT().FinalizeBlock(gomock.Any(), gomock.Any()).Times(10).Do(func(_ context.Context, r *abci.RequestFinalizeBlock) {
		beginBlockTime = r.Time
		height = int(r.Height)
	})
	testChain.app.EXPECT().Commit(gomock.Any(), gomock.Any()).Times(10)

	testChain.chain.ForwardTime(step)
	assert.True(t, beginBlockTime.Equal(now.Add(18*time.Second))) // the start of the next block will take us to +20 seconds
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

func testTimeForwardingRequestConversion(t *testing.T) {
	now := time.Time{}

	// Bad input
	_, err := nullchain.RequestToDuration("nonsense", now)
	assert.Error(t, err)

	// Valid duration
	d, err := nullchain.RequestToDuration("1m10s", now)
	assert.NoError(t, err)
	assert.Equal(t, d, time.Minute+(10*time.Second))

	// backwards duration
	_, err = nullchain.RequestToDuration("-1m10s", now)
	assert.Error(t, err)
	// Valid datetime
	forward := now.Add(time.Minute)
	d, err = nullchain.RequestToDuration(forward.Format(time.RFC3339), now)
	assert.NoError(t, err)
	assert.Equal(t, time.Minute, d)

	// backwards in datetime
	forward = now.Add(-time.Hour)
	_, err = nullchain.RequestToDuration(forward.Format(time.RFC3339), now)
	assert.Error(t, err)
}

func testReplayWithSnapshotRestore(t *testing.T) {
	ctx := context.Background()
	rplFile := path.Join(t.TempDir(), "rfile")
	testChain := getTestUnstartedNullChain(t, 2, time.Second, &blockchain.ReplayConfig{Record: true, Replay: true, ReplayFile: rplFile})
	defer testChain.ctrl.Finish()

	generateChain(t, testChain, 15)
	testChain.chain.Stop()

	// pretend the protocol restores to block height 10
	restoredBlockTime := time.Unix(10000, 15)
	restoreBlockHeight := int64(10)
	testChain.app.EXPECT().Info(gomock.Any(), gomock.Any()).Times(1).Return(
		&abci.ResponseInfo{
			LastBlockHeight: restoreBlockHeight,
		}, nil,
	)

	// we'll replay 5 blocks
	testChain.app.EXPECT().FinalizeBlock(gomock.Any(), gomock.Any()).Times(5)
	testChain.app.EXPECT().Commit(gomock.Any(), gomock.Any()).Times(5)
	testChain.ts.EXPECT().GetTimeNow().Times(1).Return(restoredBlockTime)

	// start the nullchain from a snapshot
	err := testChain.chain.StartChain()
	require.NoError(t, err)

	// continue the chain and check we're at the right block height and stuff
	// the next begin block should be at block height 16 (restored to 10, replayed 5, starting the next)
	req := &abci.RequestFinalizeBlock{}
	testChain.app.EXPECT().FinalizeBlock(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
		req = r
		return &abci.ResponseFinalizeBlock{}, nil
	}).AnyTimes()
	testChain.app.EXPECT().Commit(gomock.Any(), gomock.Any()).Times(1)

	// fill the block
	testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))
	testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))

	genesis, err := testChain.chain.GetGenesisTime(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(16), req.Height)
	require.Equal(t, genesis.Add(15*time.Second).UnixNano(), req.Time.UnixNano())
}

func testReplayFromGenesis(t *testing.T) {
	// replay file
	rplFile := path.Join(t.TempDir(), "rfile")
	testChain := getTestUnstartedNullChain(t, 2, time.Second, &blockchain.ReplayConfig{Record: true, ReplayFile: rplFile})
	defer testChain.ctrl.Finish()

	generateChain(t, testChain, 15)
	testChain.chain.Stop()

	newChain := getTestUnstartedNullChain(t, 2, time.Second, &blockchain.ReplayConfig{Replay: true, ReplayFile: rplFile})
	defer newChain.ctrl.Finish()

	// protocol is starting from 0
	newChain.app.EXPECT().Info(gomock.Any(), gomock.Any()).Times(1).Return(
		&abci.ResponseInfo{
			LastBlockHeight: 0,
		}, nil,
	)

	// we'll replay 15 blocks
	newChain.app.EXPECT().InitChain(gomock.Any(), gomock.Any()).Times(1)
	newChain.app.EXPECT().FinalizeBlock(gomock.Any(), gomock.Any()).Times(15).Return(&abci.ResponseFinalizeBlock{}, nil)
	newChain.app.EXPECT().Commit(gomock.Any(), gomock.Any()).Times(15)

	// start the nullchain from genesis
	err := newChain.chain.StartChain()
	require.NoError(t, err)
}

func testReplayPanicBlock(t *testing.T) {
	ctx := context.Background()
	// replay file
	rplFile := path.Join(t.TempDir(), "rfile")
	testChain := getTestUnstartedNullChain(t, 2, time.Second, &blockchain.ReplayConfig{Record: true, ReplayFile: rplFile})
	defer testChain.ctrl.Finish()

	generateChain(t, testChain, 5)

	// send in a single transaction that works

	testChain.app.EXPECT().FinalizeBlock(gomock.Any(), gomock.Any()).Do(func(_ context.Context, rr *abci.RequestFinalizeBlock) {
		panic("ah panic processing transaction")
	}).Times(1)
	testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))

	require.Panics(t, func() {
		testChain.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))
	})

	// now stop the nullchain so we save the unfinished block
	testChain.chain.Stop()

	// replay the chain
	newChain := getTestUnstartedNullChain(t, 2, time.Second, &blockchain.ReplayConfig{Replay: true, ReplayFile: rplFile})
	defer newChain.ctrl.Finish()

	// protocol is starting from 0
	newChain.app.EXPECT().Info(gomock.Any(), gomock.Any()).Times(1).Return(
		&abci.ResponseInfo{
			LastBlockHeight: 0,
		}, nil,
	)

	// we'll replay 5 full blocks, and process the 6th "panic" block ready to start the 7th
	newChain.app.EXPECT().InitChain(gomock.Any(), gomock.Any()).Times(1)
	newChain.app.EXPECT().FinalizeBlock(gomock.Any(), gomock.Any()).Times(6)
	newChain.app.EXPECT().Commit(gomock.Any(), gomock.Any()).Times(6)

	// start the nullchain from genesis
	err := newChain.chain.StartChain()
	require.NoError(t, err)
}

type testNullBlockChain struct {
	chain *nullchain.NullBlockchain
	ctrl  *gomock.Controller
	app   *mocks.MockApplicationService
	ts    *mocks.MockTimeService
	cfg   blockchain.NullChainConfig
}

func getTestUnstartedNullChain(t *testing.T, txnPerBlock uint64, d time.Duration, rplCfg *blockchain.ReplayConfig) *testNullBlockChain {
	t.Helper()

	ctrl := gomock.NewController(t)

	app := mocks.NewMockApplicationService(ctrl)
	ts := mocks.NewMockTimeService(ctrl)

	cfg := blockchain.NewDefaultNullChainConfig()
	cfg.GenesisFile = newGenesisFile(t)
	cfg.BlockDuration = encoding.Duration{Duration: d}
	cfg.TransactionsPerBlock = txnPerBlock
	cfg.Level = encoding.LogLevel{Level: logging.DebugLevel}
	if rplCfg != nil {
		cfg.Replay = *rplCfg
	}

	n := nullchain.NewClient(logging.NewTestLogger(), cfg, ts)
	n.SetABCIApp(app)
	require.NotNil(t, n)

	return &testNullBlockChain{
		chain: n,
		ctrl:  ctrl,
		app:   app,
		ts:    ts,
		cfg:   cfg,
	}
}

func getTestNullChain(t *testing.T, txnPerBlock uint64, d time.Duration) *testNullBlockChain {
	t.Helper()
	nc := getTestUnstartedNullChain(t, txnPerBlock, d, nil)

	nc.app.EXPECT().Info(gomock.Any(), gomock.Any()).Times(1).Return(&abci.ResponseInfo{}, nil)
	nc.app.EXPECT().InitChain(gomock.Any(), gomock.Any()).Times(1)

	err := nc.chain.StartChain()
	require.NoError(t, err)

	return nc
}

func newGenesisFile(t *testing.T) string {
	t.Helper()
	data := fmt.Sprintf("{ \"genesis_time\": \"%s\",\"chain_id\": \"%s\", \"app_state\": { \"validators\": {}}}", genesisTime, chainID)

	filePath := filepath.Join(t.TempDir(), "genesis.json")
	if err := vgfs.WriteFile(filePath, []byte(data)); err != nil {
		t.Fatalf("couldn't write file: %v", err)
	}
	return filePath
}

// generateChain start the nullblockchain and generates random chain data until it reaches the given block height.
func generateChain(t *testing.T, nc *testNullBlockChain, height int) {
	t.Helper()

	nTxns := int(nc.cfg.TransactionsPerBlock) * height

	ctx := context.Background()
	nc.app.EXPECT().InitChain(gomock.Any(), gomock.Any()).Times(1)
	nc.app.EXPECT().FinalizeBlock(gomock.Any(), gomock.Any()).Times(height).Return(&abci.ResponseFinalizeBlock{}, nil)
	nc.app.EXPECT().Commit(gomock.Any(), gomock.Any()).Times(height)
	nc.app.EXPECT().Info(gomock.Any(), gomock.Any()).Times(1).Return(
		&abci.ResponseInfo{
			LastBlockHeight: 0,
		}, nil,
	)

	// start the nullchain
	err := nc.chain.StartChain()
	require.NoError(t, err)

	// send in enough transactions to fill the required blocks

	for i := 0; i < nTxns; i++ {
		nc.chain.SendTransactionSync(ctx, []byte(vgrand.RandomStr(5)))
	}
}
