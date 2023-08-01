package snapshot_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/snapshot/mocks"
	"code.vegaprotocol.io/vega/core/types"
	typemocks "code.vegaprotocol.io/vega/core/types/mocks"
	vegactx "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmtypes "github.com/tendermint/tendermint/abci/types"
)

func TestEngine(t *testing.T) {
	t.Run("Restoring state succeeds", testRestoringStateSucceeds)
	t.Run("Restoring state at a specific block height succeeds", testRestoringStateAtSpecificBlockHeightSucceeds)
	t.Run("Taking a snapshot succeeds", TestTakingSnapshotSucceeds)
	t.Run("State providers can live under same namespace but with different keys", testProvidersSameNamespaceDifferentKeys)
}

// testRestoringStateSucceeds restores a state by simulating state-sync, and save
// the snapshot locally. It then simulates restoring the state from the newly
// saved snapshots.
func testRestoringStateSucceeds(t *testing.T) {
	// The snapshot to be restored via state-sync and then, from local storage.
	testSnapshot := firstSnapshot(t)

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	vegaPaths := paths.New(t.TempDir())
	log := logging.NewTestLogger()

	// Some providers matching the snapshot payloads.
	governanceProvider := newGovernanceProvider(t, ctrl)
	delegationProvider := newDelegationProvider(t, ctrl)
	epochProvider := newEpochProvider(t, ctrl)
	statsService := mocks.NewMockStatsService(ctrl)
	timeService := mocks.NewMockTimeService(ctrl)

	engine, err := snapshot.NewEngine(vegaPaths, snapshot.DefaultConfig(), log, timeService, statsService)
	require.NoError(t, err)
	closeEngine := vgtest.OnlyOnce(engine.Close)
	defer closeEngine()

	// Since we are initializing an engine, in a brand new environment, the snapshot
	// databases should be empty.
	hasSnapshot, err := engine.HasSnapshots()
	require.NoError(t, err)
	require.False(t, hasSnapshot, "The engine shouldn't have any snapshot")
	latestSnapshots, err := engine.ListLatestSnapshots()
	require.NoError(t, err)
	require.Empty(t, latestSnapshots, "There shouldn't be any snapshot")

	// Add the providers.
	engine.AddProviders(governanceProvider)
	engine.AddProviders(delegationProvider)
	engine.AddProviders(epochProvider)

	// From that point, start simulating state-sync.

	// Starting the engine.
	require.NoError(t, engine.Start(ctx))

	// No state should be restored because there is no local snapshot.
	require.False(t, engine.HasRestoredStateAlready(), "No state should have been restored")
	// Therefore, the `Info()` should return empty information.
	snapshotHash, height, chainID := engine.Info()
	require.Zero(t, snapshotHash)
	require.Zero(t, height)
	require.Zero(t, chainID)

	// Simulating the call to the engine from Tendermint ABCI `OfferSnapshot()`.
	response := engine.ReceiveSnapshot(testSnapshot.snapshot)
	require.Equal(t, tmtypes.ResponseOfferSnapshot{
		Result: tmtypes.ResponseOfferSnapshot_ACCEPT,
	}, response)

	// When all the chunks are loaded, the state restoration is triggered by
	// converting the chunks to payload, that are then broadcast to the providers.

	timeService.EXPECT().SetTimeNow(gomock.Any(), time.Unix(0, testSnapshot.appState.Time)).Times(1)
	statsService.EXPECT().SetHeight(testSnapshot.appState.Height).Times(1)
	governanceProvider.EXPECT().LoadState(gomock.Any(), testSnapshot.PayloadGovernanceActive()).Return(nil, nil).Times(1)
	governanceProvider.EXPECT().LoadState(gomock.Any(), testSnapshot.PayloadGovernanceEnacted()).Return(nil, nil).Times(1)
	delegationProvider.EXPECT().LoadState(gomock.Any(), testSnapshot.PayloadDelegationActive()).Return(nil, nil).Times(1)
	epochProvider.EXPECT().LoadState(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

	// Loading each chunk in the engine. When done,  the state restoration is
	// triggered automatically.
	for idx, rawChunk := range testSnapshot.rawChunks {
		response := engine.ReceiveSnapshotChunk(ctx, rawChunk, vgrand.RandomStr(5))
		require.Equal(t, tmtypes.ResponseApplySnapshotChunk{
			Result: tmtypes.ResponseApplySnapshotChunk_ACCEPT,
		}, response, "The raw chunk with index %d should be accepted", idx)
	}

	// Since the state has been restored, the snapshot databases should have one
	// snapshot saved.
	hasSnapshot, err = engine.HasSnapshots()
	require.NoError(t, err)
	require.True(t, hasSnapshot, "The engine should have a snapshot")
	latestSnapshots, err = engine.ListLatestSnapshots()
	require.NoError(t, err)
	require.Len(t, latestSnapshots, 1, "There should have 1 snapshot")
	require.True(t, engine.HasRestoredStateAlready(), "The state should be marked as restored")
	// And, the method `Info()` should return information of the current state.
	snapshotHash, height, chainID = engine.Info()
	require.Equal(t, testSnapshot.snapshot.Hash, snapshotHash)
	require.EqualValues(t, testSnapshot.appState.Height, height)
	require.Equal(t, testSnapshot.appState.ChainID, chainID)

	// Start simulating restoration from the local snapshot that has been creating
	// in the previous steps. This also helps verifying the previous snapshot
	// has correctly been saved locally, as it should after a state-sync.

	// Closing the previous engine instance, so we can simulate a restart.
	closeEngine()

	engine, err = snapshot.NewEngine(vegaPaths, snapshot.DefaultConfig(), log, timeService, statsService)
	require.NoError(t, err)
	closeEngine = vgtest.OnlyOnce(engine.Close)
	defer closeEngine()

	// Add same providers as the previous engine instance.
	engine.AddProviders(governanceProvider)
	engine.AddProviders(delegationProvider)
	engine.AddProviders(epochProvider)

	// Since we should have reload the local snapshot, we should find the previous
	// state loaded but not restored.
	hasSnapshot, err = engine.HasSnapshots()
	require.NoError(t, err)
	require.True(t, hasSnapshot, "The engine should have a snapshot")
	latestSnapshots, err = engine.ListLatestSnapshots()
	require.NoError(t, err)
	require.Len(t, latestSnapshots, 1, "There should have 1 snapshot")

	// The state is not restored yet.
	require.False(t, engine.HasRestoredStateAlready(), "The state should not be restored yet")
	snapshotHash, height, chainID = engine.Info()
	require.Zero(t, snapshotHash)
	require.Zero(t, height)
	require.Zero(t, chainID)

	// Setting up the expectation when the the local snapshot will be started.

	// State restored on the snapshot engine itself, from the local snapshot.
	timeService.EXPECT().SetTimeNow(gomock.Any(), time.Unix(0, testSnapshot.appState.Time)).Times(1)
	statsService.EXPECT().SetHeight(testSnapshot.appState.Height).Times(1)

	// LoadState() is called once for each key. If there are 2 keys, it's called twice.
	governanceProvider.EXPECT().LoadState(gomock.Any(), testSnapshot.PayloadGovernanceActive()).Return(nil, nil).Times(1)
	governanceProvider.EXPECT().LoadState(gomock.Any(), testSnapshot.PayloadGovernanceEnacted()).Return(nil, nil).Times(1)
	delegationProvider.EXPECT().LoadState(gomock.Any(), testSnapshot.PayloadDelegationActive()).Return(nil, nil).Times(1)
	epochProvider.EXPECT().LoadState(gomock.Any(), testSnapshot.PayloadEpoch()).Return(nil, nil).Times(1)

	// Starting the engine.
	require.NoError(t, engine.Start(ctx))

	// Since we have a local snapshot, this time, the engine should have restore
	// the state.
	require.True(t, engine.HasRestoredStateAlready(), "The state should be marked as restored")
	// Therefore, the method `Info()` should return information of the current
	// state.
	snapshotHash, height, chainID = engine.Info()
	require.Equal(t, testSnapshot.snapshot.Hash, snapshotHash)
	require.EqualValues(t, testSnapshot.appState.Height, height)
	require.Equal(t, testSnapshot.appState.ChainID, chainID)

	// Attempt to load a snapshot via state-sync after the state has been restored
	// from the local storage. This should not be possible.

	response = engine.ReceiveSnapshot(testSnapshot.snapshot)
	require.Equal(t, tmtypes.ResponseOfferSnapshot{
		Result: tmtypes.ResponseOfferSnapshot_ABORT,
	}, response)

	responseForChunk := engine.ReceiveSnapshotChunk(ctx, testSnapshot.rawChunks[0], vgrand.RandomStr(5))
	require.Equal(t, tmtypes.ResponseApplySnapshotChunk{
		Result: tmtypes.ResponseApplySnapshotChunk_ABORT,
	}, responseForChunk)

	// Attempt to start the engine a second time to restore state once again
	// from the local storage. This should not be possible.

	require.Error(t, engine.Start(ctx))
}

func testRestoringStateAtSpecificBlockHeightSucceeds(t *testing.T) {
	// The snapshot to be restored via state-sync and then, from local storage.
	testSnapshotV1 := firstSnapshot(t)
	testSnapshotV2 := secondSnapshot(t)

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	vegaPaths := paths.New(t.TempDir())
	log := logging.NewTestLogger()

	// Some providers matching the snapshot payloads.
	governanceProvider := newGovernanceProvider(t, ctrl)
	delegationProvider := newDelegationProvider(t, ctrl)
	epochProvider := newEpochProvider(t, ctrl)
	statsService := mocks.NewMockStatsService(ctrl)
	timeService := mocks.NewMockTimeService(ctrl)

	config := snapshot.DefaultConfig()

	// We want to restart from the first snapshot. Proposing a more recent
	// snapshot should be rejected.
	config.StartHeight = int64(testSnapshotV1.appState.Height)
	config.RetryLimit = 5

	engine, err := snapshot.NewEngine(vegaPaths, config, log, timeService, statsService)
	require.NoError(t, err)
	closeEngine := vgtest.OnlyOnce(engine.Close)
	defer closeEngine()

	// Add the providers.
	engine.AddProviders(governanceProvider)
	engine.AddProviders(delegationProvider)
	engine.AddProviders(epochProvider)

	// From that point, start simulating state-sync.

	// Starting the engine.
	require.NoError(t, engine.Start(ctx))

	// Simulating the call to the engine from Tendermint ABCI `OfferSnapshot()`.
	// We are expecting the v1, so the v2 should be rejected.
	response := engine.ReceiveSnapshot(testSnapshotV2.snapshot)
	require.Equal(t, tmtypes.ResponseOfferSnapshot{
		Result: tmtypes.ResponseOfferSnapshot_REJECT,
	}, response)

	// Simulating the call to the engine from Tendermint ABCI `OfferSnapshot()`.
	response = engine.ReceiveSnapshot(testSnapshotV1.snapshot)
	require.Equal(t, tmtypes.ResponseOfferSnapshot{
		Result: tmtypes.ResponseOfferSnapshot_ACCEPT,
	}, response)

	// Attempting to load snapshot chunks that do not match the accepted snapshot
	// to ensure the engine rejects them.
	nodeSendingWrongChunk := vgrand.RandomStr(5)
	responseForChunk := engine.ReceiveSnapshotChunk(ctx, testSnapshotV2.rawChunks[0], nodeSendingWrongChunk)
	require.Equal(t, tmtypes.ResponseApplySnapshotChunk{
		Result:        tmtypes.ResponseApplySnapshotChunk_RETRY,
		RejectSenders: []string{nodeSendingWrongChunk},
	}, responseForChunk, "This raw chunk and its sender should be rejected")
}

func TestTakingSnapshotSucceeds(t *testing.T) {
	// The snapshot to be restored via state-sync and then, from local storage.
	testSnapshot := firstSnapshot(t)

	ctrl := gomock.NewController(t)

	vegaPaths := paths.New(t.TempDir())
	log := logging.NewTestLogger()
	ctx := vegactx.WithChainID(vegactx.WithTraceID(vegactx.WithBlockHeight(context.Background(),
		int64(testSnapshot.appState.Height)), testSnapshot.appState.Block), testSnapshot.appState.ChainID,
	)

	// Some providers matching the snapshot payloads.
	governanceProvider := newGovernanceProvider(t, ctrl)
	delegationProvider := newDelegationProvider(t, ctrl)
	epochProvider := newEpochProvider(t, ctrl)
	statsService := mocks.NewMockStatsService(ctrl)
	timeService := mocks.NewMockTimeService(ctrl)

	config := snapshot.DefaultConfig()
	// To test we keep 2 snapshots.
	config.KeepRecent = 2

	engine, err := snapshot.NewEngine(vegaPaths, config, log, timeService, statsService)
	require.NoError(t, err)
	defer engine.Close()

	// Add the providers.
	engine.AddProviders(governanceProvider)
	engine.AddProviders(delegationProvider)
	engine.AddProviders(epochProvider)

	// Starting the engine.
	require.NoError(t, engine.Start(ctx))

	// Set the snapshot interval to 20 to verify the engine only triggers the
	// snapshot at the right moment.
	require.NoError(t, engine.OnSnapshotIntervalUpdate(ctx, num.NewUint(20)))

	// Attempt to take the snapshot, 9 times, which should do nothing, as the
	// engine is set to take the snapshot every 20 blocks.
	for i := 0; i < 9; i++ {
		hash, _, err := engine.Snapshot(ctx)
		require.NoError(t, err)
		assert.Empty(t, hash)
	}

	// Decrease the snapshot interval to 10 to verify the engine only triggers the
	// snapshot at the right moment. Since the left attempts (11) are above the new
	// limit, they are reset to the new interval. So it will take 10 attempts
	// to snapshot, like the new interval.
	require.NoError(t, engine.OnSnapshotIntervalUpdate(ctx, num.NewUint(10)))

	// Attempt to take the snapshot, 9 times, which should do nothing, as the
	// engine is set to take the snapshot every 10 blocks.
	for i := 0; i < 9; i++ {
		hash, _, err := engine.Snapshot(ctx)
		require.NoError(t, err)
		assert.Empty(t, hash)
	}

	// According to the previous the configuration, the next call to snapshot
	// would have trigger the snapshot. Increase the snapshot interval to 12, this
	// should also re-peg the left attempt, by adding 2 new attempts.
	require.NoError(t, engine.OnSnapshotIntervalUpdate(ctx, num.NewUint(12)))

	// Attempt to take the snapshot, twice, which should do nothing, again, as
	// the engine is set to take the snapshot every 12 blocks.
	for i := 0; i < 2; i++ {
		hash, _, err := engine.Snapshot(ctx)
		require.NoError(t, err)
		assert.Empty(t, hash)
	}

	// Add state to the providers for next snapshot attempt.
	governanceActivePayload := testSnapshot.PayloadGovernanceActive()
	governanceEnactedPayload := testSnapshot.PayloadGovernanceEnacted()
	payloadDelegationActive := testSnapshot.PayloadDelegationActive()
	payloadEpoch := testSnapshot.PayloadEpoch()
	governanceProvider.EXPECT().GetState(governanceEnactedPayload.Key()).Return(serialize(t, governanceEnactedPayload), nil, nil).Times(1)
	governanceProvider.EXPECT().GetState(governanceActivePayload.Key()).Return(serialize(t, governanceActivePayload), nil, nil).Times(1)
	delegationProvider.EXPECT().GetState(payloadDelegationActive.Key()).Return(serialize(t, payloadDelegationActive), nil, nil).Times(1)
	epochProvider.EXPECT().GetState(payloadEpoch.Key()).Return(serialize(t, payloadEpoch), nil, nil).Times(1)
	timeService.EXPECT().GetTimeNow().Return(time.Now()).Times(1)

	// This time, the snapshot is triggered.
	hash, done, err := engine.Snapshot(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Wait for the async save to be done.
	<-done

	// Add state to the providers for next snapshot attempt.
	governanceProvider.EXPECT().GetState(governanceEnactedPayload.Key()).Return(serialize(t, governanceEnactedPayload), nil, nil).Times(1)
	governanceProvider.EXPECT().GetState(governanceActivePayload.Key()).Return(serialize(t, governanceActivePayload), nil, nil).Times(1)
	delegationProvider.EXPECT().GetState(payloadDelegationActive.Key()).Return(serialize(t, payloadDelegationActive), nil, nil).Times(1)
	epochProvider.EXPECT().GetState(payloadEpoch.Key()).Return(serialize(t, payloadEpoch), nil, nil).Times(1)
	timeService.EXPECT().GetTimeNow().Return(time.Now()).Times(1)

	// First 11 iterations as the snapshot occurs on th 12th one, as we configured
	// above.
	for i := 0; i < 11; i++ {
		hash, _, err := engine.Snapshot(ctx)
		require.NoError(t, err)
		require.Empty(t, hash)
	}

	// Take a second snapshot on the 12th iteration.
	hash, _, err = engine.Snapshot(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// Wait for the async save to be done.
	<-done

	savedSnapshots, err := engine.ListLatestSnapshots()
	require.NoError(t, err)
	assert.Len(t, savedSnapshots, 2)
}

func testProvidersSameNamespaceDifferentKeys(t *testing.T) {
	ctrl := gomock.NewController(t)

	vegaPaths := paths.New(t.TempDir())
	log := logging.NewTestLogger()

	// Some providers matching the snapshot payloads.
	statsService := mocks.NewMockStatsService(ctrl)
	timeService := mocks.NewMockTimeService(ctrl)

	engine, err := snapshot.NewEngine(vegaPaths, snapshot.DefaultConfig(), log, timeService, statsService)
	require.NoError(t, err)
	defer engine.Close()

	namespace := types.DelegationSnapshot
	key1 := vgrand.RandomStr(5)
	key2 := vgrand.RandomStr(5)
	key3 := vgrand.RandomStr(5)
	key4 := vgrand.RandomStr(5)

	provider1 := typemocks.NewMockStateProvider(ctrl)
	provider1.EXPECT().Namespace().Return(namespace).AnyTimes()
	provider1.EXPECT().Keys().Return([]string{key1, key2}).AnyTimes()
	provider1.EXPECT().Stopped().Return(false).AnyTimes()

	provider2 := typemocks.NewMockStateProvider(ctrl)
	provider2.EXPECT().Namespace().Return(namespace).AnyTimes()
	provider2.EXPECT().Keys().Return([]string{key3}).AnyTimes()
	provider2.EXPECT().Stopped().Return(false).AnyTimes()

	require.NotPanics(t, func() {
		engine.AddProviders(provider1, provider2)
	})

	// This provider reuses a key from provider1.
	provider3 := typemocks.NewMockStateProvider(ctrl)
	provider3.EXPECT().Namespace().Return(namespace).AnyTimes()
	provider3.EXPECT().Keys().Return([]string{key2, key4}).AnyTimes()
	provider3.EXPECT().Stopped().Return(false).AnyTimes()

	require.Panics(t, func() {
		engine.AddProviders(provider3)
	})
}
