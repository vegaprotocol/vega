package checkpoint_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/checkpoint"
	"code.vegaprotocol.io/vega/checkpoint/mocks"

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
	components := map[string]*mocks.MockState{
		"foo": mocks.NewMockState(ctrl),
		"bar": mocks.NewMockState(ctrl),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	eng, err := checkpoint.New(components["foo"], components["bar"])
	require.NoError(t, err)
	hashes := map[string][]byte{
		"foo": []byte("foohash"),
		"bar": []byte("barhash"),
	}
	data := map[string][]byte{
		"foo": []byte("foodata"),
		"bar": []byte("bardata"),
	}
	for k, c := range components {
		c.EXPECT().Hash().Times(1).Return(hashes[k])
		c.EXPECT().Checkpoint().Times(1).Return(data[k])
	}
	checkpoints := eng.GetCheckpoints()
	for k, cp := range checkpoints {
		require.EqualValues(t, hashes[k], cp.Hash())
		require.EqualValues(t, data[k], cp.Data())
	}
}

func testGetCheckpointsAdd(t *testing.T) {
	t.Parallel()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	components := map[string]*mocks.MockState{
		"foo": mocks.NewMockState(eng.ctrl),
		"bar": mocks.NewMockState(eng.ctrl),
	}
	hashes := map[string][]byte{
		"foo": []byte("foohash"),
		"bar": []byte("barhash"),
	}
	data := map[string][]byte{
		"foo": []byte("foodata"),
		"bar": []byte("bardata"),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	require.NoError(t, eng.Add(components["foo"], components["bar"]))
	for k, c := range components {
		c.EXPECT().Hash().Times(1).Return(hashes[k])
		c.EXPECT().Checkpoint().Times(1).Return(data[k])
	}
	checkpoints := eng.GetCheckpoints()
	for k, cp := range checkpoints {
		require.EqualValues(t, hashes[k], cp.Hash())
		require.EqualValues(t, data[k], cp.Data())
	}
}

func testAddDuplicate(t *testing.T) {
	t.Parallel()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	comp := mocks.NewMockState(eng.ctrl)
	comp.EXPECT().Name().Times(2).Return("duplicate")
	require.NoError(t, eng.Add(comp, comp)) // adding the exact same component (same ptr value)
	comp2 := mocks.NewMockState(eng.ctrl)
	comp2.EXPECT().Name().Times(1).Return("duplicate")
	require.Error(t, eng.Add(comp2))
}

func testDuplicateConstructor(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	comp := mocks.NewMockState(ctrl)
	comp.EXPECT().Name().Times(3).Return("duplicate")
	comp2 := mocks.NewMockState(ctrl)
	comp2.EXPECT().Name().Times(1).Return("duplicate")
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
	components := map[string]*mocks.MockState{
		"foo": mocks.NewMockState(eng.ctrl),
		"bar": mocks.NewMockState(eng.ctrl),
	}
	hashes := map[string][]byte{
		"foo": []byte("foohash"),
		"bar": []byte("barhash"),
	}
	data := map[string][]byte{
		"foo": []byte("foodata"),
		"bar": []byte("bardata"),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	require.NoError(t, eng.Add(components["foo"], components["bar"]))
	for k, c := range components {
		c.EXPECT().Hash().Times(1).Return(hashes[k])
		c.EXPECT().Checkpoint().Times(1).Return(data[k])
	}
	checkpoints := eng.GetCheckpoints()
	for k, cp := range checkpoints {
		require.EqualValues(t, hashes[k], cp.Hash())
		require.EqualValues(t, data[k], cp.Data())
	}
	// create new components to load data in to
	wComps := map[string]*wrappedMock{
		"foo": wrapMock(mocks.NewMockState(eng.ctrl)),
		"bar": wrapMock(mocks.NewMockState(eng.ctrl)),
	}
	for k, c := range wComps {
		c.EXPECT().Name().Times(1).Return(k)
		cp := checkpoints[k]
		c.EXPECT().Load(cp.Data(), cp.Hash()).Times(1).Return(nil)
	}
	newEng, err := checkpoint.New(wComps["foo"], wComps["bar"])
	require.NoError(t, err)
	require.NotNil(t, newEng)
	require.NoError(t, newEng.Load(checkpoints))
	for k, cp := range checkpoints {
		wc := wComps[k]
		require.EqualValues(t, cp.Data(), wc.data)
		require.EqualValues(t, cp.Hash(), wc.hash)
	}
}

func testLoadMissingCheckpoint(t *testing.T) {
	t.Parallel()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	checkpoints := map[string]checkpoint.Snapshot{
		"foobar": checkpoint.Snapshot{},
	}
	err := eng.Load(checkpoints)
	require.Error(t, err)
	require.Equal(t, checkpoint.ErrUnknownCheckpointName, err)
}

func testLoadSparse(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	components := map[string]*mocks.MockState{
		"foo": mocks.NewMockState(ctrl),
		"bar": mocks.NewMockState(ctrl),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	eng, err := checkpoint.New(components["foo"])
	require.NoError(t, err)
	hashes := map[string][]byte{
		"foo": []byte("foohash"),
		// "bar": []byte("barhash"),
	}
	data := map[string][]byte{
		"foo": []byte("foodata"),
		// "bar": []byte("bardata"),
	}
	c := components["foo"]
	c.EXPECT().Hash().Times(1).Return(hashes["foo"])
	c.EXPECT().Checkpoint().Times(1).Return(data["foo"])
	checkpoints := eng.GetCheckpoints()
	require.NoError(t, eng.Add(components["bar"])) // load another component, not part of the checkpoints map
	c.EXPECT().Load(data["foo"], hashes["foo"]).Times(1).Return(nil)
	require.NoError(t, eng.Load(checkpoints))
}

func testLoadError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	components := map[string]*mocks.MockState{
		"foo": mocks.NewMockState(ctrl),
		"bar": mocks.NewMockState(ctrl),
	}
	for k, c := range components {
		c.EXPECT().Name().Times(1).Return(k)
	}
	eng, err := checkpoint.New(components["foo"], components["bar"])
	require.NoError(t, err)
	hashes := map[string][]byte{
		"foo": []byte("foohash"),
		"bar": []byte("barhash"),
	}
	data := map[string][]byte{
		"foo": []byte("foodata"),
		"bar": []byte("bardata"),
	}
	for k, c := range components {
		c.EXPECT().Hash().Times(1).Return(hashes[k])
		c.EXPECT().Checkpoint().Times(1).Return(data[k])
	}
	ret := map[string]error{
		"foo": errors.New("random error"),
		"bar": nil, // we always load checkpoints in order, so bar will go first, and should not return an error
	}
	checkpoints := eng.GetCheckpoints()
	for k, cp := range checkpoints {
		c := components[k]
		c.EXPECT().Load(cp.Data(), cp.Hash()).Times(1).Return(ret[k])
	}
	err = eng.Load(checkpoints)
	require.Error(t, err)
	require.Equal(t, ret["foo"], err)
}

type wrappedMock struct {
	*mocks.MockState
	hash, data []byte
}

func wrapMock(m *mocks.MockState) *wrappedMock {
	return &wrappedMock{
		MockState: m,
	}
}

func (w *wrappedMock) Load(data, hash []byte) error {
	w.data = data
	w.hash = hash
	return w.MockState.Load(data, hash)
}
