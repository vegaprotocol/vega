package snapshot

import (
	"bytes"
	"context"
	"strings"

	"code.vegaprotocol.io/vega/logging"

	"github.com/cosmos/iavl"
	db "github.com/tendermint/tm-db"
)

type StateProvider interface {
	// Namespace this provider operates in, basically a prefix for the keys
	Namespace() string
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

type Engine struct {
	Config

	ctx        context.Context
	cfunc      context.CancelFunc
	log        *logging.Logger
	avl        *iavl.MutableTree
	namespaces []string
	keys       [][]byte
	nsKeys     map[string][]string
	hashes     map[string][]byte

	providers map[string]StateProvider

	last    *iavl.ImmutableTree
	hash    []byte
	version int64
}

func New(ctx context.Context, conf Config, log *logging.Logger) (*Engine, error) {
	log = log.Named(namedLogger)
	tree, err := iavl.NewMutableTree(db.NewMemDB(), 0)
	if err != nil {
		log.Error("Could not create AVL tree", logging.Error(err))
		return nil, err
	}
	sctx, cfunc := context.WithCancel(ctx)
	return &Engine{
		Config:     conf,
		ctx:        sctx,
		cfunc:      cfunc,
		log:        log,
		avl:        tree,
		namespaces: []string{},
		nsKeys:     map[string][]string{},
		hashes:     map[string][]byte{},
		providers:  map[string]StateProvider{},
	}, nil
}

func (e *Engine) Snapshot() ([]byte, error) {
	// always iterate over slices, so loops are deterministic
	updated := false
	for _, ns := range e.namespaces {
		keys := e.nsKeys[ns]
		for _, k := range keys {
			u, err := e.update(ns, k)
			if err != nil {
				return nil, err
			}
			if u {
				updated = true
			}
		}
	}
	if !updated {
		return e.hash, nil
	}
	h, v, err := e.avl.SaveVersion()
	if err != nil {
		return nil, err
	}
	e.hash = h
	e.version = v
	// get ptr to current version
	e.last = e.avl.ImmutableTree
	return h, nil
}

func (e *Engine) update(ns, k string) (bool, error) {
	p := e.providers[ns]
	nsKey := strings.Join([]string{ns, k}, ".")
	ch := e.hashes[nsKey]
	h, err := p.GetHash(k)
	if err != nil {
		return false, err
	}
	// current hash matches old one
	if bytes.Equal(ch, h) {
		return false, nil
	}
	// hash needs updating
	v, err := p.GetState(k)
	if err != nil {
		return false, err
	}
	e.hashes[nsKey] = h
	key := []byte(nsKey) // key is ns.Key as byte slice
	_ = e.avl.Set(key, v)
	return true, nil
}

func (e *Engine) Hash() ([]byte, error) {
	if len(e.hash) != 0 {
		return e.hash, nil
	}
	return e.Snapshot()
}

func (e *Engine) AddProviders(provs ...StateProvider) {
	for _, p := range provs {
		ns := p.Namespace()
		keys := p.Keys()
		haveKeys, ok := e.nsKeys[ns]
		if !ok {
			// just add
			e.nsKeys[ns] = keys
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
	}
	// just create the first snapshot
	_, _ = e.Snapshot()
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
