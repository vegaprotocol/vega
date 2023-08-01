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

package evtforward_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/evtforward"
	"code.vegaprotocol.io/vega/core/evtforward/mocks"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	snp "code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/proto"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	prototypes "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/golang/mock/gomock"
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
	*evtforward.Forwarder
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

	cfg := evtforward.NewDefaultConfig()
	// add the pubkeys
	cfg.BlockchainQueueAllowlist = allowlist
	evtfwd := evtforward.New(
		logging.NewTestLogger(), cfg,
		cmd, tim, top)

	return &testEvtFwd{
		Forwarder: evtfwd,
		ctrl:      ctrl,
		time:      tim,
		top:       top,
		cmd:       cmd,
		cb:        evtfwd.OnTick,
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
	evt := getTestChainEvent("some")
	evtfwd.top.EXPECT().AllNodeIDs().Times(1).Return(testAllPubKeys)
	// set the time so the hash match our current node
	evtfwd.cb(context.Background(), time.Unix(11, 0))
	err := evtfwd.Forward(context.Background(), evt, "not allowlisted")
	assert.EqualError(t, err, evtforward.ErrPubKeyNotAllowlisted.Error())
}

func testForwardSuccessNodeIsForwarder(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
	evt := getTestChainEvent("some")
	evtfwd.cmd.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	evtfwd.top.EXPECT().AllNodeIDs().Times(1).Return(testAllPubKeys)
	evtfwd.time.EXPECT().GetTimeNow().AnyTimes()
	// set the time so the hash match our current node
	evtfwd.cb(context.Background(), time.Unix(3, 0))
	err := evtfwd.Forward(context.Background(), evt, okEventEmitter)
	assert.NoError(t, err)
}

func testForwardFailureDuplicateEvent(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
	evt := getTestChainEvent("some")
	evtfwd.cmd.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	evtfwd.top.EXPECT().AllNodeIDs().Times(1).Return(testAllPubKeys)
	evtfwd.time.EXPECT().GetTimeNow().AnyTimes()
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
	// no event, just call callback to ensure the validator list is updated
	evtfwd.top.EXPECT().AllNodeIDs().Times(1).Return(testAllPubKeys)
	evtfwd.cb(context.Background(), initTime.Add(time.Second))
}

func testAckSuccess(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
	evt := getTestChainEvent("some")
	state1, _, err := evtfwd.GetState("all")
	require.Nil(t, err)

	ok := evtfwd.Ack(evt)
	assert.True(t, ok)
	state2, _, err := evtfwd.GetState("all")
	require.Nil(t, err)
	require.False(t, bytes.Equal(state1, state2))

	// try to ack again the same event
	ok = evtfwd.Ack(evt)
	assert.False(t, ok)
	state3, _, err := evtfwd.GetState("all")
	require.Nil(t, err)
	require.True(t, bytes.Equal(state3, state2))

	// restore the state
	var pl snapshot.Payload
	proto.Unmarshal(state3, &pl)
	payload := types.PayloadFromProto(&pl)
	_, err = evtfwd.LoadState(context.Background(), payload)
	require.Nil(t, err)

	// the event exists after the reload so expect to fail
	ok = evtfwd.Ack(evt)
	assert.False(t, ok)

	// expect the state after the reload to equal what it was before
	state4, _, err := evtfwd.GetState("all")
	require.Nil(t, err)
	require.True(t, bytes.Equal(state4, state3))

	// ack a new event for the hash/state to change
	evt2 := getTestChainEvent("somenew")
	ok = evtfwd.Ack(evt2)
	assert.True(t, ok)
	state5, _, err := evtfwd.GetState("all")
	require.Nil(t, err)
	require.False(t, bytes.Equal(state5, state4))
}

func testAckFailureAlreadyAcked(t *testing.T) {
	evtfwd := getTestEvtFwd(t)
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

func TestSnapshotRoundTripViaEngine(t *testing.T) {
	eventForwarder1 := getTestEvtFwd(t)

	for i := 0; i < 100; i++ {
		eventForwarder1.Ack(getTestChainEvent(crypto.RandomHash()))
	}

	ctx := vgtest.VegaContext("chainid", 100)
	vegaPath := paths.New(t.TempDir())
	now := time.Now()
	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snp.DefaultConfig()

	snapshotEngine1, err := snp.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	snapshotEngine1CloseFn := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer snapshotEngine1CloseFn()

	snapshotEngine1.AddProviders(eventForwarder1)

	require.NoError(t, snapshotEngine1.Start(ctx))

	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		eventForwarder1.Ack(getTestChainEvent(fmt.Sprintf("txHash%d", i)))
	}

	state1 := map[string][]byte{}
	for _, key := range eventForwarder1.Keys() {
		state, additionalProvider, err := eventForwarder1.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state1[key] = state
	}

	snapshotEngine1CloseFn()

	eventForwarder2 := getTestEvtFwd(t)
	snapshotEngine2, err := snp.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	defer snapshotEngine2.Close()

	snapshotEngine2.AddProviders(eventForwarder2)

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, snapshotEngine2.Start(ctx))

	// Comparing the hash after restoration, to ensure it produces the same result.
	hash2, _, _ := snapshotEngine2.Info()
	require.Equal(t, hash1, hash2)

	for i := 0; i < 10; i++ {
		eventForwarder2.Ack(getTestChainEvent(fmt.Sprintf("txHash%d", i)))
	}

	state2 := map[string][]byte{}
	for _, key := range eventForwarder2.Keys() {
		state, additionalProvider, err := eventForwarder2.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state2[key] = state
	}

	for key := range state1 {
		assert.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}
