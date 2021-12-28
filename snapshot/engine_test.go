package snapshot_test

import (
	"context"
	"testing"
	"time"

	vegactx "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/snapshot"
	"code.vegaprotocol.io/vega/snapshot/mocks"
	"code.vegaprotocol.io/vega/types"
	tmocks "code.vegaprotocol.io/vega/types/mocks"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

type tstEngine struct {
	*snapshot.Engine
	ctx   context.Context
	cfunc context.CancelFunc
	ctrl  *gomock.Controller
	time  *mocks.MockTimeService
}

func getTestEngine(t *testing.T) *tstEngine {
	t.Helper()
	ctx, cfunc := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	time := mocks.NewMockTimeService(ctrl)
	eng, err := snapshot.New(context.Background(), nil, snapshot.NewTestConfig(), logging.NewTestLogger(), time)
	require.NoError(t, err)
	ctx = vegactx.WithTraceID(vegactx.WithBlockHeight(ctx, 1), "0xDEADBEEF")
	return &tstEngine{
		ctx:    ctx,
		cfunc:  cfunc,
		Engine: eng,
		ctrl:   ctrl,
		time:   time,
	}
}

// basic engine functionality tests.
func TestEngine(t *testing.T) {
	t.Run("Adding a provider calls what we expect on the state provider", testAddProviders)
	t.Run("Adding provider with duplicate key in same namespace: first come, first serve", testAddProvidersDuplicateKeys)
	t.Run("Create a snapshot, if nothing changes, we don't get the data and the hash remains unchanged", testTakeSnapshot)
}

func TestRestore(t *testing.T) {
	t.Run("Restoring a snapshot from chain works as expected", testReloadSnapshot)
	t.Run("Restoring replay protectors replaces the provider as expected", testReloadReplayProtectors)
	t.Run("Restoring a snapshot calls the post-restore callback if available", testReloadRestore)
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
	for i, k := range keys1 {
		prov1.EXPECT().GetHash(k).Times(1).Return(hash1[i], nil)
		prov1.EXPECT().GetState(k).Times(1).Return(data1[i], nil, nil)
	}
	// duplicate key is skipped
	prov2.EXPECT().GetHash(keys2[1]).Times(1).Return(hash2[0], nil)
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
	engine.time.EXPECT().GetTimeNow().Times(2).Return(now)
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
		hash := crypto.Hash(data)
		prov.EXPECT().GetHash(k).Times(2).Return(hash, nil)
		prov.EXPECT().GetState(k).Times(1).Return(data, nil, nil)
	}

	// take the snapshot knowing state has changed:
	// we need the ctx that goes with the mock, because that has block height and hash set
	hash, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
	secondHash, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
	require.EqualValues(t, hash, secondHash)
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
		hash := crypto.Hash(data)
		prov.EXPECT().GetHash(k).Times(1).Return(hash, nil)
		prov.EXPECT().GetState(k).Times(1).Return(data, nil, nil)
	}
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
}

func testReloadReplayProtectors(t *testing.T) {
	e := getTestEngine(t)
	defer e.Finish()
	rpPl := types.PayloadReplayProtection{
		Blocks: []*types.ReplayBlockTransactions{
			{
				Transactions: []string{"foo", "bar"},
			},
		},
	}
	payload := types.Payload{
		Data: &rpPl,
	}
	data, err := proto.Marshal(payload.IntoProto())
	require.NoError(t, err)
	hash := crypto.Hash(data)
	rpl := e.getNewProviderMock()
	rpl.EXPECT().Namespace().Times(1).Return(payload.Namespace())
	rpl.EXPECT().Keys().Times(1).Return([]string{payload.Key()})

	old := e.getNewProviderMock()
	old.EXPECT().Namespace().Times(2).Return(payload.Namespace())
	old.EXPECT().Keys().Times(3).Return([]string{payload.Key()}) // this gets called a second time when replacing
	old.EXPECT().GetHash(payload.Key()).Times(1).Return(hash, nil)
	old.EXPECT().GetState(payload.Key()).Times(1).Return(data, nil, nil)
	old.EXPECT().LoadState(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, pl *types.Payload) ([]types.StateProvider, error) {
		switch dt := pl.Data.(type) {
		case *types.PayloadReplayProtection:
			require.Equal(t, len(dt.Blocks), len(rpPl.Blocks))
			require.EqualValues(t, dt.Blocks[0], rpPl.Blocks[0])
		default:
			t.Fatal("Incorrect payload type passed")
		}
		return []types.StateProvider{rpl}, nil
	})
	// call is made when creating the snapshot
	now := time.Now()
	e.time.EXPECT().GetTimeNow().Times(1).Return(now)

	// add old provider
	e.AddProviders(old)
	// take a snapshot
	snapHash, err := e.Snapshot(e.ctx)
	require.NoError(t, err)
	require.NotNil(t, snapHash)
	// now get the snapshot we've just created
	snaps, err := e.List()
	require.NoError(t, err)
	require.NotEmpty(t, snaps)
	require.Equal(t, 1, len(snaps))

	snap := snaps[0]

	// now reload the snapshot on a new engine
	e2 := getTestEngine(t)
	defer e2.Finish()
	e2.time.EXPECT().GetTimeNow().Times(1).Return(now)
	// calls we expect to see when reloading
	e2.time.EXPECT().SetTimeNow(gomock.Any(), gomock.Any()).Times(1).Do(func(_ context.Context, newT time.Time) {
		require.Equal(t, newT.Unix(), now.Unix())
	})
	e2.AddProviders(old)
	require.NoError(t, e2.ReceiveSnapshot(snap))
	ready := false
	for i := uint32(0); i < snap.Chunks; i++ {
		chunk, err := e.LoadSnapshotChunk(snap.Height, uint32(snap.Format), i)
		require.NoError(t, err)
		ready, err = e2.ApplySnapshotChunk(chunk)
		require.NoError(t, err)
	}
	require.True(t, ready)
	// OK, snapshot is ready to be applied
	require.NoError(t, e2.ApplySnapshot(e.ctx))
	// so now we can check if taking a snapshot calls the methods on the replacement provider

	rpl.EXPECT().GetHash(payload.Key()).Times(1).Return(hash, nil)
	snapHash2, err := e2.Snapshot(e.ctx)
	require.NoError(t, err)
	require.NotNil(t, snapHash2)
	require.EqualValues(t, snapHash, snapHash2)
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
		hash := crypto.Hash(data)
		prov.EXPECT().GetHash(k).Times(1).Return(hash, nil)
		prov.EXPECT().GetState(k).Times(1).Return(data, nil, nil)
	}
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
