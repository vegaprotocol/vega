package evtforward_test

import (
	"context"
	"testing"
	"time"

	types "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/evtforward/mocks"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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
	evt := getTestChainEvent()
	evtfwd.top.EXPECT().AllNodeIDs().Times(1).Return(testAllPubKeys)
	// set the time so the hash match our current node
	evtfwd.cb(context.Background(), time.Unix(11, 0))
	err := evtfwd.Forward(context.Background(), evt, "not allowlisted")
	assert.EqualError(t, err, evtforward.ErrPubKeyNotAllowlisted.Error())
}

func testForwardSuccessNodeIsForwarder(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
	defer evtfwd.ctrl.Finish()
	evt := getTestChainEvent()
	evtfwd.cmd.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	evtfwd.top.EXPECT().AllNodeIDs().Times(1).Return(testAllPubKeys)
	// set the time so the hash match our current node
	evtfwd.cb(context.Background(), time.Unix(9, 0))
	err := evtfwd.Forward(context.Background(), evt, okEventEmitter)
	assert.NoError(t, err)
}

func testForwardFailureDuplicateEvent(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
	defer evtfwd.ctrl.Finish()
	evt := getTestChainEvent()
	evtfwd.cmd.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	evtfwd.top.EXPECT().AllNodeIDs().Times(1).Return(testAllPubKeys)
	// set the time so the hash match our current node
	evtfwd.cb(context.Background(), time.Unix(10, 0))
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
	evt := getTestChainEvent()
	ok := evtfwd.Ack(evt)
	assert.True(t, ok)
}

func testAckFailureAlreadyAcked(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
	defer evtfwd.ctrl.Finish()
	evt := getTestChainEvent()
	ok := evtfwd.Ack(evt)
	assert.True(t, ok)
	// try to ack again
	ko := evtfwd.Ack(evt)
	assert.False(t, ko)
}

func getTestChainEvent() *commandspb.ChainEvent {
	return &commandspb.ChainEvent{
		TxId: "somehash",
		Event: &commandspb.ChainEvent_Erc20{
			Erc20: &types.ERC20Event{
				Index: 1,
				Block: 100,
				Action: &types.ERC20Event_AssetList{
					AssetList: &types.ERC20AssetList{
						VegaAssetId: "asset-id-1",
					},
				},
			},
		},
	}
}
