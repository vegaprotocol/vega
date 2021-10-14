package netparams_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testNetParams struct {
	*netparams.Store
	ctrl   *gomock.Controller
	broker *mocks.MockBroker
}

func getTestNetParams(t *testing.T) *testNetParams {
	t.Helper()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	store := netparams.New(
		logging.NewTestLogger(), netparams.NewDefaultConfig(), broker)

	return &testNetParams{
		Store:  store,
		ctrl:   ctrl,
		broker: broker,
	}
}

func TestNetParams(t *testing.T) {
	t.Run("test validate - succes", testValidateSuccess)
	t.Run("test validate - unknown key", testValidateUnknownKey)
	t.Run("test validate - validation failed", testValidateValidationFailed)
	t.Run("test update - success", testUpdateSuccess)
	t.Run("test update - unknown key", testUpdateUnknownKey)
	t.Run("test update - validation failed", testUpdateValidationFailed)
	t.Run("test exists - success", testExistsSuccess)
	t.Run("test exists - failure", testExistsFailure)
	t.Run("get float", testGetFloat)
	t.Run("get duration", testGetDuration)
	t.Run("dispatch after update", testDispatchAfterUpdate)
	t.Run("register dispatch function - failure", testRegisterDispatchFunctionFailure)
}

func TestCheckpoint(t *testing.T) {
	t.Run("test get snapshot not empty", testNonEmptyCheckpoint)
	t.Run("test get snapshot invalid", testInvalidCheckpoint)
}

func testRegisterDispatchFunctionFailure(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	err := netp.Watch(
		netparams.WatchParam{
			Param:   netparams.GovernanceProposalAssetMaxClose,
			Watcher: func(s string) error { return nil },
		},
	)

	assert.EqualError(t, err, "invalid type, expected func(context.Context, time.Duration) error")
}

func testDispatchAfterUpdate(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	netp.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	newDuration := "10s"
	var wasCalled bool
	f := func(_ context.Context, d time.Duration) error {
		assert.Equal(t, d, 10*time.Second)
		wasCalled = true
		return nil
	}

	err := netp.Watch(
		netparams.WatchParam{
			Param:   netparams.GovernanceProposalAssetMaxClose,
			Watcher: f,
		},
	)

	assert.NoError(t, err)

	err = netp.Update(context.Background(), netparams.GovernanceProposalAssetMaxClose, newDuration)
	assert.NoError(t, err)

	netp.DispatchChanges(context.Background())
	assert.True(t, wasCalled)
}

func testValidateSuccess(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	err := netp.Validate(netparams.GovernanceProposalMarketMinClose, "10h")
	assert.NoError(t, err)
}

func testValidateUnknownKey(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	err := netp.Validate("not.a.valid.key", "10h")
	assert.EqualError(t, err, netparams.ErrUnknownKey.Error())
}

func testValidateValidationFailed(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	err := netp.Validate(netparams.GovernanceProposalMarketMinClose, "asdasdasd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "time: invalid duration")
}

func testUpdateSuccess(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	// get the original default value
	ov, err := netp.Get(netparams.GovernanceProposalMarketMinClose)
	assert.NoError(t, err)
	assert.NotEmpty(t, ov)
	assert.NotEqual(t, ov, "10h")

	netp.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	err = netp.Update(
		context.Background(), netparams.GovernanceProposalMarketMinClose, "10h")
	assert.NoError(t, err)

	nv, err := netp.Get(netparams.GovernanceProposalMarketMinClose)
	assert.NoError(t, err)
	assert.NotEmpty(t, nv)
	assert.NotEqual(t, nv, ov)
	assert.Equal(t, nv, "10h")
}

func testUpdateUnknownKey(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	err := netp.Update(context.Background(), "not.a.valid.key", "10h")
	assert.EqualError(t, err, netparams.ErrUnknownKey.Error())
}

func testUpdateValidationFailed(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	err := netp.Update(
		context.Background(), netparams.GovernanceProposalMarketMinClose, "asdadasd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "time: invalid duration")
}

func testExistsSuccess(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	ok := netp.Exists(netparams.GovernanceProposalMarketMinClose)
	assert.True(t, ok)
}

func testExistsFailure(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	ok := netp.Exists("not.valid")
	assert.False(t, ok)
}

func testGetFloat(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	_, err := netp.GetFloat(netparams.GovernanceProposalUpdateNetParamRequiredMajority)
	assert.NoError(t, err)
	_, err = netp.GetFloat(netparams.GovernanceProposalAssetMaxClose)
	assert.EqualError(t, err, "not a float value")
}

func testGetDuration(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	_, err := netp.GetDuration(netparams.GovernanceProposalAssetMaxClose)
	assert.NoError(t, err)
	_, err = netp.GetDuration(netparams.GovernanceProposalAssetMinProposerBalance)
	assert.EqualError(t, err, "not a time.Duration value")
}

func testNonEmptyCheckpoint(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()
	ctx := context.Background()

	// get the original default value
	ov, err := netp.Get(netparams.GovernanceProposalMarketMinClose)
	assert.NoError(t, err)
	assert.NotEmpty(t, ov)
	assert.NotEqual(t, ov, "10h")

	netp.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	err = netp.Update(ctx, netparams.GovernanceProposalMarketMinClose, "10h")
	assert.NoError(t, err)

	nv, err := netp.Get(netparams.GovernanceProposalMarketMinClose)
	assert.NoError(t, err)
	assert.NotEmpty(t, nv)
	assert.NotEqual(t, nv, ov)
	assert.Equal(t, nv, "10h")

	data, err := netp.Checkpoint()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// now try and load the checkpoint
	netp2 := getTestNetParams(t)
	defer netp2.ctrl.Finish()

	// ensure the state != checkpoint we took
	ov2, err := netp2.Get(netparams.GovernanceProposalMarketMinClose)
	assert.NoError(t, err)
	assert.NotEmpty(t, ov2)
	assert.NotEqual(t, ov2, "10h")
	require.Equal(t, ov, ov2)

	netp2.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	require.NoError(t, netp2.Load(ctx, data))

	nv2, err := netp2.Get(netparams.GovernanceProposalMarketMinClose)
	assert.NoError(t, err)
	assert.NotEmpty(t, nv2)
	assert.NotEqual(t, nv2, ov)
	assert.Equal(t, nv, nv2)

	// make sure that, once restored, the same checkpoint data is restored
	data2, err := netp2.Checkpoint()
	require.NoError(t, err)
	require.EqualValues(t, data, data2)
}

func testInvalidCheckpoint(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()
	ctx := context.Background()

	data, err := netp.Checkpoint()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	data = append(data, []byte("foobar")...) // corrupt the data
	require.Error(t, netp.Load(ctx, data))
}
