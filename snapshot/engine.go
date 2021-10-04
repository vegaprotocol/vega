package snapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	vegactx "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"github.com/cosmos/iavl"
	db "github.com/tendermint/tm-db"
)

type StateProvider interface {
	// Namespace this provider operates in, basically a prefix for the keys
	Namespace() types.SnapshotNamespace
	// Keys gets all the nodes this provider populates
	Keys() []string
	// GetHash returns the hash for the state for a given key
	// this can be used to check for changes
	GetHash(key string) ([]byte, error)
	// Snapshot is a sync call to get the state for all keys
	Snapshot() (map[string][]byte, error)
	// GetState is a sync call to fetch the current state for a current key
	// the same as Snapshot, basically, but for specific keys
	// e.g. foo.Snapshot(bar) returns the current state of foo for key bar
	GetState(key string) ([]byte, error)
	// Watch sets up the channels that contain new state, it's the providers' job to write to them
	// each time the state is updated
	// Watch(ctx context.Context, key string) (<-chan []byte, <-chan error)

	// Sync is a blocking call that returns once there are no changes to the provider state
	// that haven't been sent on the channels
	// Sync() error
}

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
	namespaces []string
	keys       []string
	nsKeys     map[string][]string
	nsTreeKeys map[string][][]byte
	hashes     map[string][]byte
	versions   []int64

	providers map[string]StateProvider

	last    *iavl.ImmutableTree
	hash    []byte
	version int64

	snapshot  *types.Snapshot
	snapRetry int
}

// New returns a new snapshot engine.
func New(ctx context.Context, conf Config, log *logging.Logger, tm TimeService) (*Engine, error) {
	log = log.Named(namedLogger)
	dbConn := db.NewMemDB()
	tree, err := iavl.NewMutableTree(dbConn, 0)
	if err != nil {
		log.Error("Could not create AVL tree", logging.Error(err))
		return nil, err
	}
	sctx, cfunc := context.WithCancel(ctx)
	app := string(types.AppSnapshot)
	return &Engine{
		Config: conf,
		ctx:    sctx,
		cfunc:  cfunc,
		time:   tm,
		db:     dbConn,
		log:    log,
		avl:    tree,
		namespaces: []string{
			app,
		},
		keys: []string{},
		nsKeys: map[string][]string{
			app: {"all"},
		},
		nsTreeKeys: map[string][][]byte{
			app: {
				[]byte(strings.Join([]string{app, "all"}, ".")),
			},
		},
		hashes:    map[string][]byte{},
		providers: map[string]StateProvider{},
		versions:  make([]int64, 0, conf.Versions), // cap determines how many versions we keep
	}, nil
}

// List returns all snapshots available.
func (e *Engine) List() ([]*types.Snapshot, error) {
	trees := make([]*types.Snapshot, 0, len(e.versions))
	for _, v := range e.versions {
		tree, err := e.avl.GetImmutable(v)
		if err != nil {
			return nil, err
		}
		snap, err := types.SnapshotFromIAVL(tree, e.keys)
		if err != nil {
			return nil, err
		}
		trees = append(trees, snap)
	}
	return trees, nil
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

func (e *Engine) ApplySnapshot() error {
	if e.snapshot == nil {
		return types.ErrUnknownSnapshot
	}
	// @TODO we have all the data we need
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
		// @TODO try and retrieve the chunk
		return nil, types.ErrUnknownSnapshotChunkHeight
	}
	// check format:
	f, err := types.SnapshotFromatFromU32(format)
	if err != nil {
		return nil, err
	}
	if f != e.snapshot.Format {
		return nil, types.ErrSnapshotFormatMismatch
	}
	return e.snapshot.GetRawChunk(height)
}

func (e *Engine) GetMissingChunks() []uint32 {
	if e.snapshot == nil {
		return nil
	}
	return e.snapshot.GetMissing()
}

func (e *Engine) ReceiveChunk() error {
	return nil
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
	if !updated {
		return e.hash, nil
	}
	// set height and all that jazz
	if err := e.addAppSnap(ctx); err != nil {
		return nil, err
	}
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

func (e *Engine) addAppSnap(ctx context.Context) error {
	height, err := vegactx.BlockHeightFromContext(ctx)
	if err != nil {
		return err
	}
	_, block := vegactx.TraceIDFromContext(ctx)
	app := types.AppState{
		Height: uint64(height),
		Block:  block,
		Time:   e.time.GetTimeNow().Unix(),
	}
	as, err := json.Marshal(app)
	if err != nil {
		return err
	}
	// we know the key:
	_ = e.avl.Set(e.nsTreeKeys[string(types.AppSnapshot)][0], as)
	return nil
}

func (e *Engine) update(ns string) (bool, error) {
	p := e.providers[ns]
	update := false
	for _, nsKey := range e.nsTreeKeys[ns] {
		sKey := string(nsKey)
		ch := e.hashes[sKey]
		pKey := string(nsKey[len([]byte(ns))+1:]) // truncate namespace + . gets key
		h, err := p.GetHash(pKey)
		if err != nil {
			return update, err
		}
		if bytes.Equal(ch, h) {
			// no update, we're done with this key
			continue
		}
		// hashes don't match
		v, err := p.GetState(pKey)
		if err != nil {
			return update, err
		}
		// we have new state, and new hash
		e.hashes[sKey] = h
		_ = e.avl.Set(nsKey, v)
		update = true
	}
	return update, nil
}

func (e *Engine) Hash(ctx context.Context) ([]byte, error) {
	if len(e.hash) != 0 {
		return e.hash, nil
	}
	return e.Snapshot(ctx)
}

func (e *Engine) AddProviders(provs ...StateProvider) {
	for _, p := range provs {
		ns := p.Namespace().String()
		keys := p.Keys()
		haveKeys, ok := e.nsKeys[ns]
		if !ok {
			// just add
			e.nsKeys[ns] = keys
			nsTreeKeys := make([][]byte, 0, len(keys))
			for _, k := range keys {
				key := strings.Join([]string{ns, k}, ".")
				e.keys = append(e.keys, key)
				nsTreeKeys = append(nsTreeKeys, []byte(key))
			}
			e.nsTreeKeys[ns] = nsTreeKeys
			e.namespaces = append(e.namespaces, ns)
			continue
		}
		dedup := uniqueSubset(haveKeys, keys)
		if len(dedup) == 0 {
			continue
		}
		if len(dedup) != len(keys) {
			e.log.Debug("Skipping keys we already have")
		}
		e.nsKeys[ns] = append(haveKeys, dedup...)
		nsTreeKeys := e.nsTreeKeys[ns]
		for _, k := range dedup {
			key := strings.Join([]string{ns, k}, ".")
			e.keys = append(e.keys, key)
			nsTreeKeys = append(nsTreeKeys, []byte(key))
		}
		e.nsTreeKeys[ns] = nsTreeKeys
	}
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
