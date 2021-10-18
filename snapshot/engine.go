package snapshot

import (
	"bytes"
	"context"
	"encoding/hex"
	"time"

	"code.vegaprotocol.io/shared/paths"
	vegactx "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"github.com/cosmos/iavl"
	"github.com/golang/protobuf/proto"
	db "github.com/tendermint/tm-db"
)

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

// Engine the snapshot engine.
type Engine struct {
	Config

	ctx        context.Context
	cfunc      context.CancelFunc
	time       TimeService
	db         db.DB
	log        *logging.Logger
	avl        *iavl.MutableTree
	namespaces []types.SnapshotNamespace
	nsKeys     map[types.SnapshotNamespace][]string
	nsTreeKeys map[types.SnapshotNamespace][][]byte
	keyNoNS    map[string]string // full key => key used by provider
	hashes     map[string][]byte
	versions   []int64

	providers   map[string]types.StateProvider
	providersNS map[types.SnapshotNamespace][]types.StateProvider
	providerTs  map[string]StateProviderT
	pollCtx     context.Context
	pollCfunc   context.CancelFunc

	last          *iavl.ImmutableTree
	hash          []byte
	version       int64
	versionHeight map[uint64]int64

	snapshot  *types.Snapshot
	snapRetry int

	// the general snapshot info this engine is responsible for
	wrap *types.PayloadAppState
	app  *types.AppState
}

// order in which snapshots are to be restored.
var nodeOrder = []types.SnapshotNamespace{
	types.AppSnapshot,
	types.AssetsSnapshot,
	types.BankingSnapshot,
	types.CollateralSnapshot,
	types.NotarySnapshot,
	types.NetParamsSnapshot,
	types.CheckpointSnapshot,
	types.DelegationSnapshot,
	types.ExecutionSnapshot, // creates the markets, returns matching and positions engines for state providers
	types.MatchingSnapshot,  // this requires a market
	types.PositionsSnapshot, // again, needs a market
	types.EpochSnapshot,
	types.StakingSnapshot,
	types.SpamSnapshot,
	types.LimitSnapshot,
	types.ReplayProtectionSnapshot,
	types.RewardSnapshot,
}

// New returns a new snapshot engine.
func New(ctx context.Context, vegapath paths.Paths, conf Config, log *logging.Logger, tm TimeService) (*Engine, error) {
	log = log.Named(namedLogger)
	dbConn, err := getDB(conf, vegapath)
	if err != nil {
		log.Error("Failed to open DB connection", logging.Error(err))
		return nil, err
	}
	tree, err := iavl.NewMutableTree(dbConn, 0)
	if err != nil {
		log.Error("Could not create AVL tree", logging.Error(err))
		return nil, err
	}
	sctx, cfunc := context.WithCancel(ctx)
	appPL := &types.PayloadAppState{
		AppState: &types.AppState{},
	}
	app := appPL.Namespace()
	return &Engine{
		Config:     conf,
		ctx:        sctx,
		cfunc:      cfunc,
		time:       tm,
		db:         dbConn,
		log:        log,
		avl:        tree,
		namespaces: []types.SnapshotNamespace{},
		nsKeys: map[types.SnapshotNamespace][]string{
			app: {appPL.Key()},
		},
		nsTreeKeys: map[types.SnapshotNamespace][][]byte{
			app: {
				[]byte(types.KeyFromPayload(appPL)),
			},
		},
		keyNoNS:       map[string]string{},
		hashes:        map[string][]byte{},
		providers:     map[string]types.StateProvider{},
		providersNS:   map[types.SnapshotNamespace][]types.StateProvider{},
		versions:      make([]int64, 0, conf.Versions), // cap determines how many versions we keep
		versionHeight: map[uint64]int64{},
		wrap:          appPL,
		app:           appPL.AppState,
	}, nil
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

func getDB(conf Config, vegapath paths.Paths) (db.DB, error) {
	if conf.Storage == memDB {
		return db.NewMemDB(), nil
	}
	dbPath, err := vegapath.DataPathFor(paths.SnapshotStateHome)
	if err != nil {
		return nil, err
	}
	return db.NewGoLevelDB("snapshot", dbPath)
}

// List returns all snapshots available.
func (e *Engine) List() ([]*types.Snapshot, error) {
	trees := make([]*types.Snapshot, 0, len(e.versions))
	for _, v := range e.versions {
		tree, err := e.avl.GetImmutable(v)
		if err != nil {
			return nil, err
		}
		snap, err := types.SnapshotFromTree(tree)
		if err != nil {
			return nil, err
		}
		trees = append(trees, snap)
		e.versionHeight[snap.Height] = snap.Meta.Version
	}
	return trees, nil
}

func (e *Engine) LoadFromStore(ctx context.Context) error {
	version, err := e.avl.Load()
	if err != nil {
		return err
	}
	e.version = version
	e.last = e.avl.ImmutableTree
	snap, err := types.SnapshotFromTree(e.last)
	if err != nil {
		return err
	}
	e.snapshot = snap
	// apply, no need to set the tree, it's coming from local store
	return e.applySnap(ctx, false)
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

func (e *Engine) ApplySnapshot(ctx context.Context) error {
	return e.applySnap(ctx, true)
}

func (e *Engine) applySnap(ctx context.Context, cas bool) error {
	if e.snapshot == nil {
		return types.ErrUnknownSnapshot
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
		if cas {
			if err := e.nodeCAS(i, pl); err != nil {
				return err
			}
		}
		// node was verified and set on tree
		ordered[ns] = append(ordered[ns], pl)
	}

	// start with app state
	e.wrap = ordered[types.AppSnapshot][0].GetAppState()
	e.app = e.wrap.AppState
	// set the context with the height + block
	ctx = vegactx.WithTraceID(vegactx.WithBlockHeight(ctx, int64(e.app.Height)), e.app.Block)

	// now let's load the data in the correct order, skip app state, we've already handled that
	for _, ns := range nodeOrder[1:] {
		for _, n := range ordered[ns] {
			p, ok := e.providers[n.GetTreeKey()]
			if !ok {
				return types.ErrUnknownSnapshotNamespace
			}
			nps, err := p.LoadState(ctx, n)
			if err != nil {
				return err
			}
			if len(nps) != 0 {
				e.AddProviders(nps...)
			}
		}
	}
	// we're done restoring, now save the snapshot locally, so we can provide it moving forwards
	now := time.Unix(e.app.Time, 0)
	// restore app state
	e.time.SetTimeNow(ctx, now)
	// we're done, we can clear the snapshot state
	e.snapshot = nil
	// no need to save, return here
	if !cas {
		return nil
	}
	if _, err := e.saveCurrentTree(); err != nil {
		return err
	}
	return nil
}

func (e *Engine) nodeCAS(i int, p *types.Payload) error {
	h, err := e.setTreeNode(p)
	if err != nil {
		return err
	}
	if exp := e.snapshot.Meta.NodeHashes[i].Hash; exp != hex.EncodeToString(h) {
		key := p.GetTreeKey()
		e.log.Error("Snapshot node not restored - hash mismatch",
			logging.String("node-key", key),
		)
		_, _ = e.avl.Remove([]byte(key))
		return types.ErrNodeHashMismatch
	}
	return nil
}

func (e *Engine) setTreeNode(p *types.Payload) ([]byte, error) {
	data, err := proto.Marshal(p.IntoProto())
	if err != nil {
		return nil, err
	}
	hash := crypto.Hash(data)
	key := p.GetTreeKey()
	e.hashes[key] = hash
	_ = e.avl.Set([]byte(key), data)
	return hash, nil
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
	return e.snapshot.GetRawChunk(uint32(height))
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

func (e *Engine) Snapshot(ctx context.Context) ([]byte, error) {
	// always iterate over slices, so loops are deterministic
	updated := false
	for _, ns := range e.namespaces {
		u, err := e.update(ns)
		if err != nil {
			return nil, err
		}
		if u {
			updated = true
		}
	}
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
	vNow := e.time.GetTimeNow().Unix()
	if e.app.Time != vNow {
		e.app.Time = vNow
		appUpdate = true
	}
	if appUpdate {
		if updated, err = e.updateAppState(); err != nil {
			return nil, err
		}
	}
	if !updated {
		return e.hash, nil
	}
	return e.saveCurrentTree()
}

func (e *Engine) saveCurrentTree() ([]byte, error) {
	h, v, err := e.avl.SaveVersion()
	if err != nil {
		return nil, err
	}
	e.hash = h
	e.version = v
	if len(e.versions) >= cap(e.versions) {
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
	treeKeys, ok := e.nsTreeKeys[ns]
	if !ok || len(treeKeys) == 0 {
		return false, nil
	}
	update := false
	for _, tk := range treeKeys {
		k := string(tk)
		p := e.providers[k] // get the specific provider for this key
		ch := e.hashes[k]
		kNNS := e.keyNoNS[k]
		h, err := p.GetHash(kNNS)
		if err != nil {
			return update, err
		}
		// nothing has changed
		if bytes.Equal(ch, h) {
			continue
		}
		// hashes were different, we need to update
		v, err := p.GetState(kNNS)
		if err != nil {
			return update, err
		}
		e.log.Debug("State updated",
			logging.String("node-key", k),
			logging.String("state-hash", hex.EncodeToString(h)),
		)
		e.hashes[k] = h
		_ = e.avl.Set(tk, v)
		update = true
	}
	return update, nil
}

func (e *Engine) updateAppState() (bool, error) {
	keys, ok := e.nsTreeKeys[e.wrap.Namespace()]
	if !ok {
		return false, types.ErrNoPrefixFound
	}
	// there should be only 1 entry here
	if len(keys) > 1 || len(keys) == 0 {
		return false, types.ErrUnexpectedKey
	}
	// we only have 1 key
	pl := types.Payload{
		Data: e.wrap,
	}
	data, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return false, err
	}
	_ = e.avl.Set(keys[0], data)
	return true, nil
}

func (e *Engine) Hash(ctx context.Context) ([]byte, error) {
	if len(e.hash) != 0 {
		return e.hash, nil
	}
	return e.Snapshot(ctx)
}

func (e *Engine) AddProviders(provs ...types.StateProvider) {
	for _, p := range provs {
		ks := p.Keys()
		ns := p.Namespace()
		haveKeys, ok := e.nsKeys[ns]
		if !ok {
			e.providersNS[ns] = []types.StateProvider{
				p,
			}
			e.nsTreeKeys[ns] = make([][]byte, 0, len(ks))
			e.namespaces = append(e.namespaces, ns)
			for _, k := range ks {
				fullKey := types.GetNodeKey(ns, k)
				e.keyNoNS[fullKey] = k
				e.providers[fullKey] = p
				e.nsTreeKeys[ns] = append(e.nsTreeKeys[ns], []byte(fullKey))
			}
			e.nsKeys[ns] = ks
			continue
		}
		dedup := uniqueSubset(haveKeys, ks)
		// @TODO log this - or panic?
		if len(dedup) == 0 {
			continue // no new keys were added
		}
		e.nsKeys[ns] = append(e.nsKeys[ns], dedup...)
		// new provider in the same namespace
		e.providersNS[ns] = append(e.providersNS[ns], p)
		for _, k := range dedup {
			fullKey := types.GetNodeKey(ns, k)
			e.keyNoNS[fullKey] = k
			e.providers[fullKey] = p
			e.nsTreeKeys[ns] = append(e.nsTreeKeys[ns], []byte(fullKey))
		}
	}
}

func (e *Engine) RegisteredNamespaces() []types.SnapshotNamespace {
	return e.namespaces
}

func (e *Engine) Close() error {
	return e.db.Close()
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
