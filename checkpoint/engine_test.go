package checkpoint_test

import (
	"errors"
	"fmt"
	"testing"

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

func TestLoadCheckpoints(t *testing.T) {
	t.Run("test loading checkpoints after generating them - success", testLoadCheckpoints)
	t.Run("load non-registered components", testLoadMissingCheckpoint)
	t.Run("load sparse checkpoint", testLoadSparse)
	t.Run("error loading checkpoint", testLoadError)
}

func testGetCheckpointsConstructor(t *testing.T) {
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
		types.GovernanceCheckpoint: []byte("foodata"),
		types.AssetsCheckpoint:     []byte("bardata"),
	}
	for k, c := range components {
		c.EXPECT().Checkpoint().Times(1).Return(data[k], nil)
	}
	checkpoints, err := eng.GetCheckpoints()
	require.NoError(t, err)
	for k, cp := range checkpoints {
		require.EqualValues(t, data[types.CheckpointName(k)], cp.Data())
	}
}

func testGetCheckpointsAdd(t *testing.T) {
	t.Parallel()
	eng := getTestEngine(t)
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
	checkpoints, err := eng.GetCheckpoints()
	require.NoError(t, err)
	for k, cp := range checkpoints {
		require.EqualValues(t, data[types.CheckpointName(k)], cp.Data())
	}
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
	checkpoints, err := eng.GetCheckpoints()
	require.NoError(t, err)
	for k, cp := range checkpoints {
		require.EqualValues(t, data[types.CheckpointName(k)], cp.Data())
	}
	// create new components to load data in to
	wComps := map[types.CheckpointName]*wrappedMock{
		types.GovernanceCheckpoint: wrapMock(mocks.NewMockState(eng.ctrl)),
		types.AssetsCheckpoint:     wrapMock(mocks.NewMockState(eng.ctrl)),
	}
	for k, c := range wComps {
		c.EXPECT().Name().Times(1).Return(k)
		cp := checkpoints[string(k)]
		c.EXPECT().Load(cp.Data()).Times(1).Return(nil)
	}
	newEng, err := checkpoint.New(wComps[types.GovernanceCheckpoint], wComps[types.AssetsCheckpoint])
	require.NoError(t, err)
	require.NotNil(t, newEng)
	require.NoError(t, newEng.Load(checkpoints))
	for k, cp := range checkpoints {
		wc := wComps[types.CheckpointName(k)]
		require.EqualValues(t, cp.Data(), wc.data)
	}
}

func testLoadMissingCheckpoint(t *testing.T) {
	t.Parallel()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	k := string(types.AssetsCheckpoint)
	checkpoints := map[string]checkpoint.Snapshot{
		k: checkpoint.Snapshot{},
	}
	err := eng.Load(checkpoints)
	require.Error(t, err)
	require.Equal(t, checkpoint.ErrUnknownCheckpointName, err)
}

func testLoadSparse(t *testing.T) {
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
	eng, err := checkpoint.New(components[types.GovernanceCheckpoint])
	require.NoError(t, err)
	data := map[types.CheckpointName][]byte{
		types.GovernanceCheckpoint: []byte("foodata"),
	}
	c := components[types.GovernanceCheckpoint]
	c.EXPECT().Checkpoint().Times(1).Return(data[types.GovernanceCheckpoint], nil)
	checkpoints, err := eng.GetCheckpoints()
	require.NoError(t, err)
	require.NoError(t, eng.Add(components[types.AssetsCheckpoint])) // load another component, not part of the checkpoints map
	c.EXPECT().Load(data[types.GovernanceCheckpoint]).Times(1).Return(nil)
	require.NoError(t, eng.Load(checkpoints))
}

func testLoadError(t *testing.T) {
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
	checkpoints, err := eng.GetCheckpoints()
	require.NoError(t, err)
	for k, cp := range checkpoints {
		name := types.CheckpointName(k)
		c := components[name]
		c.EXPECT().Load(cp.Data()).Times(1).Return(ret[name])
	}
	err = eng.Load(checkpoints)
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
	checkpoints, err := eng.GetCheckpoints()
	require.Nil(t, checkpoints)
	require.Error(t, err)
	require.Equal(t, errs[types.GovernanceCheckpoint], err)
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
