package evtforward_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	prototypes "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/evtforward/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testSelfVegaPubKey = "self-pubkey"
	testAllPubKeys     = []string{
		testSelfVegaPubKey,
		"another-pubkey1",
		"another-pubkey2",
	}
	okEventEmitter = "somechaineventpubkey"
	allowlist      = []string{okEventEmitter}
	initTime       = time.Unix(10, 0)
)

type testEvtFwd struct {
	*evtforward.EvtForwarder
	ctrl *gomock.Controller
	time *mocks.MockTimeService
	top  *mocks.MockValidatorTopology
	cmd  *mocks.MockCommander
	cb   func(context.Context, time.Time)
}

func getTestEvtFwd(t *testing.T) *testEvtFwd {
	t.Helper()
	ctrl := gomock.NewController(t)
	tim := mocks.NewMockTimeService(ctrl)
	top := mocks.NewMockValidatorTopology(ctrl)
	cmd := mocks.NewMockCommander(ctrl)

	top.EXPECT().AllNodeIDs().Times(1).Return(testAllPubKeys)
	top.EXPECT().SelfNodeID().AnyTimes().Return(testSelfVegaPubKey)
	var cb func(context.Context, time.Time)
	tim.EXPECT().NotifyOnTick(gomock.Any()).Do(func(f func(context.Context, time.Time)) {
		cb = f
	})

	tim.EXPECT().GetTimeNow().Times(1).Return(initTime)

	cfg := evtforward.NewDefaultConfig()
	// add the pubkeys
	cfg.BlockchainQueueAllowlist = allowlist
	evtfwd := evtforward.New(
		logging.NewTestLogger(), cfg,
		cmd, tim, top)

	return &testEvtFwd{
		EvtForwarder: evtfwd,
		ctrl:         ctrl,
		time:         tim,
		top:          top,
		cmd:          cmd,
		cb:           cb,
	}
}

func TestEvtForwarder(t *testing.T) {
	t.Run("test forward success node is forwarder", testForwardSuccessNodeIsForwarder)
	t.Run("test forward failure duplicate event", testForwardFailureDuplicateEvent)
	t.Run("test ensure validators lists are updated", testUpdateValidatorList)
	t.Run("test ack success", testAckSuccess)
	t.Run("test ack failure already acked", testAckFailureAlreadyAcked)
	t.Run("error event emitter not allowlisted", testEventEmitterNotAllowlisted)
}

func testEventEmitterNotAllowlisted(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
	defer evtfwd.ctrl.Finish()
	evt := getTestChainEvent("some")
	evtfwd.top.EXPECT().AllNodeIDs().Times(1).Return(testAllPubKeys)
	// set the time so the hash match our current node
	evtfwd.cb(context.Background(), time.Unix(11, 0))
	err := evtfwd.Forward(context.Background(), evt, "not allowlisted")
	assert.EqualError(t, err, evtforward.ErrPubKeyNotAllowlisted.Error())
}

func testForwardSuccessNodeIsForwarder(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
	defer evtfwd.ctrl.Finish()
	evt := getTestChainEvent("some")
	evtfwd.cmd.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	evtfwd.top.EXPECT().AllNodeIDs().Times(1).Return(testAllPubKeys)
	// set the time so the hash match our current node
	evtfwd.cb(context.Background(), time.Unix(3, 0))
	err := evtfwd.Forward(context.Background(), evt, okEventEmitter)
	assert.NoError(t, err)
}

func testForwardFailureDuplicateEvent(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
	defer evtfwd.ctrl.Finish()
	evt := getTestChainEvent("some")
	evtfwd.cmd.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	evtfwd.top.EXPECT().AllNodeIDs().Times(1).Return(testAllPubKeys)
	// set the time so the hash match our current node
	evtfwd.cb(context.Background(), time.Unix(12, 0))
	err := evtfwd.Forward(context.Background(), evt, okEventEmitter)
	assert.NoError(t, err)
	// now the event should exist, let's try toforward againt
	err = evtfwd.Forward(context.Background(), evt, okEventEmitter)
	assert.EqualError(t, err, evtforward.ErrEvtAlreadyExist.Error())
}

func testUpdateValidatorList(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
	defer evtfwd.ctrl.Finish()
	// no event, just call callback to ensure the validator list is updated
	evtfwd.top.EXPECT().AllNodeIDs().Times(1).Return(testAllPubKeys)
	evtfwd.cb(context.Background(), initTime.Add(time.Second))
}

func testAckSuccess(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
	defer evtfwd.ctrl.Finish()
	evt := getTestChainEvent("some")

	hash1, err := evtfwd.GetHash("all")
	require.Nil(t, err)
	state1, err := evtfwd.GetState("all")
	require.Nil(t, err)

	ok := evtfwd.Ack(evt)
	assert.True(t, ok)
	hash2, err := evtfwd.GetHash("all")
	require.Nil(t, err)
	state2, err := evtfwd.GetState("all")
	require.Nil(t, err)

	require.False(t, bytes.Equal(hash1, hash2))
	require.False(t, bytes.Equal(state1, state2))

	// try to ack again the same event
	ok = evtfwd.Ack(evt)
	assert.False(t, ok)
	hash3, err := evtfwd.GetHash("all")
	require.Nil(t, err)
	state3, err := evtfwd.GetState("all")
	require.Nil(t, err)

	require.True(t, bytes.Equal(hash3, hash2))
	require.True(t, bytes.Equal(state3, state2))

	// restore the state
	var pl snapshot.Payload
	proto.Unmarshal(state3, &pl)
	payload := types.PayloadFromProto(&pl)
	err = evtfwd.LoadState(context.Background(), payload)
	require.Nil(t, err)

	// the event exists after the reload so expect to fail
	ok = evtfwd.Ack(evt)
	assert.False(t, ok)

	// expect the hash/state after the reload to equal what it was before
	hash4, err := evtfwd.GetHash("all")
	require.Nil(t, err)
	state4, err := evtfwd.GetState("all")
	require.Nil(t, err)

	require.True(t, bytes.Equal(hash4, hash3))
	require.True(t, bytes.Equal(state4, state3))

	// ack a new event for the hash/state to change
	evt2 := getTestChainEvent("somenew")
	ok = evtfwd.Ack(evt2)
	assert.True(t, ok)
	hash5, err := evtfwd.GetHash("all")
	require.Nil(t, err)
	state5, err := evtfwd.GetState("all")
	require.Nil(t, err)

	require.False(t, bytes.Equal(hash5, hash4))
	require.False(t, bytes.Equal(state5, state4))
}

func testAckFailureAlreadyAcked(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
	defer evtfwd.ctrl.Finish()
	evt := getTestChainEvent("some")
	ok := evtfwd.Ack(evt)
	assert.True(t, ok)
	// try to ack again
	ko := evtfwd.Ack(evt)
	assert.False(t, ko)
}

func getTestChainEvent(txid string) *commandspb.ChainEvent {
	return &commandspb.ChainEvent{
		TxId: txid,
		Event: &commandspb.ChainEvent_Erc20{
			Erc20: &prototypes.ERC20Event{
				Index: 1,
				Block: 100,
				Action: &prototypes.ERC20Event_AssetList{
					AssetList: &prototypes.ERC20AssetList{
						VegaAssetId: "asset-id-1",
					},
				},
			},
		},
	}
}
