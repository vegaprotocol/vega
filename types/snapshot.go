package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"

	"github.com/cosmos/iavl"

	tmtypes "github.com/tendermint/tendermint/abci/types"
)

type SnapshotNamespace string

const (
	undefinedSnapshot  SnapshotNamespace = ""
	AppSnapshot        SnapshotNamespace = "app"
	CollateralSnapshot SnapshotNamespace = "collateral"
)

var (
	nsMap = map[string]SnapshotNamespace{
		"collateral": CollateralSnapshot,
		"app":        AppSnapshot,
	}

	ErrUnknownSnapshotNamespace  = errors.New("unknown snapshot namespace")
	ErrNoPrefixFound             = errors.New("no prefix in chunk keys")
	ErrInconsistentNamespaceKeys = errors.New("chunk contains several namespace keys")
	ErrChunkHashMismatch         = errors.New("loaded chunk hash does not match metadata")
)

type SnapshotFormat uint32

const (
	SnapshotFormatUnspecified SnapshotFormat = iota
	SnapshotFormatProto
	SnapshotFormatJSON
	SnapshotFormatPJSON
)

// SnapshotMeta the hashes marshalled... could be hashes on a per engine/namespace basis?
type SnapshotMeta struct {
	Version    int64  `json:"version"`
	Collateral []byte `json:"collateral"`
	App        []byte `json:"app"`
}

type AppState struct {
	Height uint64 `json:"height"`
	Block  string `json:"block"`
}

// TMSnapshot is the snapshot type as listed by ListSnapshots etc...
type TMSnapshot struct {
	Height   uint64
	Format   SnapshotFormat
	Chunks   uint32
	Hash     []byte
	Metadata []byte
	chunks   []*Chunk
	meta     *SnapshotMeta
}

type Chunk struct {
	Namespace SnapshotNamespace
	Hash      []byte
	ID        uint32 // chunk number
	data      map[string][]byte
	Bytes     []byte
}

func NewTMSnapshotFromTM(tms *tmtypes.Snapshot) (*TMSnapshot, error) {
	snap := TMSnapshot{
		Height:   tms.Height,
		Format:   SnapshotFormat(tms.Format),
		Chunks:   tms.Chunks,
		chunks:   make([]*Chunk, int(tms.Chunks)),
		Hash:     tms.Hash,
		Metadata: tms.Metadata,
		meta:     &SnapshotMeta{},
	}
	if err := json.Unmarshal(tms.Metadata, snap.meta); err != nil {
		return nil, err
	}
	return &snap, nil
}

func NewTMSnapshotFromIAVL(tree *iavl.ImmutableTree, nsKey map[string][][]byte) (*TMSnapshot, error) {
	snap := &TMSnapshot{
		Hash:   tree.Hash(),
		chunks: make([]*Chunk, 0, len(nsKey)),
		meta: &SnapshotMeta{
			Version: tree.Version(),
		},
	}
	chunks := make(map[string]*Chunk, len(nsKey))
	names := make([]string, 0, len(nsKey))
	for n := range nsKey {
		names = append(names, n)
	}
	// sort the namespaces so the chunks will always match up
	sort.Strings(names)
	for _, n := range names {
		keys := nsKey[n]
		chunk, ok := chunks[n]
		if !ok {
			ns, err := namespaceFromString(n)
			if err != nil {
				return nil, err
			}
			snap.Chunks++
			chunk = &Chunk{
				Namespace: ns,
				ID:        snap.Chunks,
				data:      make(map[string][]byte, len(keys)),
			}
			snap.chunks = append(snap.chunks, chunk)
		}
		// the map as-is, but ready to serialise
		serialise := make(map[string]json.RawMessage, len(keys))
		for _, key := range keys {
			_, val := tree.Get(key)
			// now get the key without the prefix:
			k := string(key[len([]byte(n))+1:])
			// we can just use this as-is, we're only serialising here
			// and then not re-use the value
			serialise[k] = json.RawMessage(val)
			// this is the application data, we need to get the height
			if n == string(AppSnapshot) {
				app := &AppState{}
				if err := json.Unmarshal(val, app); err != nil {
					return nil, err
				}
				snap.Height = app.Height
			}
			// we need to copy this, because this may be used by the IAVL
			// so just in case we write to that for whatever reason
			cpy := make([]byte, 0, len(val))
			copy(cpy, val)
			chunk.data[k] = cpy
		}
		// chunk should be done now
		b, err := json.Marshal(serialise)
		if err != nil {
			return nil, err
		}
		chunk.Bytes = b
	}
	// ensure Metadata is set, and chunk hashes have been calculated
	if err := snap.SetMeta(); err != nil {
		return nil, err
	}
	return snap, nil
}

func (s TMSnapshot) ChunksLeft() int {
	i := 0
	for _, c := range s.chunks {
		if c == nil {
			i++
		}
	}
	return i
}

func (s *TMSnapshot) LoadChunk(idx uint32, bytes []byte) error {
	chunk := &Chunk{
		Namespace: undefinedSnapshot,
		ID:        idx,
		Bytes:     bytes,
		Hash:      crypto.Hash(bytes),
	}
	// ensure the hash of the data matches what we expect
	if err := s.checkChunkHash(chunk); err != nil {
		return err
	}
	data := map[string]json.RawMessage{}
	if err := json.Unmarshal(bytes, data); err != nil {
		return err
	}
	chunk.data = make(map[string][]byte, len(data))
	for k, v := range data {
		// get the namespace
		i := 0
		for i < len(k)-1 && k[i] != '.' {
			i++
		}
		if k[i] != '.' {
			return ErrNoPrefixFound
		}
		ns, err := namespaceFromString(string(k[:i]))
		if err != nil {
			return err
		}
		if chunk.Namespace == undefinedSnapshot {
			chunk.Namespace = ns
		} else if ns != chunk.Namespace {
			return ErrInconsistentNamespaceKeys
		}
		chunk.data[k] = []byte(v)
	}
	s.chunks[int(idx)] = chunk
	return nil
}

func (s *TMSnapshot) SetMeta() error {
	for _, c := range s.chunks {
		switch c.Namespace {
		case CollateralSnapshot:
			s.meta.Collateral = crypto.Hash(c.Bytes)
			c.Hash = s.meta.Collateral
		case AppSnapshot:
			s.meta.App = crypto.Hash(c.Bytes)
			c.Hash = s.meta.App
		}
	}
	m, err := json.Marshal(s.meta)
	if err != nil {
		return err
	}
	s.Metadata = m
	return nil
}

func (s TMSnapshot) CheckMeta() error {
	for _, c := range s.chunks {
		if err := s.checkChunkHash(c); err != nil {
			return err
		}
	}
	return nil
}

func (s TMSnapshot) checkChunkHash(c *Chunk) error {
	switch c.Namespace {
	case CollateralSnapshot:
		if !bytes.Equal(c.Hash, s.meta.Collateral) {
			return ErrChunkHashMismatch
		}
	case AppSnapshot:
		if !bytes.Equal(c.Hash, s.meta.App) {
			return ErrChunkHashMismatch
		}
	}
	return nil
}

func (s TMSnapshot) ToTM() *tmtypes.Snapshot {
	return &tmtypes.Snapshot{
		Height:   s.Height,
		Format:   uint32(s.Format),
		Chunks:   s.Chunks,
		Hash:     s.Hash,
		Metadata: s.Metadata,
	}
}

func namespaceFromString(s string) (SnapshotNamespace, error) {
	ns, ok := nsMap[s]
	if !ok {
		return undefinedSnapshot, ErrUnknownSnapshotNamespace
	}
	return ns, nil
}
