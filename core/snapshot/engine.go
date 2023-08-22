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

package snapshot

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/snapshot/tree"
	"code.vegaprotocol.io/vega/core/types"
	vegactx "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	tmtypes "github.com/tendermint/tendermint/abci/types"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

const (
	namedLogger = "snapshot"
	numWorkers  = 1000

	// This is a limitation by Tendermint. It must be strictly positive, and
	// non-zero.
	maxLengthOfSnapshotList = 10
)

var ErrEngineHasAlreadyBeenStarted = errors.New("the engine has already been started")

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/snapshot TimeService,StatsService

type DoneCh <-chan interface{}

type TimeService interface {
	GetTimeNow() time.Time
	SetTimeNow(context.Context, time.Time)
}

type StatsService interface {
	SetHeight(uint64)
}

// Engine the snapshot engine.
type Engine struct {
	config Config
	log    *logging.Logger

	timeService  TimeService
	statsService StatsService

	// started tells if the snapshot engine is started or not. It is set by the
	// method `Start()`.
	// Because the snapshot engine requires providers to be registered to be useful,
	// after being initialized, it can't just load the snapshots during
	// initialization. As a result, it needs to separate the initialization
	// and start steps. This also benefits the caller, as it has full control on
	// when to start the state restoration process, like after additional setup.
	started atomic.Bool

	// snapshotLock is used when accessing the maps below while concurrently
	// constructing the new snapshot
	snapshotLock sync.RWMutex
	// registeredNamespaces holds all the namespaces of the registered providers
	// when added with the method `AddProvider()`.
	registeredNamespaces []types.SnapshotNamespace
	// namespacesToProviderKeys takes us from a namespace to the provider keys in that
	// namespace (e.g governance => {active, enacted}).
	namespacesToProviderKeys map[types.SnapshotNamespace][]string
	// namespacesToTreeKeys takes us from a namespace to the AVL tree keys in
	// that namespace (e.g governance => {governance.active, governance.enacted}).
	namespacesToTreeKeys map[types.SnapshotNamespace][][]byte
	// treeKeysToProviderKeys takes us from the key of the AVL tree node, to the provider
	// key (e.g checkpoint.all => all).
	treeKeysToProviderKeys map[string]string
	// treeKeysToProviders tracks all the components that need state to be reloaded.
	treeKeysToProviders map[string]types.StateProvider
	// preRestoreProviders tracks all the components that need to be called
	// before the state as been reloaded in all providers.
	preRestoreProviders []types.PreRestore
	// postRestoreProviders tracks all the components that need to be called
	// after the state as been reloaded in all providers.
	postRestoreProviders []types.PostRestore

	// offeredSnapshot holds the snapshot that is currently being loaded through
	// state-sync. If nil, it means there is no snapshot being loaded through
	// state-sync.
	offeredSnapshot *types.Snapshot
	// attemptsToApplySnapshotChunk counts the number of attempts made to apply
	// a chunk to a snapshot, during a state-sync. When the number of attempts
	// exceeds Config.RetryLimit, the state-sync is aborted, and the counter
	// reset.
	attemptsToApplySnapshotChunk uint
	// stateRestored tells if the state has been restored already or not. This
	// is use to guard against multiple state restoration.
	stateRestored atomic.Bool

	// loadedSnapshot holds the snapshot that is currently being used to share
	// snapshot chunks with peers on the network.
	loadedSnapshot *types.Snapshot

	// appState holds the general state of the application. It is the responsibility
	// of the snapshot engine to update, and snapshot it.
	appState *types.AppState

	// snapshotTree hold the snapshot as an AVL tree.
	snapshotTree *tree.Tree
	// snapshotTreeLock is used every time it is needed to read from or writing to
	// the AVL tree.
	snapshotTreeLock sync.Mutex

	// intervalBetweenSnapshots defines the internal between snapshots. The unit
	// is based on the network commits. An interval of 10 means a snapshot is taken
	// every 10 commits.
	intervalBetweenSnapshots uint64
	// commitsLeftBeforeSnapshot tracks the number of commits left before the
	// engine snapshots the state of the node.
	commitsLeftBeforeSnapshot uint64
}

// NewEngine returns a new snapshot engine.
func NewEngine(vegaPath paths.Paths, conf Config, log *logging.Logger, timeSvc TimeService, statsSvc StatsService) (*Engine, error) {
	if err := conf.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	appStatePayload := &types.PayloadAppState{}

	eng := &Engine{
		config: conf,
		log:    log,

		timeService:  timeSvc,
		statsService: statsSvc,

		registeredNamespaces: []types.SnapshotNamespace{},
		namespacesToProviderKeys: map[types.SnapshotNamespace][]string{
			types.AppSnapshot: {appStatePayload.Key()},
		},
		namespacesToTreeKeys: map[types.SnapshotNamespace][][]byte{
			types.AppSnapshot: {
				[]byte(types.KeyFromPayload(appStatePayload)),
			},
		},
		treeKeysToProviderKeys: map[string]string{},

		treeKeysToProviders: map[string]types.StateProvider{},
		appState:            &types.AppState{},

		// By default, a snapshot is triggered after each commit.
		intervalBetweenSnapshots: 1,
		// This must not be set to 0, otherwise a snapshot will be generated
		// right after the state has been reload, leading effectively to a
		// duplicated snapshot.
		commitsLeftBeforeSnapshot: 1,
	}

	if err := eng.initializeTree(vegaPath); err != nil {
		return nil, err
	}

	return eng, nil
}

func (e *Engine) Start(ctx context.Context) error {
	if e.started.Load() {
		return ErrEngineHasAlreadyBeenStarted
	}

	if !e.snapshotTree.HasSnapshotsLoaded() {
		e.started.Store(true)
		return nil
	}

	e.log.Info("Local snapshots found, initiating state restoration")

	if err := e.restoreStateFromTree(ctx); err != nil {
		return fmt.Errorf("could not load local snapshot: %w", err)
	}

	e.log.Info("The state has been restored", zap.Uint64("block-height", e.appState.Height))

	e.started.Store(true)

	return nil
}

func (e *Engine) OnSnapshotIntervalUpdate(_ context.Context, newIntervalBetweenSnapshots *num.Uint) error {
	newIntervalBetweenSnapshotsU := newIntervalBetweenSnapshots.Uint64()
	if newIntervalBetweenSnapshotsU < e.commitsLeftBeforeSnapshot || e.commitsLeftBeforeSnapshot == 0 {
		e.commitsLeftBeforeSnapshot = newIntervalBetweenSnapshotsU
	} else if newIntervalBetweenSnapshotsU > e.intervalBetweenSnapshots {
		e.commitsLeftBeforeSnapshot += newIntervalBetweenSnapshotsU - e.intervalBetweenSnapshots
	}
	e.intervalBetweenSnapshots = newIntervalBetweenSnapshotsU
	return nil
}

func (e *Engine) ReloadConfig(cfg Config) {
	e.log.Info("Reloading configuration")

	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("Updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}
	e.config = cfg
}

func (e *Engine) Close() {
	// Locking in case a write operation is happening on the snapshot tree,
	// before releasing.
	e.snapshotTreeLock.Lock()
	defer e.snapshotTreeLock.Unlock()

	e.snapshotTree.Release()
}

func (e *Engine) Info() ([]byte, int64, string) {
	if !e.stateRestored.Load() {
		return nil, 0, ""
	}

	return e.snapshotTree.Hash(), int64(e.appState.Height), e.appState.ChainID
}

// AddProviders add a state providers to the engine. Added providers will be called
// when a snapshot is taken, and when the state is restored.
// It supports multiple providers on the same namespace, but their generated
// tree keys (namespace + key) must be unique.
func (e *Engine) AddProviders(newProviders ...types.StateProvider) {
	e.snapshotLock.Lock()
	defer e.snapshotLock.Unlock()

	e.addProviders(newProviders...)
}

func (e *Engine) HasRestoredStateAlready() bool {
	return e.stateRestored.Load()
}

// ListLatestSnapshots list the last N snapshots in accordance to the variable
// `maxLengthOfSnapshotList`.
func (e *Engine) ListLatestSnapshots() ([]*tmtypes.Snapshot, error) {
	e.snapshotTreeLock.Lock()
	defer e.snapshotTreeLock.Unlock()

	snapshots, err := e.snapshotTree.ListLatestSnapshots(maxLengthOfSnapshotList)
	if err != nil {
		return nil, fmt.Errorf("could not list lastest snapshots: %w", err)
	}

	return snapshots, nil
}

// HasSnapshots will return whether we have snapshots, or not. This can be use to
// safely call Info().
func (e *Engine) HasSnapshots() (bool, error) {
	return e.snapshotTree.HasSnapshotsLoaded(), nil
}

// ReceiveSnapshot is called by Tendermint to restore state from state-sync.
// This must be called before load snapshot chunks.
// If this method is called while a snapshot is already being loaded, the
// current snapshot loading is aborted, and the new one is used instead.
// Proceeding as such allows Tendermint to start over when an error occurs during
// state-sync.
func (e *Engine) ReceiveSnapshot(offeredSnapshot *types.Snapshot) tmtypes.ResponseOfferSnapshot {
	e.ensureEngineIsStarted()

	if e.stateRestored.Load() {
		e.log.Error("Attempt to offer a snapshot whereas the state has already been restored, aborting offer")
		return tmtypes.ResponseOfferSnapshot{
			Result: tmtypes.ResponseOfferSnapshot_ABORT,
		}
	}

	e.log.Info("New snapshot received from state-sync",
		zap.Uint64("snapshot-height", offeredSnapshot.Height),
	)

	if e.config.StartHeight > 0 && offeredSnapshot.Height != uint64(e.config.StartHeight) {
		e.log.Info("The block height of the received snapshot does not match the expected one, rejecting offer",
			zap.Uint64("snapshot-height", offeredSnapshot.Height),
			zap.Int64("expected-height", e.config.StartHeight),
		)
		return tmtypes.ResponseOfferSnapshot{
			Result: tmtypes.ResponseOfferSnapshot_REJECT,
		}
	}

	// If Tendermint fails to fetch a chunk after some time, it will reject the
	// snapshot and try a different one via `OfferSnapshot()`. Therefore, the
	// ongoing snapshot process must be reset.
	if e.offeredSnapshot != nil {
		e.log.Warn("Resetting the process loading from state-sync to accept the new one")
		e.resetOfferedSnapshot()
	}

	e.offeredSnapshot = offeredSnapshot

	e.log.Info("New snapshot received from state-sync accepted",
		zap.Uint64("snapshot-height", offeredSnapshot.Height),
	)

	return tmtypes.ResponseOfferSnapshot{
		Result: tmtypes.ResponseOfferSnapshot_ACCEPT,
	}
}

// ReceiveSnapshotChunk is called by Tendermint to restore state from state-sync.
// It receives the chunks matching the snapshot received via the `ReceiveSnapshot()`.
func (e *Engine) ReceiveSnapshotChunk(ctx context.Context, chunk *types.RawChunk, sender string) tmtypes.ResponseApplySnapshotChunk {
	e.ensureEngineIsStarted()

	if e.stateRestored.Load() {
		e.log.Error("Attempt to load snapshot chunks whereas the state has already been loaded from local snapshots, aborting state-sync")
		return e.abortStateSync()
	}

	e.log.Info("New snapshot chunk received from state-sync",
		zap.Uint32("chunk-number", chunk.Nr),
	)

	if e.offeredSnapshot == nil {
		// If this condition is valid, it means the engine is tasked to load
		// snapshot chunks, without being offered one, first. It does not seem
		// Tendermint will ever do that, according to the ABCI documentation, so
		// it seems it would all come down to a programming error.
		// However, prior the refactoring, this was interpreted as "not being
		// ready". The reason remains obscure.
		// Panicking would probably be the best thing to do.
		// In the meantime, we should monitor the error messages, and see if it's
		// ever happening, to know if we can safely remove it.
		e.log.Error("Attempt to load snapshot chunks without offering a snapshot first, this should not have happened, aborting state-sync")
		return tmtypes.ResponseApplySnapshotChunk{
			Result: tmtypes.ResponseApplySnapshotChunk_RETRY_SNAPSHOT,
		}
	}

	if err := e.offeredSnapshot.LoadChunk(chunk); err != nil {
		if errors.Is(err, types.ErrChunkOutOfRange) {
			if e.shouldAbortStateSync() {
				e.log.Error("Engine reached the maximum number of retry for loading snapshot chunk, aborting state-sync", logging.Error(err))
				return e.abortStateSync()
			} else {
				e.log.Warn("Reject offered snapshot as received chunk does not match",
					zap.String("sender", sender),
					logging.Error(err),
				)
				return tmtypes.ResponseApplySnapshotChunk{
					Result: tmtypes.ResponseApplySnapshotChunk_REJECT_SNAPSHOT,
				}
			}
		} else if errors.Is(err, types.ErrMissingChunks) {
			if e.shouldAbortStateSync() {
				e.log.Error("Engine reached the maximum number of retry for loading snapshot chunk, aborting state-sync", logging.Error(err))
				return e.abortStateSync()
			} else {
				e.log.Warn("Snapshot is missing chunks, retrying",
					logging.Error(err),
				)
				return tmtypes.ResponseApplySnapshotChunk{
					Result:        tmtypes.ResponseApplySnapshotChunk_RETRY,
					RefetchChunks: e.offeredSnapshot.MissingChunks(),
				}
			}
		} else if errors.Is(err, types.ErrChunkHashMismatch) {
			if e.shouldAbortStateSync() {
				e.log.Error("Engine reached the maximum number of retry for loading snapshot chunk, aborting state-sync", logging.Error(err))
				return e.abortStateSync()
			} else {
				e.log.Warn("Received chunk is not consistent with metadata from the offered snapshot, rejecting sender and retrying",
					zap.String("rejected-sender", sender),
					logging.Error(err),
				)
				return tmtypes.ResponseApplySnapshotChunk{
					Result:        tmtypes.ResponseApplySnapshotChunk_RETRY,
					RejectSenders: []string{sender},
				}
			}
		}

		e.log.Error("An error occurred while loading chunk in the snapshot during state-sync, aborting state-sync",
			logging.Uint32("chunk-number", chunk.Nr),
			logging.Error(err),
		)
		return e.abortStateSync()
	}

	e.log.Info("New snapshot chunk received from state-sync accepted",
		zap.Uint32("chunk-number", chunk.Nr),
	)

	if !e.offeredSnapshot.Ready() {
		return tmtypes.ResponseApplySnapshotChunk{
			Result: tmtypes.ResponseApplySnapshotChunk_ACCEPT,
		}
	}

	e.log.Info("All snapshot chunks received, initiating state restoration")

	// Saving snapshot in the tree. At this point, this should be the only snapshot
	// it has, as restoring state from state-sync require an empty snapshot
	// database, and thus, an empty tree.
	e.snapshotTreeLock.Lock()
	defer e.snapshotTreeLock.Unlock()

	if err := e.snapshotTree.AddSnapshot(e.offeredSnapshot); err != nil {
		e.log.Error("Could not add offered snapshot to the tree, aborting state-sync", logging.Error(err))
		return e.abortStateSync()
	}

	if err := e.restoreStateFromSnapshot(ctx, e.offeredSnapshot.Nodes); err != nil {
		e.log.Error("Could not restore state, aborting state-sync", logging.Error(err))
		return e.abortStateSync()
	}

	// The state has been successfully restored, so resources are released.
	e.resetOfferedSnapshot()

	e.log.Info("The state has been restored")

	return tmtypes.ResponseApplySnapshotChunk{
		Result: tmtypes.ResponseApplySnapshotChunk_ACCEPT,
	}
}

// RetrieveSnapshotChunk is called by Tendermint to retrieve a snapshot chunk
// to help a peer node to restore its state from state-sync.
func (e *Engine) RetrieveSnapshotChunk(height uint64, format, chunkIndex uint32) (*types.RawChunk, error) {
	if e.loadedSnapshot == nil || height != e.loadedSnapshot.Height {
		loadedSnapshot, err := e.findTreeByBlockHeight(height)
		if err != nil {
			return nil, err
		}
		e.loadedSnapshot = loadedSnapshot
	}

	expectedFormat, err := types.SnapshotFormatFromU32(format)
	if err != nil {
		return nil, fmt.Errorf("could not deserialize snapshot format: %w", err)
	}

	if expectedFormat != e.loadedSnapshot.Format {
		return nil, types.ErrSnapshotFormatMismatch
	}

	if e.loadedSnapshot.Chunks == chunkIndex {
		// The network is asking for the last chunk of the loaded snapshot.
		// Let's free up some memory.
		defer func() {
			e.loadedSnapshot = nil
		}()
	}

	return e.loadedSnapshot.RawChunkByIndex(chunkIndex)
}

// Snapshot triggers the snapshot process at defined interval. Do nothing if the
// the interval bound is not reached.
func (e *Engine) Snapshot(ctx context.Context) ([]byte, DoneCh, error) {
	e.ensureEngineIsStarted()

	e.commitsLeftBeforeSnapshot--

	if e.commitsLeftBeforeSnapshot > 0 {
		return nil, nil, nil
	}

	e.commitsLeftBeforeSnapshot = e.intervalBetweenSnapshots

	return e.snapshotNow(ctx, true)
}

// SnapshotNow triggers the snapshot process right now, ignoring the defined
// interval.
func (e *Engine) SnapshotNow(ctx context.Context) ([]byte, error) {
	e.ensureEngineIsStarted()

	now, _, err := e.snapshotNow(ctx, false)

	return now, err
}

func (e *Engine) snapshotNow(ctx context.Context, saveAsync bool) ([]byte, DoneCh, error) {
	defer metrics.StartSnapshot("all")()
	e.snapshotTreeLock.Lock()

	// When a node requests a snapshot, it means it holds state, regardless
	// it reloaded from a snapshot or not. Therefore, the engine must mark
	// the state as restored to ensure it won't have multiple state restorations.
	e.stateRestored.Store(true)

	treeKeysToSnapshot := make([]treeKeyToSnapshot, 0, len(e.registeredNamespaces))
	for _, namespace := range e.registeredNamespaces {
		// FIXME: This metric is not used the way it has been thought.
		//  See https://github.com/vegaprotocol/vega/issues/8775
		defer metrics.StartSnapshot(namespace.String())()

		treeKeys := e.namespacesToTreeKeys[namespace]

		for _, treeKey := range treeKeys {
			treeKeysToSnapshot = append(treeKeysToSnapshot, treeKeyToSnapshot{treeKey: treeKey, namespace: namespace})
		}
	}

	treeKeysCounter := atomic.Int64{}
	treeKeysCounter.Store(int64(len(treeKeysToSnapshot)))

	treeKeysToSnapshotChan := make(chan treeKeyToSnapshot, numWorkers)
	serializedStateChan := make(chan snapshotResult, numWorkers)

	// Start the gathering of providers state asynchronously.
	wg := &sync.WaitGroup{}
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			gatherState(e, treeKeysToSnapshotChan, serializedStateChan, &treeKeysCounter)
			wg.Done()
		}()
	}

	for _, treeKeyToSnapshot := range treeKeysToSnapshot {
		treeKeysToSnapshotChan <- treeKeyToSnapshot
	}

	if len(treeKeysToSnapshot) == 0 {
		close(treeKeysToSnapshotChan)
		close(serializedStateChan)
	}

	// analyse the results
	results := make([]snapshotResult, 0, numWorkers)
	for res := range serializedStateChan {
		if res.err != nil {
			e.log.Panic("Failed to update snapshot namespace",
				logging.String("snapshot-namespace", res.input.namespace.String()),
				logging.Error(res.err),
			)
		}
		results = append(results, res)
	}

	// wait for all workers to complete
	wg.Wait()

	// all results are int - split them by namespace first
	updated := false
	resultByTreeKey := make(map[string]snapshotResult, len(results))
	for _, tkRes := range results {
		resultByTreeKey[string(tkRes.input.treeKey)] = tkRes
	}

	for _, ns := range e.registeredNamespaces {
		treeKeys, ok := e.namespacesToTreeKeys[ns]
		if !ok {
			continue
		}

		// sort the tree keys because providers may be added in a random order
		sort.Slice(treeKeys, func(i, j int) bool { return string(treeKeys[i]) < string(treeKeys[j]) })

		toRemove := []int{}
		for i, treeKey := range treeKeys {
			snapshotRes, ok := resultByTreeKey[string(treeKey)]
			if !ok {
				continue
			}

			if !snapshotRes.updated || snapshotRes.toRemove {
				if snapshotRes.toRemove {
					toRemove = append(toRemove, i)
				}
				continue
			}

			e.log.Info("State updated", logging.String("tree-key", string(treeKey)))

			if len(snapshotRes.state) == 0 {
				// empty state -> remove data from snapshot
				updated = e.snapshotTree.RemoveKey(treeKey)
				continue
			}
			e.snapshotTree.AddState(treeKey, snapshotRes.state)
			updated = true
		}

		if len(toRemove) == 0 {
			continue
		}

		for ind := len(toRemove) - 1; ind >= 0; ind-- {
			i := toRemove[ind]
			tk := treeKeys[i]
			tkRes, ok := resultByTreeKey[string(tk)]
			if !ok {
				continue
			}
			updated = true
			treeKey := tkRes.input.treeKey
			treeKeyStr := string(treeKey)

			// delete everything we've got stored
			e.log.Debug("State to be removed", logging.String("tree-key", treeKeyStr))
			delete(e.treeKeysToProviders, treeKeyStr)
			delete(e.treeKeysToProviderKeys, treeKeyStr)

			if !e.snapshotTree.RemoveKey(treeKey) {
				e.log.Panic("failed to remove node from AVL tree", logging.String("key", treeKeyStr))
			}

			e.namespacesToTreeKeys[ns] = append(e.namespacesToTreeKeys[ns][:i], e.namespacesToTreeKeys[ns][i+1:]...)
		}
	}

	// update appstate separately
	appUpdate := false
	height, err := vegactx.BlockHeightFromContext(ctx)
	if err != nil {
		e.snapshotTreeLock.Unlock()
		return nil, nil, err
	}
	if height != int64(e.appState.Height) {
		appUpdate = true
		e.appState.Height = uint64(height)
	}
	_, block := vegactx.TraceIDFromContext(ctx)
	if block != e.appState.Block {
		appUpdate = true
		e.appState.Block = block
	}
	vNow := e.timeService.GetTimeNow().UnixNano()
	if e.appState.Time != vNow {
		e.appState.Time = vNow
		appUpdate = true
	}

	cid, err := vegactx.ChainIDFromContext(ctx)
	if err != nil {
		e.snapshotTreeLock.Unlock()
		return nil, nil, err
	}
	if e.appState.ChainID != cid {
		e.appState.ChainID = cid
		appUpdate = true
	}

	if appUpdate {
		if err = e.updateAppState(); err != nil {
			e.snapshotTreeLock.Unlock()
			return nil, nil, err
		}
		updated = true
	}

	if !updated {
		hash := e.snapshotTree.Hash()
		e.snapshotTreeLock.Unlock()
		return hash, nil, nil
	}

	doneCh := make(chan interface{})

	save := func() {
		defer func() {
			close(doneCh)

			if r := recover(); r != nil {
				e.log.Panic("Panic occurred", zap.Any("reason", r))
			}
		}()

		if err := e.snapshotTree.SaveVersion(); err != nil {
			// If this fails, we are screwed. The tree version is used to construct
			// the root-hash so if we can't save it, the next snapshot we take
			// will mismatch so we need to fail hard here.
			e.log.Panic("Could not save the snapshot tree", logging.Error(err))
		}
	}

	var hash []byte
	if saveAsync {
		// Using the working hash instead of the hash computed on save. This is an
		// "optimistic" hack, that comes from the fact the tree is saved asynchronously.
		// As a result, the hash for the working version of the tree is not computed
		// yet. So, using calling `Hash()` will either return an empty hash (when first tree),
		// either is returns the one from the previous version.
		// Therefore, we have to use the working hash. It shouldn't be a problem
		// as long as the tree is not modified past this point (this is the
		// "optimistic" part). In the end, the hashes should match.
		hash = e.snapshotTree.WorkingHash()
		go func() {
			save()
			e.snapshotTreeLock.Unlock()
		}()
	} else {
		save()
		hash = e.snapshotTree.Hash()
		e.snapshotTreeLock.Unlock()
	}

	e.log.Info("Snapshot taken",
		logging.Int64("height", height),
		logging.ByteString("hash", hash),
	)

	return hash, doneCh, nil
}

func (e *Engine) updateAppState() error {
	keys, ok := e.namespacesToTreeKeys[types.AppSnapshot]
	if !ok {
		return types.ErrNoPrefixFound
	}
	// there should be only 1 entry here
	if len(keys) > 1 || len(keys) == 0 {
		return types.ErrUnexpectedKey
	}

	pl := types.Payload{
		Data: &types.PayloadAppState{
			AppState: e.appState,
		},
	}

	data, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return fmt.Errorf("could not serialize the payload to proto: %w", err)
	}

	e.snapshotTree.AddState(keys[0], data)
	return nil
}

func (e *Engine) findTreeByBlockHeight(height uint64) (*types.Snapshot, error) {
	e.snapshotTreeLock.Lock()
	defer e.snapshotTreeLock.Unlock()

	immutableTree, err := e.snapshotTree.FindImmutableTreeByHeight(height)
	if err != nil {
		return nil, fmt.Errorf("could not find snapshot associated to block height %d: %w", height, err)
	}

	loadedSnapshot, err := types.SnapshotFromTree(immutableTree)
	if err != nil {
		return nil, fmt.Errorf("could not convert tree into snapshot: %w", err)
	}

	return loadedSnapshot, nil
}

func (e *Engine) shouldAbortStateSync() bool {
	e.attemptsToApplySnapshotChunk++

	return e.attemptsToApplySnapshotChunk >= e.config.RetryLimit
}

func (e *Engine) abortStateSync() tmtypes.ResponseApplySnapshotChunk {
	e.resetOfferedSnapshot()

	return tmtypes.ResponseApplySnapshotChunk{
		Result: tmtypes.ResponseApplySnapshotChunk_ABORT,
	}
}

func (e *Engine) resetOfferedSnapshot() {
	e.offeredSnapshot = nil
	e.attemptsToApplySnapshotChunk = 0
}

func (e *Engine) restoreStateFromTree(ctx context.Context) error {
	e.snapshotTreeLock.Lock()
	defer e.snapshotTreeLock.Unlock()

	lastSnapshotPayloads, err := e.snapshotTree.AsPayloads()
	if err != nil {
		return fmt.Errorf("could not generate the immutable AVL tree: %w", err)
	}

	if err := e.restoreStateFromSnapshot(ctx, lastSnapshotPayloads); err != nil {
		return fmt.Errorf("could not restore the state from the local snapshot: %w", err)
	}

	return nil
}

func (e *Engine) restoreStateFromSnapshot(ctx context.Context, payloads []*types.Payload) error {
	payloadsPerNamespace := groupPayloadsPerNamespace(payloads)

	// The snapshot engine is responsible of snapshotting the general state
	// of the node.
	e.appState = payloadsPerNamespace[types.AppSnapshot][0].GetAppState().AppState

	// These values are needed in the context by providers, to send events.
	ctx = vegactx.WithTraceID(vegactx.WithBlockHeight(ctx, int64(e.appState.Height)), e.appState.Block)
	ctx = vegactx.WithChainID(ctx, e.appState.ChainID)

	// Restoring state in globally shared services.
	e.timeService.SetTimeNow(ctx, time.Unix(0, e.appState.Time))
	e.statsService.SetHeight(e.appState.Height)

	// Calling providers that need to be called before restoring their state.
	for _, provider := range e.preRestoreProviders {
		if err := provider.OnStateLoadStarts(ctx); err != nil {
			return fmt.Errorf("an error occurred on provider %q during snapshot pre-restoration: %w", provider.Namespace(), err)
		}
	}

	// Restoring state in providers.
	for _, namespace := range providersInCallOrder {
		for _, payload := range payloadsPerNamespace[namespace] {
			provider, ok := e.treeKeysToProviders[payload.TreeKey()]
			if !ok {
				return fmt.Errorf("%w %s", types.ErrUnknownSnapshotNamespace, payload.TreeKey())
			}

			e.log.Info("Restoring state in provider", logging.String("tree-key", payload.TreeKey()))

			newProviders, err := provider.LoadState(ctx, payload)
			if err != nil {
				return fmt.Errorf("an error occurred on provider %q while restoring state on tree-key %q: %w", provider.Namespace(), payload.TreeKey(), err)
			}

			// Some providers depend on resources that also need to have their
			// state restored. Therefore, we add them, on the fly, to the existing
			// provider list.
			if len(newProviders) != 0 {
				e.addProviders(newProviders...)
			}
		}
	}

	// Calling providers that need to be called after their state has been restored.
	for _, provider := range e.postRestoreProviders {
		if err := provider.OnStateLoaded(ctx); err != nil {
			return fmt.Errorf("an error occurred on provider %q during snapshot post-restoration: %w", provider.Namespace(), err)
		}
	}

	e.stateRestored.Store(true)

	return nil
}

func (e *Engine) initializeTree(vegaPaths paths.Paths) error {
	var storageOption tree.Options
	switch e.config.Storage {
	case InMemoryDB:
		storageOption = tree.WithInMemoryDatabase()
	case LevelDB:
		storageOption = tree.WithLevelDBDatabase(vegaPaths)
	default:
		return types.ErrInvalidSnapshotStorageMethod
	}

	snapshotTree, err := tree.New(
		e.log,
		tree.WithMaxNumberOfSnapshotsToKeep(uint64(e.config.KeepRecent)),
		tree.StartingAtBlockHeight(uint64(e.config.StartHeight)),
		storageOption,
	)
	if err != nil {
		return fmt.Errorf("could not initialize the snapshot tree: %w", err)
	}
	e.snapshotTree = snapshotTree

	return nil
}

func (e *Engine) addProviders(newProviders ...types.StateProvider) {
	for _, newProvider := range newProviders {
		newKeys := newProvider.Keys()
		namespace := newProvider.Namespace()

		if !slices.Contains(providersInCallOrder, namespace) {
			// This is a programming error that can happen when introducing a new
			// provider. All namespaces must be listed to the list providersInCallOrder,
			// otherwise their state won't be restored, as the state restoration
			// iterates through this list, is strict order, to know in which order
			// state must be restored.
			e.log.Panic(fmt.Sprintf("The provider %q is not listed in the sorted provider list", namespace))
		}

		existingKeys, ok := e.namespacesToProviderKeys[namespace]
		if !ok {
			e.namespacesToTreeKeys[namespace] = make([][]byte, 0, len(newKeys))
			e.registeredNamespaces = append(e.registeredNamespaces, namespace)
		}

		duplicatedKeys := findDuplicatedKeys(existingKeys, newKeys)
		if len(duplicatedKeys) > 0 {
			// This is a programming error that might happen when adding a
			// provider to the codebase, during an .
			e.log.Panic("A state provider in the same namespace is already using these keys",
				zap.String("namespace", namespace.String()),
				zap.Any("keys", duplicatedKeys),
				zap.String("culprit", reflect.TypeOf(newProvider).String()),
			)
		}

		e.namespacesToProviderKeys[namespace] = append(e.namespacesToProviderKeys[namespace], newKeys...)
		for _, newKey := range newKeys {
			treeKey := types.GetNodeKey(namespace, newKey)
			e.treeKeysToProviderKeys[treeKey] = newKey
			e.treeKeysToProviders[treeKey] = newProvider
			e.namespacesToTreeKeys[namespace] = append(e.namespacesToTreeKeys[namespace], []byte(treeKey))
		}

		if p, ok := newProvider.(types.PostRestore); ok {
			e.postRestoreProviders = append(e.postRestoreProviders, p)
		}
		if p, ok := newProvider.(types.PreRestore); ok {
			e.preRestoreProviders = append(e.preRestoreProviders, p)
		}
	}
}

func (e *Engine) ensureEngineIsStarted() {
	if !e.started.Load() {
		// This is a programming error.
		e.log.Panic("The snapshot engine has not started!")
	}
}

func findDuplicatedKeys(existingKeys, newKeys []string) []string {
	duplicatedKeys := []string{}
	for _, newKey := range newKeys {
		if slices.Contains(existingKeys, newKey) {
			duplicatedKeys = append(duplicatedKeys, newKey)
		}
	}
	return duplicatedKeys
}
