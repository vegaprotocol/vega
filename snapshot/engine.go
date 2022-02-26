package snapshot

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/shared/paths"
	vegactx "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/types"

	"github.com/cosmos/iavl"
	"github.com/golang/protobuf/proto"
	"github.com/tendermint/tendermint/libs/strings"
	db "github.com/tendermint/tm-db"
)

type SnapshotSource int

const SnapshotDBName = "snapshot"

type StateProviderT interface {
	// Namespace this provider operates in, basically a prefix for the keys
	Namespace() types.SnapshotNamespace
	// Keys gets all the nodes this provider populates
	Keys() []string
	// HasChanged should return true if state for a given key was updated
	HasChanged(key string) bool
	// GetState returns the new state as a payload type
	GetState(key string) *types.Payload
	// PollChanges waits for an update on a channel - if nothing was updated, then nil can be sent
	// we can call this at the end of a block, so the engines have time until commit to provide the data
	// rather than a series of blocking calls
	PollChanges(ctx context.Context, k string, ch chan<- *types.Payload)
	// Sync is called when polling for changes, but we need the snapshot data now. Similar to wg.Wait()
	// on all of the state providers
	Sync() error
	// Err is called if the provider sent nil on the poll channel. Return nil if all was well (just no changes)
	// or the relevant error if something failed. The same error can be returned when calling Sync()
	Err() error

	// LoadState is called to set the state once again, has to return state providers
	// in case a new engine is created in the process (e.g. execution engine creating markets, with positions and matching engines)
	LoadState(ctx context.Context, pl *types.Payload) ([]types.StateProvider, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_mock.go -package mocks code.vegaprotocol.io/vega/snapshot TimeService
type TimeService interface {
	GetTimeNow() time.Time
	SetTimeNow(context.Context, time.Time)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/stats_mock.go -package mocks code.vegaprotocol.io/vega/snapshot StatsService
type StatsService interface {
	SetHeight(uint64)
}

// Engine the snapshot engine.
type Engine struct {
	Config
	log *logging.Logger

	ctx          context.Context
	cfunc        context.CancelFunc
	timeService  TimeService
	statsService StatsService
	db           db.DB
	dbPath       string

	avl             *iavl.MutableTree
	namespaces      []types.SnapshotNamespace
	nsKeys          map[types.SnapshotNamespace][]string // takes us from a namespace to the provider keys in that namespace e.g governance => {active, enacted}
	nsTreeKeys      map[types.SnapshotNamespace][][]byte // takes us from a namespace to the AVL tree keys in that namespace e.g governanec => {governance.active, governance.enacted}
	treeKeyProvider map[string]string                    // takes us from the key of the AVL tree node, to the provider key e.g checkpoint.all => all
	keyHashes       map[string][]byte                    // takes us from the key of the AVL tree, to the last known hash of that node
	versions        []int64
	interval        int64
	current         int64

	providers          map[string]types.StateProvider
	restoreProvs       []types.PostRestore
	beforeRestoreProvs []types.PreRestore
	providersNS        map[types.SnapshotNamespace][]types.StateProvider

	last             *iavl.ImmutableTree
	lastSnapshotHash []byte // the root hash of the last snapshot that was taken
	versionHeight    map[uint64]int64

	snapshot  *types.Snapshot
	snapRetry int

	// the general snapshot info this engine is responsible for
	wrap *types.PayloadAppState
	app  *types.AppState

	// unused bit related to experiemental channel based snapshot update stuff?
	providerTS map[string]StateProviderT
	pollCtx    context.Context
	pollCfunc  context.CancelFunc
}

// order in which snapshots are to be restored.
var nodeOrder = []types.SnapshotNamespace{
	types.AppSnapshot,
	types.AssetsSnapshot,
	types.WitnessSnapshot, // needs to happen before banking and governance
	types.GovernanceSnapshot,
	types.BankingSnapshot,
	types.CollateralSnapshot,
	types.NotarySnapshot,
	types.NetParamsSnapshot,
	types.CheckpointSnapshot,
	types.DelegationSnapshot,
	types.FloatingPointConsensusSnapshot, // shouldn't matter but maybe best before the markets are restored
	types.ExecutionSnapshot,              // creates the markets, returns matching and positions engines for state providers
	types.MatchingSnapshot,               // this requires a market
	types.PositionsSnapshot,              // again, needs a market
	types.LiquiditySnapshot,
	types.LiquidityTargetSnapshot,
	types.EpochSnapshot,
	types.StakingSnapshot,
	types.StakeVerifierSnapshot,
	types.SpamSnapshot,
	types.LimitSnapshot,
	types.ReplayProtectionSnapshot,
	types.RewardSnapshot,
	types.TopologySnapshot,
	types.EventForwarderSnapshot,
	types.FeeTrackerSnapshot,
	types.MarketTrackerSnapshot,
}

// New returns a new snapshot engine.
func New(ctx context.Context, vegapath paths.Paths, conf Config, log *logging.Logger, tm TimeService, stats StatsService) (*Engine, error) {
	// default to min 1 version, just so we don't have to account for negative cap or nil slice.
	// A single version kept in memory is pretty harmless.
	if conf.KeepRecent < 1 {
		conf.KeepRecent = 1
	}
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	dbPath, err := conf.validate(vegapath)
	if err != nil {
		return nil, err
	}

	sctx, cfunc := context.WithCancel(ctx)
	appPL := &types.PayloadAppState{
		AppState: &types.AppState{},
	}
	app := appPL.Namespace()
	eng := &Engine{
		Config:       conf,
		log:          log,
		ctx:          sctx,
		cfunc:        cfunc,
		timeService:  tm,
		statsService: stats,
		dbPath:       dbPath,
		namespaces:   []types.SnapshotNamespace{},
		nsKeys: map[types.SnapshotNamespace][]string{
			app: {appPL.Key()},
		},
		nsTreeKeys: map[types.SnapshotNamespace][][]byte{
			app: {
				[]byte(types.KeyFromPayload(appPL)),
			},
		},
		treeKeyProvider: map[string]string{},
		keyHashes:       map[string][]byte{},
		providers:       map[string]types.StateProvider{},
		providersNS:     map[types.SnapshotNamespace][]types.StateProvider{},
		versions:        make([]int64, 0, conf.KeepRecent), // cap determines how many versions we keep
		versionHeight:   map[uint64]int64{},
		wrap:            appPL,
		app:             appPL.AppState,
		interval:        1, // default to every block
		current:         1,
	}
	return eng, nil
}

func (e *Engine) ReloadConfig(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}
	e.Config = cfg
}

// List returns all snapshots available.
func (e *Engine) List() ([]*types.Snapshot, error) {
	snapshots := make([]*types.Snapshot, 0, len(e.versions))
	// TM list of snapshots is limited to the 10 most recent ones.
	i := len(e.versions) - 11
	if i < 0 {
		i = 0
	}
	for j := len(e.versions); i < j; i++ {
		v := e.versions[i]
		tree, err := e.avl.GetImmutable(v)
		if err != nil {
			return nil, err
		}
		snap, err := types.SnapshotFromTree(tree)
		if err != nil {
			e.log.Error("could not list snapshot",
				logging.Int64("version", v),
				logging.Error(err))
			continue // if we have a borked snapshot we just won't list it
		}
		snapshots = append(snapshots, snap)
		e.versionHeight[snap.Height] = snap.Meta.Version
	}
	return snapshots, nil
}

// Start kicks the snapshot engine into its initial state setting up the DB connections
// and ensuring any pre-existing snapshot database is removed first. It is to be called
// by a chain that is starting from block 0.
func (e *Engine) Start() error {
	p := filepath.Join(e.dbPath, SnapshotDBName+".db")

	exists, err := vgfs.PathExists(p)
	if err != nil {
		return err
	}

	if exists {
		e.log.Warn("removing old snapshot data", logging.String("dbpath", p))
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}

	return e.initialiseTree()
}

// this function loads snapshots in local store as though they were generated at runtime.
// The result is that, whenever we create a new snapshot, the old one gets cleaned up.
func (e *Engine) populateLocalVersions(versions []int) {
	// is in ascending order already, so let's just iterate
	if len(versions) == 0 {
		versions = e.avl.AvailableVersions()
	}
	vc := cap(e.versions)
	for _, v := range versions {
		if len(e.versions) >= vc {
			if err := e.avl.DeleteVersion(e.versions[0]); err != nil {
				e.log.Warn("Could not delete an old version",
					logging.Int64("old-version", e.versions[0]),
					logging.Error(err),
				)
			}
			// still, we should drop this from the slice
			copy(e.versions[0:], e.versions[1:])
			e.versions[len(e.versions)-1] = int64(v)
		} else {
			e.versions = append(e.versions, int64(v))
		}
	}
}

// Loaded will return whether we have loaded from a snapshot. If we have loaded
// via stat-sync we will already know, if we are loading from local store then we do that
// node.
func (e *Engine) Loaded() (bool, error) {
	// if the avl has been initialised we must have loaded it earlier via using state-sync
	// we can go straight into loading the state into the providers
	if e.avl != nil {
		// OK, but let's make the engine aware of its local store versions
		e.populateLocalVersions(nil)
		return true, e.applySnap(e.ctx)
	}

	startHeight := e.Config.StartHeight
	if startHeight == 0 {
		// starting a new chain or replaying, not loading snapshot
		return false, nil
	}

	e.log.Debug("loading snapshot for height", logging.Int64("height", startHeight))
	// setup AVL tree from local store
	e.initialiseTree()
	if startHeight < 0 {
		return true, e.load(e.ctx)
	}

	height := uint64(startHeight)
	versions := e.avl.AvailableVersions()
	e.populateLocalVersions(versions)
	// descending order, because that makes most sense
	var last, first uint64
	for i := len(versions) - 1; i > -1; i-- {
		version := int64(versions[i])
		if _, err := e.avl.LoadVersion(version); err != nil {
			return false, err
		}
		app, err := types.AppStateFromTree(e.avl.ImmutableTree)
		if err != nil {
			e.log.Error("Failed to get app state data from snapshot",
				logging.Error(err),
				logging.Int64("snapshot-version", version),
			)
			continue
		}
		if app.AppState.Height == height {
			e.last = e.avl.ImmutableTree
			return true, e.load(e.ctx)
		}
		// we've gone past the specified height, we're not going to find the snapshot
		// log and error
		if app.AppState.Height < height {
			e.log.Error("Unable to find a snapshot for the specified height",
				logging.Uint64("snapshot-height", height),
				logging.Uint64("max-height", first),
			)
			return false, types.ErrNoSnapshot
		}
		last = app.AppState.Height
		if first == 0 {
			first = last
		}
	}
	e.log.Error("Specified height too low",
		logging.Uint64("specified-height", height),
		logging.Uint64("maximum-height", first),
		logging.Uint64("minimum-height", last),
	)
	return false, types.ErrNoSnapshot
}

// initialiseTree connects to the snapshotdb and sets the engine's state to
// point to the latest version of the tree.
func (e *Engine) initialiseTree() error {
	switch e.Config.Storage {
	case memDB:
		e.db = db.NewMemDB()
	case goLevelDB:
		conn, _ := db.NewGoLevelDB(SnapshotDBName, e.dbPath)
		e.db = conn
	default:
		return types.ErrInvalidSnapshotStorageMethod
	}

	tree, err := iavl.NewMutableTree(e.db, 0)
	if err != nil {
		e.log.Error("Could not create AVL tree", logging.Error(err))
		return err
	}

	e.avl = tree
	// Either create the first empty tree, or load the latest tree we have in the store
	if err := e.loadTree(); err != nil {
		e.log.Error("Failed to load AVL version", logging.Error(err))
		return err
	}
	return nil
}

func (e *Engine) loadTree() error {
	if _, err := e.avl.Load(); err != nil {
		return err
	}
	e.last = e.avl.ImmutableTree
	return nil
}

func (e *Engine) load(ctx context.Context) error {
	snap, err := types.SnapshotFromTree(e.last)
	if err != nil {
		return err
	}
	e.snapshot = snap
	// apply, no need to set the tree, it's coming from local store
	return e.applySnap(ctx)
}

func (e *Engine) ReceiveSnapshot(snap *types.Snapshot) error {
	if e.snapshot != nil {
		// in case other peers provide snapshots, check if their hashes match what we want
		if !bytes.Equal(e.snapshot.Hash, snap.Hash) {
			return types.ErrSnapshotHashMismatch
		}
		return e.snapshot.ValidateMeta(snap)
	}
	// @TODO here's where we check the hash or height we want
	e.snapshot = snap
	return nil
}

func (e *Engine) RejectSnapshot() error {
	e.snapRetry++
	if e.RetryLimit < e.snapRetry {
		return types.ErrSnapshotRetryLimit
	}
	if e.snapshot == nil {
		return types.ErrUnknownSnapshot
	}
	e.snapshot = nil
	return nil
}

// ApplySnapshot takes the snapshot data sent over via tendermint and reconstructs the AVL
// tree from the data. This call does *not* restore the state into the providers.
func (e *Engine) ApplySnapshot(ctx context.Context) error {
	// remove all existing snapshot and create an initial empty tree
	e.Start()

	// Import the AVL tree from the snapshot data so we have a working copy
	// that is consistent with the other nodes
	if err := e.snapshot.TreeFromSnapshot(e.avl); err != nil {
		e.log.Error("failed to recreate tree", logging.Error(err))
		return err
	}

	return nil
	// Load the snapshot data into each provider
	// return e.applySnap(ctx)
}

func (e *Engine) applySnap(ctx context.Context) error {
	if e.snapshot == nil {
		return types.ErrUnknownSnapshot
	}
	// this is the current version
	version := e.snapshot.Meta.Version
	loaded, err := e.avl.LoadVersionForOverwriting(version)
	if err != nil {
		e.log.Error("Failed to load target version",
			logging.Error(err),
			logging.Int64("loaded-version", loaded),
		)
	}
	// we need the versions of the snapshot to match, regardless of the version we actually loaded
	e.avl.SetInitialVersion(uint64(version))
	// now let's clear the versions slice and pretend the more recent versions don't exist yet
	for i := 0; i < len(e.versions); i++ {
		if e.versions[i] >= loaded {
			e.versions = append(e.versions[0:0], e.versions[:i]...)
			break
		}
	}
	// iterate over all payloads, add them to the tree
	ordered := make(map[types.SnapshotNamespace][]*types.Payload, len(nodeOrder))
	// positions and matching are linked to the market, work out how many payloads those will be:
	// total nodes - all nodes that aren't position or matching, divide by 2
	// not accounting for engines that have 2 or more nodes, this is a rough approximation of how many
	// nodes are matching/position engines, should not require reallocation
	mbPos := (len(e.snapshot.Nodes) - len(nodeOrder) + 2) / 2
	for i, pl := range e.snapshot.Nodes {
		ns := pl.Namespace()
		if _, ok := ordered[ns]; !ok {
			if ns == types.MatchingSnapshot || ns == types.PositionsSnapshot {
				ordered[ns] = make([]*types.Payload, 0, mbPos)
			} else {
				// some engines have 2, others 1
				ordered[ns] = []*types.Payload{}
			}
		}

		if err := e.setTreeNode(i, pl); err != nil {
			return err
		}
		// node was verified and set on tree
		ordered[ns] = append(ordered[ns], pl)
	}

	// start with app state
	e.wrap = ordered[types.AppSnapshot][0].GetAppState()
	e.app = e.wrap.AppState
	// set the context with the height + block + chainid
	ctx = vegactx.WithTraceID(vegactx.WithBlockHeight(ctx, int64(e.app.Height)), e.app.Block)
	ctx = vegactx.WithChainID(ctx, e.app.ChainID)
	// we're done restoring, now save the snapshot locally, so we can provide it moving forwards
	now := time.Unix(e.app.Time, 0)
	// restore app state
	e.timeService.SetTimeNow(ctx, now)
	e.statsService.SetHeight(e.app.Height)

	// before we starts restoring the providers
	for _, pp := range e.beforeRestoreProvs {
		if err := pp.OnStateLoadStarts(ctx); err != nil {
			return err
		}
	}
	// now let's load the data in the correct order, skip app state, we've already handled that
	for _, ns := range nodeOrder[1:] {
		for _, n := range ordered[ns] {
			p, ok := e.providers[n.GetTreeKey()]
			if !ok {
				return fmt.Errorf("%w %s", types.ErrUnknownSnapshotNamespace, n.GetTreeKey())
			}
			e.log.Debug("Loading provider", logging.String("tree-key", n.GetTreeKey()))
			nps, err := p.LoadState(ctx, n)
			if err != nil {
				return err
			}
			if len(nps) != 0 {
				e.AddProviders(nps...)
			}
		}
	}
	for _, pp := range e.restoreProvs {
		if err := pp.OnStateLoaded(ctx); err != nil {
			return err
		}
	}

	e.current = e.interval               // set the snapshot counter to the interval so that we do not create a duplicate snapshot on commit
	e.lastSnapshotHash = e.snapshot.Hash // set the engine's "last snapshot hash" field to the hash of the snapshot we've just loaded
	e.snapshot = nil                     // we're done, we can clear the snapshot state

	return nil
}

func (e *Engine) setTreeNode(i int, p *types.Payload) error {
	// unpack payload
	data, err := proto.Marshal(p.IntoProto())
	if err != nil {
		return err
	}

	// hash the node and save it into our cache of node hashes for comparison at the next snapshot
	hash := crypto.Hash(data)
	key := p.GetTreeKey()
	e.keyHashes[key] = hash

	return nil
}

func (e *Engine) ApplySnapshotChunk(chunk *types.RawChunk) (bool, error) {
	if e.snapshot == nil {
		return false, types.ErrUnknownSnapshot
	}
	if err := e.snapshot.LoadChunk(chunk); err != nil {
		return false, err
	}
	return e.snapshot.Ready(), nil
}

func (e *Engine) LoadSnapshotChunk(height uint64, format, chunk uint32) (*types.RawChunk, error) {
	if e.snapshot == nil {
		if err := e.setSnapshotForHeight(height); err != nil {
			return nil, err
		}
		if e.snapshot == nil {
			// @TODO try and retrieve the chunk
			return nil, types.ErrUnknownSnapshotChunkHeight
		}
	}
	// check format:
	f, err := types.SnapshotFromatFromU32(format)
	if err != nil {
		return nil, err
	}
	if f != e.snapshot.Format {
		return nil, types.ErrSnapshotFormatMismatch
	}
	return e.snapshot.GetRawChunk(chunk)
}

func (e *Engine) setSnapshotForHeight(height uint64) error {
	v, ok := e.versionHeight[height]
	if !ok {
		return types.ErrMissingSnapshotVersion
	}
	tree, err := e.avl.GetImmutable(v)
	if err != nil {
		return err
	}
	snap, err := types.SnapshotFromTree(tree)
	if err != nil {
		return err
	}
	e.snapshot = snap
	return nil
}

func (e *Engine) GetMissingChunks() []uint32 {
	if e.snapshot == nil {
		return nil
	}
	return e.snapshot.GetMissing()
}

// Info simply returns the current snapshot hash
// Can be used for the TM info call.
func (e *Engine) Info() ([]byte, int64) {
	return e.lastSnapshotHash, int64(e.app.Height)
}

func (e *Engine) Snapshot(ctx context.Context) (b []byte, errlol error) {
	e.current--
	// no snapshot to be taken yet
	if e.current > 0 {
		return nil, nil
	}
	// recent counter
	e.current = e.interval
	defer metrics.StartSnapshot("all")()
	// always iterate over slices, so loops are deterministic
	updated := false

	for _, ns := range e.namespaces {
		u, err := e.update(ns)
		if err != nil {
			e.log.Error("Failed to update snapshot namespace",
				logging.String("snapshot-namespace", ns.String()),
				logging.Error(err),
			)
			return nil, err
		}
		if u {
			updated = true
		}
	}

	// update appstate separately
	appUpdate := false
	height, err := vegactx.BlockHeightFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if height != int64(e.app.Height) {
		appUpdate = true
		e.app.Height = uint64(height)
	}
	_, block := vegactx.TraceIDFromContext(ctx)
	if block != e.app.Block {
		appUpdate = true
		e.app.Block = block
	}
	vNow := e.timeService.GetTimeNow().Unix()
	if e.app.Time != vNow {
		e.app.Time = vNow
		appUpdate = true
	}

	cid, err := vegactx.ChainIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if e.app.ChainID != cid {
		e.app.ChainID = cid
		appUpdate = true
	}

	if appUpdate {
		if err = e.updateAppState(); err != nil {
			return nil, err
		}
		updated = true
	}
	if !updated {
		return e.lastSnapshotHash, nil
	}
	return e.saveCurrentTree()
}

func (e *Engine) saveCurrentTree() ([]byte, error) {
	h, v, err := e.avl.SaveVersion()
	if err != nil {
		return nil, err
	}
	e.lastSnapshotHash = h
	if len(e.versions) >= cap(e.versions) {
		if err := e.avl.DeleteVersion(e.versions[0]); err != nil {
			// this is not a fatal error, but still we should be paying attention.
			e.log.Warn("Could not delete old version",
				logging.Int64("old-version", e.versions[0]),
				logging.Error(err),
			)
		}
		// drop first version
		copy(e.versions[0:], e.versions[1:])
		// set the last value in the slice to the current version
		e.versions[len(e.versions)-1] = v
	} else {
		// we're still building a backlog of versions
		e.versions = append(e.versions, v)
	}
	// get ptr to current version
	e.last = e.avl.ImmutableTree
	return h, nil
}

func (e *Engine) update(ns types.SnapshotNamespace) (bool, error) {
	defer metrics.StartSnapshot(string(ns))()
	treeKeys, ok := e.nsTreeKeys[ns]
	if !ok || len(treeKeys) == 0 {
		return false, nil
	}
	update := false
	toRemove := []int{}
	for i, treeKey := range treeKeys {
		treeKeyStr := string(treeKey)
		p := e.providers[treeKeyStr] // get the specific provider for this key
		lastHash := e.keyHashes[treeKeyStr]
		providerKey := e.treeKeyProvider[treeKeyStr]
		currentHash, err := p.GetHash(providerKey)
		if err != nil {
			return update, err
		}

		// nothing has changed (or both values were nil)
		if bytes.Equal(lastHash, currentHash) {
			continue
		}

		if len(currentHash) == 0 && p.Stopped() {
			// this signals the removal of this key
			toRemove = append(toRemove, i)
			continue
		}

		// hashes were different, we need to update
		v, generatedProviders, err := p.GetState(providerKey)
		if err != nil {
			return update, err
		}
		if len(generatedProviders) > 0 {
			// The provider has generated new providers, register them with the engine
			// add them to the AVL tree
			e.AddProviders(generatedProviders...)
			for _, n := range generatedProviders {
				e.log.Debug("Provider generated",
					logging.String("namespace", n.Namespace().String()),
				)
				if _, err = e.update(n.Namespace()); err != nil {
					return false, err
				}
			}
		}
		e.log.Debug("State updated",
			logging.String("node-key", treeKeyStr),
			logging.String("state-hash", hex.EncodeToString(currentHash)),
		)
		e.keyHashes[treeKeyStr] = currentHash
		if len(v) == 0 && len(currentHash) == 0 {
			// empty state -> remove data from snapshot
			if e.avl.Has(treeKey) {
				_, _ = e.avl.Remove(treeKey)
				update = true
				continue
			}
			// no value to set, but there was no node in the tree -> no update here
			continue
		}
		// new value needs to be set
		_ = e.avl.Set(treeKey, v)
		update = true
	}

	if len(toRemove) == 0 {
		return update, nil
	}

	for i := range toRemove {
		treeKey := treeKeys[i]
		treeKeyStr := string(treeKey)

		// delete everything we've got stored
		e.log.Debug("State to be removed", logging.String("node-key", treeKeyStr))
		delete(e.providers, treeKeyStr)
		delete(e.keyHashes, treeKeyStr)
		delete(e.treeKeyProvider, treeKeyStr)

		if !e.avl.Has(treeKey) {
			e.log.Panic("trying to remove non-existent payload from tree", logging.String("key", treeKeyStr))
			continue
		}

		if _, removed := e.avl.Remove(treeKey); !removed {
			e.log.Panic("failed to remove node from AVL tree", logging.String("key", treeKeyStr))
		}
	}

	for i := len(toRemove) - 1; i >= 0; i-- {
		e.nsTreeKeys[ns] = append(e.nsTreeKeys[ns][:i], e.nsTreeKeys[ns][i+1:]...)
	}

	return true, nil
}

func (e *Engine) updateAppState() error {
	keys, ok := e.nsTreeKeys[e.wrap.Namespace()]
	if !ok {
		return types.ErrNoPrefixFound
	}
	// there should be only 1 entry here
	if len(keys) > 1 || len(keys) == 0 {
		return types.ErrUnexpectedKey
	}
	// we only have 1 key
	pl := types.Payload{
		Data: e.wrap,
	}
	data, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return err
	}
	_ = e.avl.Set(keys[0], data)
	return nil
}

func (e *Engine) Hash(ctx context.Context) ([]byte, error) {
	if len(e.lastSnapshotHash) != 0 {
		return e.lastSnapshotHash, nil
	}
	return e.Snapshot(ctx)
}

func (e *Engine) AddProviders(provs ...types.StateProvider) {
	for _, p := range provs {
		keys := p.Keys()
		ns := p.Namespace()
		haveKeys, ok := e.nsKeys[ns]
		if !ok {
			e.providersNS[ns] = []types.StateProvider{
				p,
			}
			e.nsTreeKeys[ns] = make([][]byte, 0, len(keys))
			e.namespaces = append(e.namespaces, ns)
			for _, k := range keys {
				fullKey := types.GetNodeKey(ns, k)
				e.treeKeyProvider[fullKey] = k
				e.providers[fullKey] = p
				e.nsTreeKeys[ns] = append(e.nsTreeKeys[ns], []byte(fullKey))
			}
			if pp, ok := p.(types.PostRestore); ok {
				e.restoreProvs = append(e.restoreProvs, pp)
			}
			if pp, ok := p.(types.PreRestore); ok {
				e.beforeRestoreProvs = append(e.beforeRestoreProvs, pp)
			}
			e.nsKeys[ns] = keys
			continue
		}
		// in this case, we are replacing the provider
		if ns == types.ReplayProtectionSnapshot {
			for _, k := range keys {
				fullKey := types.GetNodeKey(ns, k)
				// replace the old provider with the replacement
				e.providers[fullKey] = p
			}
			rpl := false
			// replace provider reference in the NS map, too
			for i, oldP := range e.providersNS[ns] {
				// both in same namespace, have same keys -> replace
				if strings.StringSliceEqual(keys, oldP.Keys()) {
					e.providersNS[ns][i] = p
					rpl = true
					break
				}
			}
			// we found an exact match, and replaced the provider
			if rpl {
				continue
			}
			// no exact match was found, so we'll have to de-duplicate
		}
		dedup := uniqueSubset(haveKeys, keys)
		// note that the replay protection provider can replace itself (Noop -> actual protector)
		if len(dedup) == 0 && ns != types.ReplayProtectionSnapshot {
			continue // no new keys were added
		}
		e.nsKeys[ns] = append(e.nsKeys[ns], dedup...)
		// new provider in the same namespace
		e.providersNS[ns] = append(e.providersNS[ns], p)
		for _, k := range dedup {
			fullKey := types.GetNodeKey(ns, k)
			e.treeKeyProvider[fullKey] = k
			e.providers[fullKey] = p
			e.nsTreeKeys[ns] = append(e.nsTreeKeys[ns], []byte(fullKey))
		}
		if pp, ok := p.(types.PostRestore); ok {
			e.restoreProvs = append(e.restoreProvs, pp)
		}
		if pp, ok := p.(types.PreRestore); ok {
			e.beforeRestoreProvs = append(e.beforeRestoreProvs, pp)
		}
	}
}

func (e *Engine) Close() error {
	// keeps linters happy for now
	if e.pollCfunc != nil {
		e.pollCfunc()
		<-e.pollCtx.Done()
		for _, p := range e.providerTS {
			p.Sync()
		}
	}
	return e.db.Close()
}

func (e *Engine) OnSnapshotIntervalUpdate(ctx context.Context, interval int64) error {
	e.interval = interval
	if interval < e.current {
		e.current = interval
	}
	return nil
}

func uniqueSubset(have, add []string) []string {
	ret := make([]string, 0, len(add))
	for _, a := range add {
		if !inSlice(have, a) {
			ret = append(ret, a)
		}
	}
	return ret
}

func inSlice(s []string, v string) bool {
	for _, sv := range s {
		if sv == v {
			return true
		}
	}
	return false
}
