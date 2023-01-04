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

package snapshot_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/snapshot/mocks"
	"code.vegaprotocol.io/vega/core/types"
	tmocks "code.vegaprotocol.io/vega/core/types/mocks"
	vegactx "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"

	"github.com/cosmos/iavl"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	db "github.com/tendermint/tm-db"
)

type tstEngine struct {
	*snapshot.Engine
	ctx   context.Context
	cfunc context.CancelFunc
	ctrl  *gomock.Controller
	time  *mocks.MockTimeService
	stats *mocks.MockStatsService
}

func getTestEngine(t *testing.T) *tstEngine {
	t.Helper()
	ctx, cfunc := context.WithCancel(context.Background())
	ctx = vegactx.WithChainID(ctx, "chain-id")
	ctrl := gomock.NewController(t)
	time := mocks.NewMockTimeService(ctrl)
	stats := mocks.NewMockStatsService(ctrl)
	eng, err := snapshot.New(context.Background(), nil, snapshot.NewTestConfig(), logging.NewTestLogger(), time, stats)
	require.NoError(t, err)

	if err := eng.ClearAndInitialise(); err != nil {
		t.Fatalf("couldn't clear and initialise snapshot engine for tests: %v", err)
	}
	ctx = vegactx.WithTraceID(vegactx.WithBlockHeight(ctx, 1), "0xDEADBEEF")
	return &tstEngine{
		ctx:    ctx,
		cfunc:  cfunc,
		Engine: eng,
		ctrl:   ctrl,
		time:   time,
		stats:  stats,
	}
}

// returns an avl tree populated with some payloads.
func getPopulatedTree(t *testing.T) *iavl.MutableTree {
	t.Helper()
	testPayloads := []types.Payload{
		{
			Data: &types.PayloadAppState{
				AppState: &types.AppState{
					Height: 64,
				},
			},
		},
		{
			Data: &types.PayloadGovernanceActive{
				GovernanceActive: &types.GovernanceActive{},
			},
		},
		{
			Data: &types.PayloadGovernanceEnacted{
				GovernanceEnacted: &types.GovernanceEnacted{},
			},
		},
		{
			Data: &types.PayloadDelegationActive{
				DelegationActive: &types.DelegationActive{},
			},
		},
		{
			Data: &types.PayloadEpoch{
				EpochState: &types.EpochState{
					Seq:                  7,
					ReadyToStartNewEpoch: true,
				},
			},
		},
	}

	tree, err := iavl.NewMutableTree(db.NewMemDB(), 0, false)
	tree.Load()
	require.NoError(t, err)

	for _, p := range testPayloads {
		v, _ := proto.Marshal(p.IntoProto())
		tree.Set([]byte(p.GetTreeKey()), v)
	}

	// Save it
	tree.SaveVersion()
	return tree
}

// basic engine functionality tests.
func TestEngine(t *testing.T) {
	t.Run("Adding a provider calls what we expect on the state provider", testAddProviders)
	t.Run("Adding provider with duplicate key in same namespace: first come, first serve", testAddProvidersDuplicateKeys)
	t.Run("Create a snapshot successfully", testTakeSnapshot)
	t.Run("Rejecting a snapshot should return a Snapshot Retry Limit error if rejected too many times", testRejectSnapshot)
	t.Run("Removing multiple keys within a single namespace", testRemovingMultipleKeysSingleNamespace)
	t.Run("Closing the engine doesn't panic when not initialised", testClosingEngineDoesNotPanicWhenNotInitialised)
}

func TestRestore(t *testing.T) {
	t.Run("Restoring a snapshot from chain works as expected", testReloadSnapshot)
	t.Run("Restoring a snapshot calls the post-restore callback if available", testReloadRestore)
}

func TestTreeToSnapshot(t *testing.T) {
	t.Run("A tree can be exported serialised and then imported", testTreeExportImport)
}

func testTreeExportImport(t *testing.T) {
	// get a avl tree with some payloads in it
	tree := getPopulatedTree(t)

	// export the tree into snapshot data
	snap, err := types.SnapshotFromTree(tree.ImmutableTree)
	require.NoError(t, err)
	hash, _ := tree.Hash()
	require.Equal(t, snap.Hash, hash)
	require.Equal(t, snap.Meta.Version, tree.Version())
	require.Equal(t, len(snap.Nodes), int(tree.Size()))
	// We expect more nodehashes than nodes since nodes only contain the leaf nodes
	// with payloads whereas nodehashes contain the payload-less subtree-roots
	require.Greater(t, len(snap.Meta.NodeHashes), len(snap.Nodes))

	// Note IRL it would be now that snapshot is serialised and sent
	// via TM to the node restoring from a snapshot

	// Make a new tree waiting to import the snapshot
	importedTree, err := iavl.NewMutableTree(db.NewMemDB(), 0, false)
	importedTree.Load()
	require.NoError(t, err)

	// import the snapshot data into a new avl tree
	err = snap.TreeFromSnapshot(importedTree)
	require.NoError(t, err)

	// The new tree should be identical to the previous
	treeHash, _ := tree.Hash()
	importedTreeHash, _ := importedTree.Hash()

	assert.Equal(t, treeHash, importedTreeHash)
	assert.Equal(t, tree.Size(), importedTree.Size())
	assert.Equal(t, tree.Height(), importedTree.Height())
	assert.Equal(t, tree.Version(), importedTree.Version())
}

func testAddProviders(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	prov := engine.getNewProviderMock()
	prov.EXPECT().Keys().Times(1).Return([]string{"all"})
	prov.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	engine.AddProviders(prov)
}

func testAddProvidersDuplicateKeys(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	keys1 := []string{
		"foo",
		"bar",
	}
	keys2 := []string{
		keys1[0],
		"bar2",
	}
	prov1 := engine.getNewProviderMock()
	prov2 := engine.getNewProviderMock()
	prov1.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	prov2.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	prov1.EXPECT().Keys().Times(1).Return(keys1)
	prov2.EXPECT().Keys().Times(1).Return(keys2)
	// first come-first serve
	engine.AddProviders(prov1, prov2)
	hash1 := [][]byte{
		[]byte("foo"),
		[]byte("bar"),
	}
	data1 := hash1
	hash2 := [][]byte{
		[]byte("bar2"),
	}
	data2 := hash2
	prov1.EXPECT().Stopped().Times(2).Return(false)
	for i, k := range keys1 {
		prov1.EXPECT().GetState(k).Times(1).Return(data1[i], nil, nil)
	}
	// duplicate key is skipped
	prov2.EXPECT().Stopped().Times(1).Return(false)
	prov2.EXPECT().GetState(keys2[1]).Times(1).Return(data2[0], nil, nil)

	engine.time.EXPECT().GetTimeNow().Times(1).Return(time.Now())
	_, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
}

func testTakeSnapshot(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	keys := []string{
		"all",
	}
	prov := engine.getNewProviderMock()
	prov.EXPECT().Keys().Times(1).Return(keys)
	prov.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	engine.AddProviders(prov)

	// now take a snapshot
	now := time.Now()
	engine.time.EXPECT().GetTimeNow().Times(1).Return(now)
	state := map[string]*types.Payload{
		keys[0]: {
			Data: &types.PayloadCheckpoint{
				Checkpoint: &types.CPState{
					NextCp: now.Add(time.Hour).Unix(),
				},
			},
		},
	}
	// set up provider to return state
	for _, k := range keys {
		pl := state[k]
		data, err := proto.Marshal(pl.IntoProto())
		require.NoError(t, err)
		prov.EXPECT().GetState(k).Times(1).Return(data, nil, nil)
	}
	prov.EXPECT().Stopped().Times(len(keys)).Return(false)

	// take the snapshot knowing state has changed:
	// we need the ctx that goes with the mock, because that has block height and hash set
	_, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
}

func testReloadSnapshot(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	keys := []string{
		"all",
	}
	prov := engine.getNewProviderMock()
	prov.EXPECT().Keys().Times(1).Return(keys)
	prov.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	engine.AddProviders(prov)

	now := time.Now()
	engine.time.EXPECT().GetTimeNow().Times(1).Return(now)
	state := map[string]*types.Payload{
		keys[0]: {
			Data: &types.PayloadCheckpoint{
				Checkpoint: &types.CPState{
					NextCp: now.Add(time.Hour).Unix(),
				},
			},
		},
	}
	for _, k := range keys {
		pl := state[k]
		data, err := proto.Marshal(pl.IntoProto())
		require.NoError(t, err)
		prov.EXPECT().GetState(k).Times(1).Return(data, nil, nil)
	}
	prov.EXPECT().Stopped().Times(len(keys)).Return(false)
	hash, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// get the snapshot list
	snaps, err := engine.List()
	require.NoError(t, err)
	require.NotEmpty(t, snaps)
	require.Equal(t, 1, len(snaps))

	// create a new engine which will restore the snapshot
	eng2 := getTestEngine(t)
	defer eng2.Finish()
	p2 := eng2.getNewProviderMock()
	p2.EXPECT().Keys().Times(1).Return(keys)
	p2.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	eng2.AddProviders(p2)

	// calls we expect to see when reloading
	eng2.time.EXPECT().SetTimeNow(gomock.Any(), gomock.Any()).Times(1).Do(func(_ context.Context, newT time.Time) {
		require.Equal(t, newT.Unix(), now.Unix())
	})
	eng2.stats.EXPECT().SetHeight(uint64(1)).Times(1)
	// ensure we're passing the right state
	p2.EXPECT().LoadState(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil).Do(func(_ context.Context, pl *types.Payload) {
		require.EqualValues(t, pl.Data, state[keys[0]].Data)
	})

	// start receiving the snapshot
	snap := snaps[0]
	require.NoError(t, eng2.ReceiveSnapshot(snap))
	ready := false
	for i := uint32(0); i < snap.Chunks; i++ {
		chunk, err := engine.LoadSnapshotChunk(snap.Height, uint32(snap.Format), i)
		require.NoError(t, err)
		ready, err = eng2.ApplySnapshotChunk(chunk)
		require.NoError(t, err)
	}
	require.True(t, ready)

	// OK, our snapshot is ready to load
	require.NoError(t, eng2.ApplySnapshot(eng2.ctx))

	loaded, err := eng2.CheckLoaded()
	require.NoError(t, err)
	require.True(t, loaded)
}

func testReloadRestore(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	keys := []string{
		"all",
	}
	prov := engine.getNewProviderMock()
	prov.EXPECT().Keys().Times(1).Return(keys)
	prov.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	engine.AddProviders(prov)

	now := time.Now()
	engine.time.EXPECT().GetTimeNow().Times(1).Return(now)
	state := map[string]*types.Payload{
		keys[0]: {
			Data: &types.PayloadCheckpoint{
				Checkpoint: &types.CPState{
					NextCp: now.Add(time.Hour).Unix(),
				},
			},
		},
	}
	for _, k := range keys {
		pl := state[k]
		data, err := proto.Marshal(pl.IntoProto())
		require.NoError(t, err)
		prov.EXPECT().GetState(k).Times(1).Return(data, nil, nil)
	}
	prov.EXPECT().Stopped().Times(len(keys)).Return(false)
	hash, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// get the snapshot list
	snaps, err := engine.List()
	require.NoError(t, err)
	require.NotEmpty(t, snaps)
	require.Equal(t, 1, len(snaps))

	// create a new engine which will restore the snapshot
	eng2 := getTestEngine(t)
	defer eng2.Finish()
	p2 := eng2.getRestoreMock()
	p2.EXPECT().Keys().Times(1).Return(keys)
	p2.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	eng2.AddProviders(p2)

	// calls we expect to see when reloading
	eng2.time.EXPECT().SetTimeNow(gomock.Any(), gomock.Any()).Times(1).Do(func(_ context.Context, newT time.Time) {
		require.Equal(t, newT.Unix(), now.Unix())
	})
	eng2.stats.EXPECT().SetHeight(uint64(1)).Times(1)
	// ensure we're passing the right state
	p2.EXPECT().LoadState(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil).Do(func(_ context.Context, pl *types.Payload) {
		require.EqualValues(t, pl.Data, state[keys[0]].Data)
	})

	// start receiving the snapshot
	snap := snaps[0]
	require.NoError(t, eng2.ReceiveSnapshot(snap))
	ready := false
	for i := uint32(0); i < snap.Chunks; i++ {
		chunk, err := engine.LoadSnapshotChunk(snap.Height, uint32(snap.Format), i)
		require.NoError(t, err)
		ready, err = eng2.ApplySnapshotChunk(chunk)
		require.NoError(t, err)
	}
	require.True(t, ready)
	p2.EXPECT().OnStateLoaded(gomock.Any()).Times(1).Return(nil)

	// OK, our snapshot is ready to load
	require.NoError(t, eng2.ApplySnapshot(eng2.ctx))
	loaded, err := eng2.CheckLoaded()
	require.NoError(t, err)
	require.True(t, loaded)
}

func (t *tstEngine) getNewProviderMock() *tmocks.MockStateProvider {
	return tmocks.NewMockStateProvider(t.ctrl)
}

func (t *tstEngine) getRestoreMock() *tmocks.MockPostRestore {
	return tmocks.NewMockPostRestore(t.ctrl)
}

func (t *tstEngine) Finish() {
	t.cfunc()
	t.ctrl.Finish()
}

func testRejectSnapshot(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()

	for i := 0; i < engine.RetryLimit; i++ {
		err := engine.RejectSnapshot()
		assert.ErrorIs(t, types.ErrUnknownSnapshot, err)
	}

	err := engine.RejectSnapshot()
	assert.ErrorIs(t, types.ErrSnapshotRetryLimit, err)
}

func testRemovingMultipleKeysSingleNamespace(t *testing.T) {
	someState := []byte("hello-i-am-state")
	engine := getTestEngine(t)
	defer engine.Finish()

	engine.time.EXPECT().GetTimeNow().AnyTimes().Return(time.Now())

	// first provider
	prov := engine.getNewProviderMock()
	prov.EXPECT().Keys().AnyTimes().Return([]string{"key1"})
	prov.EXPECT().Namespace().AnyTimes().Return(types.PositionsSnapshot)
	prov.EXPECT().Stopped().Times(1).Return(false)
	prov.EXPECT().GetState(gomock.Any()).AnyTimes().Return(someState, nil, nil)
	engine.AddProviders(prov)

	// second provider in the same namespace but with a different key i.e two positions engines, in the same namespace but keyed to difference markets
	prov2 := engine.getNewProviderMock()
	prov2.EXPECT().Keys().AnyTimes().Return([]string{"key2"})
	prov2.EXPECT().Namespace().AnyTimes().Return(types.PositionsSnapshot)
	prov2.EXPECT().GetState(gomock.Any()).AnyTimes().Return(someState, nil, nil)
	prov2.EXPECT().Stopped().Times(1).Return(false)
	engine.AddProviders(prov2)

	// initial snapshot
	b1, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
	require.NotNil(t, b1)

	prov.EXPECT().Stopped().Times(1).Return(false)
	prov2.EXPECT().Stopped().Times(1).Return(false)

	// call again to confirm no state changes
	b2, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
	require.NotNil(t, b2)

	// Now the only change is we signal remove of a single provider, check the snapshot changes
	prov.EXPECT().Stopped().Times(1).Return(true)
	prov2.EXPECT().Stopped().Times(1).Return(false)
	b3, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
	require.NotNil(t, b3)
	require.NotEqual(t, b1, b3)

	// remove the second provider
	prov2.EXPECT().Stopped().Times(1).Return(true)
	b4, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
	require.NotNil(t, b4)
	require.NotEqual(t, b3, b4)
}

func testClosingEngineDoesNotPanicWhenNotInitialised(t *testing.T) {
	// given
	ctrl := gomock.NewController(t)
	timeSvc := mocks.NewMockTimeService(ctrl)
	stats := mocks.NewMockStatsService(ctrl)
	config := snapshot.NewTestConfig()
	logger := logging.NewTestLogger()

	// when
	engine, err := snapshot.New(context.Background(), nil, config, logger, timeSvc, stats)

	// then
	require.NoError(t, err)

	// when
	require.NotPanics(t, func() {
		require.NoError(t, engine.Close())
	})
}

func TestUpdateInterval(t *testing.T) {
	e := getTestEngine(t)
	require.NoError(t, e.OnSnapshotIntervalUpdate(context.Background(), 10))
	for i := 0; i < 9; i++ {
		b, error := e.Snapshot(context.Background())
		require.Nil(t, b)
		require.Nil(t, error)
	}
	e.time.EXPECT().GetTimeNow().Times(1).Return(time.Now())
	ctx := vegactx.WithChainID(context.Background(), "chain-id")
	ctx = vegactx.WithTraceID(vegactx.WithBlockHeight(ctx, 10), "0xDEADBEEF")
	b, error := e.Snapshot(ctx)
	require.NotNil(t, b)
	require.NoError(t, error)

	// interval is now 10 and current will have been set to 10, lets change the interval to 5
	// run 4 blocks, no snapshot should be taken
	require.NoError(t, e.OnSnapshotIntervalUpdate(context.Background(), 5))
	for i := 0; i < 4; i++ {
		b, error := e.Snapshot(context.Background())
		require.Nil(t, b)
		require.Nil(t, error)
	}
	e.time.EXPECT().GetTimeNow().Times(1).Return(time.Now())
	ctx = vegactx.WithChainID(context.Background(), "chain-id")
	ctx = vegactx.WithTraceID(vegactx.WithBlockHeight(ctx, 15), "0xDEADBEEF")
	b, error = e.Snapshot(ctx)
	require.NotNil(t, b)
	require.NoError(t, error)

	// interval is now 5 and let it run for a couple of blocks so current is 3 then we change the interval to 15 and expect current to be 12
	for i := 0; i < 2; i++ {
		b, error := e.Snapshot(context.Background())
		require.Nil(t, b)
		require.Nil(t, error)
	}
	require.NoError(t, e.OnSnapshotIntervalUpdate(context.Background(), 15))
	for i := 0; i < 12; i++ {
		b, error := e.Snapshot(context.Background())
		require.Nil(t, b)
		require.Nil(t, error)
	}
	e.time.EXPECT().GetTimeNow().Times(1).Return(time.Now())
	ctx = vegactx.WithChainID(context.Background(), "chain-id")
	ctx = vegactx.WithTraceID(vegactx.WithBlockHeight(ctx, 30), "0xDEADBEEF")
	b, error = e.Snapshot(ctx)
	require.NotNil(t, b)
	require.NoError(t, error)

	// finally lets change the interval to 1 and expect snapshot to be taken every block
	require.NoError(t, e.OnSnapshotIntervalUpdate(context.Background(), 1))
	for i := 0; i < 5; i++ {
		ctx = vegactx.WithChainID(context.Background(), "chain-id")
		ctx = vegactx.WithTraceID(vegactx.WithBlockHeight(ctx, 31+int64(i)), "0xDEADBEEF")
		e.time.EXPECT().GetTimeNow().Times(1).Return(time.Now())
		b, error := e.Snapshot(ctx)
		require.NotNil(t, b)
		require.NoError(t, error)
	}
}
