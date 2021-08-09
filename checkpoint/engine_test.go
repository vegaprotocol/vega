package checkpoint_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/checkpoint"
	"code.vegaprotocol.io/vega/checkpoint/mocks"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type testEngine struct {
	*checkpoint.Engine
	ctrl *gomock.Controller
}

func getTestEngine(t *testing.T) *testEngine {
	ctrl := gomock.NewController(t)
	eng, _ := checkpoint.New()
	return &testEngine{
		Engine: eng,
		ctrl:   ctrl,
	}
}

func TestGetCheckpoints(t *testing.T) {
	t.Run("test getting checkpoints loading in components via constructor - no duplicates", testGetCheckpointsConstructor)
	t.Run("test getting checkpoints loading in components using Add method - no duplicates", testGetCheckpointsAdd)
	t.Run("test adding duplicate components using Add methods", testAddDuplicate)
	t.Run("test adding duplicate component via constructor", testDuplicateConstructor)
	t.Run("test getting checkpoints - error", testGetCheckpointsErr)
}

func TestCheckpointIntervals(t *testing.T) {
	t.Run("test getting checkpoint before interval has passed", testCheckpointBeforeInterval)
	t.Run("test updating interval creates new checkpoint sooner", testCheckpointUpdatedInterval)
}

func TestLoadCheckpoints(t *testing.T) {
	t.Run("test loading checkpoints after generating them - success", testLoadCheckpoints)
	t.Run("load non-registered components", testLoadMissingCheckpoint)
	t.Run("load checkpoint with invalid hash", testLoadInvalidHash)
	t.Run("load sparse checkpoint", testLoadSparse)
	t.Run("error loading checkpoint", testLoadError)
	t.Run("a checkpoint can only be loaded once if configured", testLoadGenesisHashOnlyOnce)
}

type genesis struct {
	CP *checkpoint.GenesisState `json:"checkpoint"`
}

func testGetCheckpointsConstructor(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	defer ctrl.Finish()
	components := map[types.CheckpointName]*mocks.MockState{
		types.GovernanceCheckpoint: mocks.NewMockState(ctrl),
		types.AssetsCheckpoint:     mocks.NewMockState(ctrl),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	eng, err := checkpoint.New(components[types.GovernanceCheckpoint], components[types.AssetsCheckpoint])
	require.NoError(t, err)
	data := map[types.CheckpointName][]byte{
		types.GovernanceCheckpoint: []byte("foodata"),
		types.AssetsCheckpoint:     []byte("bardata"),
	}
	for k, c := range components {
		c.EXPECT().Checkpoint().Times(1).Return(data[k], nil)
	}
	raw, err := eng.Checkpoint(time.Now())
	require.NoError(t, err)
	// now to check if the checkpoint contains the expected data
	for k, c := range components {
		c.EXPECT().Load(data[k]).Times(1).Return(nil)
	}
	// pretend like the genesis block specified this hash to restore
	set := genesis{
		CP: &checkpoint.GenesisState{
			CheckpointHash: raw.Hash,
		},
	}
	gen, err := json.Marshal(set)
	require.NoError(t, err)
	require.NoError(t, eng.UponGenesis(ctx, gen))
	require.NoError(t, eng.Load(ctx, raw))
}

func testGetCheckpointsAdd(t *testing.T) {
	t.Parallel()
	eng := getTestEngine(t)
	ctx := context.Background()
	defer eng.ctrl.Finish()
	components := map[types.CheckpointName]*mocks.MockState{
		types.GovernanceCheckpoint: mocks.NewMockState(eng.ctrl),
		types.AssetsCheckpoint:     mocks.NewMockState(eng.ctrl),
	}
	data := map[types.CheckpointName][]byte{
		types.GovernanceCheckpoint: []byte("foodata"),
		types.AssetsCheckpoint:     []byte("bardata"),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	require.NoError(t, eng.Add(components[types.GovernanceCheckpoint], components[types.AssetsCheckpoint]))
	for k, c := range components {
		c.EXPECT().Checkpoint().Times(1).Return(data[k], nil)
	}
	raw, err := eng.Checkpoint(time.Now())
	require.NoError(t, err)
	// now to check if the checkpoint contains the expected data
	for k, c := range components {
		c.EXPECT().Load(data[k]).Times(1).Return(nil)
	}
	// pretend like the genesis block specified this hash to restore
	set := genesis{
		CP: &checkpoint.GenesisState{
			CheckpointHash: raw.Hash,
		},
	}
	gen, err := json.Marshal(set)
	require.NoError(t, err)
	require.NoError(t, eng.UponGenesis(ctx, gen))
	require.NoError(t, eng.Load(ctx, raw))
}

func testAddDuplicate(t *testing.T) {
	t.Parallel()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	comp := mocks.NewMockState(eng.ctrl)
	comp.EXPECT().Name().Times(2).Return(types.GovernanceCheckpoint)
	require.NoError(t, eng.Add(comp, comp)) // adding the exact same component (same ptr value)
	comp2 := mocks.NewMockState(eng.ctrl)
	comp2.EXPECT().Name().Times(1).Return(types.GovernanceCheckpoint)
	require.Error(t, eng.Add(comp2))
}

func testDuplicateConstructor(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	comp := mocks.NewMockState(ctrl)
	comp.EXPECT().Name().Times(3).Return(types.GovernanceCheckpoint)
	comp2 := mocks.NewMockState(ctrl)
	comp2.EXPECT().Name().Times(1).Return(types.GovernanceCheckpoint)
	// this is all good
	eng, err := checkpoint.New(comp, comp)
	require.NoError(t, err)
	require.NotNil(t, eng)
	eng, err = checkpoint.New(comp, comp2)
	require.Error(t, err)
	require.Nil(t, eng)
}

func testLoadCheckpoints(t *testing.T) {
	t.Parallel()
	eng := getTestEngine(t)
	ctx := context.Background()
	defer eng.ctrl.Finish()
	components := map[types.CheckpointName]*mocks.MockState{
		types.GovernanceCheckpoint: mocks.NewMockState(eng.ctrl),
		types.AssetsCheckpoint:     mocks.NewMockState(eng.ctrl),
	}
	data := map[types.CheckpointName][]byte{
		types.GovernanceCheckpoint: []byte("foodata"),
		types.AssetsCheckpoint:     []byte("bardata"),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	require.NoError(t, eng.Add(components[types.GovernanceCheckpoint], components[types.AssetsCheckpoint]))
	for k, c := range components {
		c.EXPECT().Checkpoint().Times(1).Return(data[k], nil)
	}
	snapshot, err := eng.Checkpoint(time.Now())
	require.NoError(t, err)
	require.NotEmpty(t, snapshot)
	// create new components to load data in to
	wComps := map[types.CheckpointName]*wrappedMock{
		types.GovernanceCheckpoint: wrapMock(mocks.NewMockState(eng.ctrl)),
		types.AssetsCheckpoint:     wrapMock(mocks.NewMockState(eng.ctrl)),
	}
	for k, c := range wComps {
		c.EXPECT().Name().Times(1).Return(k)
		c.EXPECT().Load(data[k]).Times(1).Return(nil)
	}
	newEng, err := checkpoint.New(wComps[types.GovernanceCheckpoint], wComps[types.AssetsCheckpoint])
	require.NoError(t, err)
	require.NotNil(t, newEng)
	// pretend like the genesis block specified this hash to restore
	set := genesis{
		CP: &checkpoint.GenesisState{
			CheckpointHash: snapshot.Hash,
		},
	}
	gen, err := json.Marshal(set)
	require.NoError(t, err)
	require.NoError(t, newEng.UponGenesis(ctx, gen))
	require.NoError(t, newEng.Load(ctx, snapshot))
	for k, exp := range data {
		wc := wComps[k]
		require.EqualValues(t, exp, wc.data)
	}
}

func testLoadMissingCheckpoint(t *testing.T) {
	t.Parallel()
	eng := getTestEngine(t)
	ctx := context.Background()
	defer eng.ctrl.Finish()

	// create checkpoint data
	cp := &types.Checkpoint{
		Assets: []byte("assets"),
	}
	snap := &types.Snapshot{}
	snap.SetCheckpoint(cp)
	// pretend like the genesis block specified this hash to restore
	set := genesis{
		CP: &checkpoint.GenesisState{
			CheckpointHash: snap.Hash,
		},
	}
	gen, err := json.Marshal(set)
	require.NoError(t, err)
	require.NoError(t, eng.UponGenesis(ctx, gen))
	err = eng.Load(ctx, snap)
	require.Error(t, err)
	require.Equal(t, checkpoint.ErrUnknownCheckpointName, err)
	// now try to tamper with the data itself in such a way that the has no longer matches:
	cp.Assets = []byte("foobar")
	b, err := vega.Marshal(cp.IntoProto())
	require.NoError(t, err)
	snap.State = b
	// reset genesis hash
	require.NoError(t, eng.UponGenesis(ctx, gen))
	err = eng.Load(ctx, snap)
	require.Error(t, err)
	require.Equal(t, checkpoint.ErrSnapshotHashIncorrect, err)
}

func testLoadInvalidHash(t *testing.T) {
	t.Parallel()
	eng := getTestEngine(t)
	ctx := context.Background()
	defer eng.ctrl.Finish()

	cp := &types.Checkpoint{
		Assets: []byte("assets"),
	}
	snap := &types.Snapshot{}
	snap.SetCheckpoint(cp)
	// pretend like the genesis block specified this hash to restore
	set := genesis{
		CP: &checkpoint.GenesisState{
			CheckpointHash: snap.Hash,
		},
	}
	gen, err := json.Marshal(set)
	require.NoError(t, err)
	require.NoError(t, eng.UponGenesis(ctx, gen))
	// update data -> hash is invalid
	cp.Assets = []byte("foobar")
	b, err := vega.Marshal(cp.IntoProto())
	require.NoError(t, err)
	snap.State = b
	err = eng.Load(ctx, snap)
	require.Error(t, err)
	require.Equal(t, checkpoint.ErrSnapshotHashIncorrect, err)
}

func testLoadSparse(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	defer ctrl.Finish()
	components := map[types.CheckpointName]*mocks.MockState{
		types.GovernanceCheckpoint: mocks.NewMockState(ctrl),
		types.AssetsCheckpoint:     mocks.NewMockState(ctrl),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	eng, err := checkpoint.New(components[types.GovernanceCheckpoint])
	require.NoError(t, err)
	data := map[types.CheckpointName][]byte{
		types.GovernanceCheckpoint: []byte("foodata"),
	}
	c := components[types.GovernanceCheckpoint]
	c.EXPECT().Checkpoint().Times(1).Return(data[types.GovernanceCheckpoint], nil)
	snapshot, err := eng.Checkpoint(time.Now())
	require.NoError(t, err)
	require.NoError(t, eng.Add(components[types.AssetsCheckpoint])) // load another component, not part of the checkpoints map
	c.EXPECT().Load(data[types.GovernanceCheckpoint]).Times(1).Return(nil)
	// pretend like the genesis block specified this hash to restore
	set := genesis{
		CP: &checkpoint.GenesisState{
			CheckpointHash: snapshot.Hash,
		},
	}
	gen, err := json.Marshal(set)
	require.NoError(t, err)
	require.NoError(t, eng.UponGenesis(ctx, gen))
	require.NoError(t, eng.Load(ctx, snapshot))
}

func testLoadError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	defer ctrl.Finish()
	components := map[types.CheckpointName]*mocks.MockState{
		types.GovernanceCheckpoint: mocks.NewMockState(ctrl),
		types.AssetsCheckpoint:     mocks.NewMockState(ctrl),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	eng, err := checkpoint.New(components[types.GovernanceCheckpoint], components[types.AssetsCheckpoint])
	require.NoError(t, err)
	data := map[types.CheckpointName][]byte{
		types.GovernanceCheckpoint: []byte("foodata"),
		types.AssetsCheckpoint:     []byte("bardata"),
	}
	for k, c := range components {
		c.EXPECT().Checkpoint().Times(1).Return(data[k], nil)
	}
	ret := map[types.CheckpointName]error{
		types.GovernanceCheckpoint: errors.New("random error"),
		types.AssetsCheckpoint:     nil, // we always load checkpoints in order, so bar will go first, and should not return an error
	}
	checkpoints, err := eng.Checkpoint(time.Now())
	require.NoError(t, err)
	for k, r := range ret {
		c := components[k]
		c.EXPECT().Load(data[k]).Times(1).Return(r)
	}
	// pretend like the genesis block specified this hash to restore
	set := genesis{
		CP: &checkpoint.GenesisState{
			CheckpointHash: checkpoints.Hash,
		},
	}
	gen, err := json.Marshal(set)
	require.NoError(t, err)
	require.NoError(t, eng.UponGenesis(ctx, gen))
	err = eng.Load(ctx, checkpoints)
	require.Error(t, err)
	require.Equal(t, ret[types.GovernanceCheckpoint], err)
}

func testGetCheckpointsErr(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	components := map[types.CheckpointName]*mocks.MockState{
		types.GovernanceCheckpoint: mocks.NewMockState(ctrl),
		types.AssetsCheckpoint:     mocks.NewMockState(ctrl),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	eng, err := checkpoint.New(components[types.GovernanceCheckpoint], components[types.AssetsCheckpoint])
	require.NoError(t, err)
	data := map[types.CheckpointName][]byte{
		types.GovernanceCheckpoint: nil,
		types.AssetsCheckpoint:     []byte("bardata"),
	}
	errs := map[types.CheckpointName]error{
		types.GovernanceCheckpoint: fmt.Errorf("random error"),
		types.AssetsCheckpoint:     nil,
	}
	for k, c := range components {
		c.EXPECT().Checkpoint().Times(1).Return(data[k], errs[k])
	}
	checkpoints, err := eng.Checkpoint(time.Now())
	require.Nil(t, checkpoints)
	require.Error(t, err)
	require.Equal(t, errs[types.GovernanceCheckpoint], err)
}

func testCheckpointBeforeInterval(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	components := map[types.CheckpointName]*mocks.MockState{
		types.GovernanceCheckpoint: mocks.NewMockState(ctrl),
		types.AssetsCheckpoint:     mocks.NewMockState(ctrl),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	eng, err := checkpoint.New(components[types.GovernanceCheckpoint], components[types.AssetsCheckpoint])
	require.NoError(t, err)
	// set interval of 1 hour
	hour, _ := time.ParseDuration("1h")
	eng.OnTimeElapsedUpdate(ctx, hour)
	data := map[types.CheckpointName][]byte{
		types.GovernanceCheckpoint: []byte("foodata"),
		types.AssetsCheckpoint:     []byte("bardata"),
	}
	for k, c := range components {
		c.EXPECT().Checkpoint().Times(1).Return(data[k], nil)
	}
	now := time.Now()
	raw, err := eng.Checkpoint(now)
	require.NoError(t, err)

	halfHour := time.Duration(int64(hour) / 2)
	now = now.Add(halfHour)
	raw, err = eng.Checkpoint(now)
	require.Nil(t, raw)
	require.Nil(t, err)
}

// same test as above, but the interval is upadted to trigger a second checkpoint
// to be created anyway
func testCheckpointUpdatedInterval(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	components := map[types.CheckpointName]*mocks.MockState{
		types.GovernanceCheckpoint: mocks.NewMockState(ctrl),
		types.AssetsCheckpoint:     mocks.NewMockState(ctrl),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	eng, err := checkpoint.New(components[types.GovernanceCheckpoint], components[types.AssetsCheckpoint])
	require.NoError(t, err)
	// set interval of 1 hour
	hour, _ := time.ParseDuration("1h")
	eng.OnTimeElapsedUpdate(ctx, hour)
	data := map[types.CheckpointName][]byte{
		types.GovernanceCheckpoint: []byte("foodata"),
		types.AssetsCheckpoint:     []byte("bardata"),
	}
	for k, c := range components {
		// we expect 2 calls
		c.EXPECT().Checkpoint().Times(2).Return(data[k], nil)
	}
	now := time.Now()
	raw, err := eng.Checkpoint(now)
	require.NoError(t, err)

	// this is before we ought to create a checkpoint, and should return nil
	halfHour := time.Duration(int64(hour) / 2)
	now = now.Add(halfHour)
	raw, err = eng.Checkpoint(now)
	require.Nil(t, raw)
	require.Nil(t, err)

	// now the second calls to the components are made
	now = now.Add(time.Second)             // t+30m1s
	eng.OnTimeElapsedUpdate(ctx, halfHour) // delta is 30 min
	raw, err = eng.Checkpoint(now)
	require.NoError(t, err)
	require.NotNil(t, raw)
}

func testLoadGenesisHashOnlyOnce(t *testing.T) {
	t.Parallel()
	eng := getTestEngine(t)
	ctx := context.Background()
	defer eng.ctrl.Finish()
	components := map[types.CheckpointName]*mocks.MockState{
		types.GovernanceCheckpoint: mocks.NewMockState(eng.ctrl),
		types.AssetsCheckpoint:     mocks.NewMockState(eng.ctrl),
	}
	data := map[types.CheckpointName][]byte{
		types.GovernanceCheckpoint: []byte("foodata"),
		types.AssetsCheckpoint:     []byte("bardata"),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	require.NoError(t, eng.Add(components[types.GovernanceCheckpoint], components[types.AssetsCheckpoint]))
	for k, c := range components {
		c.EXPECT().Checkpoint().Times(1).Return(data[k], nil)
	}
	raw, err := eng.Checkpoint(time.Now())
	require.NoError(t, err)
	// calling load with this checkpoint now is a noop
	require.NoError(t, eng.Load(ctx, raw))
	// pretend like the genesis block specified this hash to restore
	set := genesis{
		CP: &checkpoint.GenesisState{
			CheckpointHash: raw.Hash,
		},
	}
	gen, err := json.Marshal(set)
	require.NoError(t, err)
	// set the genesis hash to some new value
	set.CP.CheckpointHash = append(set.CP.CheckpointHash, []byte("foo")...)
	different, err := json.Marshal(set)
	require.NoError(t, err)
	// set up the engine to accept that hash
	require.NoError(t, eng.UponGenesis(ctx, different))
	// this doesn´t  call "load" on the components
	require.NoError(t, eng.Load(ctx, raw))
	// now set the engine to accept the hash of the data we want to load
	require.NoError(t, eng.UponGenesis(ctx, gen))
	// now we do expect the calls to be made, but only once
	for k, c := range components {
		c.EXPECT().Load(data[k]).Times(1).Return(nil)
	}
	require.NoError(t, eng.Load(ctx, raw))
	// subsequent calls to load this checkpoint do nothing
	require.NoError(t, eng.Load(ctx, raw))
}

type wrappedMock struct {
	*mocks.MockState
	data []byte
}

func wrapMock(m *mocks.MockState) *wrappedMock {
	return &wrappedMock{
		MockState: m,
	}
}

func (w *wrappedMock) Load(data []byte) error {
	w.data = data
	return w.MockState.Load(data)
}
