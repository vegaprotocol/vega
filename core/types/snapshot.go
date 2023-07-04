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

package types

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/crypto"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/cosmos/iavl"
	tmtypes "github.com/tendermint/tendermint/abci/types"
)

// StateProvider - not a huge fan of this interface being here, but it ensures that the state providers
// don't have to import the snapshot package
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/state_provider_mock.go -package mocks code.vegaprotocol.io/vega/core/types StateProvider
type StateProvider interface {
	Namespace() SnapshotNamespace
	Keys() []string
	// NB: GetState must be threadsafe as it may be called from multiple goroutines concurrently!
	GetState(key string) ([]byte, []StateProvider, error)
	LoadState(ctx context.Context, pl *Payload) ([]StateProvider, error)
	Stopped() bool
}

// PostRestore is basically a StateProvider which, after the full core state is restored, expects a callback to finalise the state restore
// Note that the order in which the calls to this OnStateLoaded functions are called is not pre-defined. As such, this method should only be used
// for engine internals (upkeep, essentially)
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/restore_state_provider_mock.go -package mocks code.vegaprotocol.io/vega/core/types PostRestore
type PostRestore interface {
	StateProvider
	OnStateLoaded(ctx context.Context) error
}

type PreRestore interface {
	StateProvider
	OnStateLoadStarts(ctx context.Context) error
}

type SnapshotNamespace string

const (
	undefinedSnapshot              SnapshotNamespace = ""
	AppSnapshot                    SnapshotNamespace = "app"
	AssetsSnapshot                 SnapshotNamespace = "assets"
	WitnessSnapshot                SnapshotNamespace = "witness" // Must be done before any engine that call RestoreResource
	BankingSnapshot                SnapshotNamespace = "banking"
	CheckpointSnapshot             SnapshotNamespace = "checkpoint"
	CollateralSnapshot             SnapshotNamespace = "collateral"
	NetParamsSnapshot              SnapshotNamespace = "netparams"
	DelegationSnapshot             SnapshotNamespace = "delegation"
	GovernanceSnapshot             SnapshotNamespace = "governance"
	PositionsSnapshot              SnapshotNamespace = "positions"
	MatchingSnapshot               SnapshotNamespace = "matching"
	ExecutionSnapshot              SnapshotNamespace = "execution"
	EpochSnapshot                  SnapshotNamespace = "epoch"
	StakingSnapshot                SnapshotNamespace = "staking"
	IDGenSnapshot                  SnapshotNamespace = "idgenerator"
	RewardSnapshot                 SnapshotNamespace = "rewards"
	SpamSnapshot                   SnapshotNamespace = "spam"
	LimitSnapshot                  SnapshotNamespace = "limits"
	NotarySnapshot                 SnapshotNamespace = "notary"
	StakeVerifierSnapshot          SnapshotNamespace = "stakeverifier"
	EventForwarderSnapshot         SnapshotNamespace = "eventforwarder"
	TopologySnapshot               SnapshotNamespace = "topology"
	LiquiditySnapshot              SnapshotNamespace = "liquidity"
	LiquidityTargetSnapshot        SnapshotNamespace = "liquiditytarget"
	FloatingPointConsensusSnapshot SnapshotNamespace = "floatingpoint"
	MarketActivityTrackerSnapshot  SnapshotNamespace = "marketActivityTracker"
	ERC20MultiSigTopologySnapshot  SnapshotNamespace = "erc20multisigtopology"
	PoWSnapshot                    SnapshotNamespace = "pow"
	ProtocolUpgradeSnapshot        SnapshotNamespace = "protocolUpgradeProposals"
	SettlementSnapshot             SnapshotNamespace = "settlement"

	MaxChunkSize   = 16 * 1000 * 1000 // technically 16 * 1024 * 1024, but you know
	IdealChunkSize = 10 * 1000 * 1000 // aim for 10MB
)

var (
	ErrSnapshotHashMismatch         = errors.New("snapshot hashes do not match")
	ErrSnapshotMetaMismatch         = errors.New("snapshot metadata does not match")
	ErrUnknownSnapshotNamespace     = errors.New("unknown snapshot namespace")
	ErrNoPrefixFound                = errors.New("no prefix in chunk keys")
	ErrInconsistentNamespaceKeys    = errors.New("chunk contains several namespace keys")
	ErrChunkHashMismatch            = errors.New("loaded chunk hash does not match metadata")
	ErrChunkOutOfRange              = errors.New("chunk number out of range")
	ErrUnknownSnapshot              = errors.New("no shapshot to reject")
	ErrMissingChunks                = errors.New("missing previous chunks")
	ErrSnapshotRetryLimit           = errors.New("could not load snapshot, retry limit reached")
	ErrSnapshotKeyDoesNotExist      = errors.New("unknown key for snapshot")
	ErrInvalidSnapshotNamespace     = errors.New("invalid snapshot namespace")
	ErrUnknownSnapshotType          = errors.New("snapshot data type not known")
	ErrUnknownSnapshotChunkHeight   = errors.New("no snapshot or chunk found for given height")
	ErrInvalidSnapshotFormat        = errors.New("invalid snapshot format")
	ErrSnapshotFormatMismatch       = errors.New("snapshot formats do not match")
	ErrUnexpectedKey                = errors.New("snapshot namespace has unknown/unexpected key(s)")
	ErrNodeHashMismatch             = errors.New("hash of a node does not match the hash from the snapshot meta")
	ErrNoSnapshot                   = errors.New("no snapshot found")
	ErrMissingSnapshotVersion       = errors.New("unknown snapshot version")
	ErrInvalidSnapshotStorageMethod = errors.New("invalid snapshot storage method")
	ErrMissingAppstateNode          = errors.New("appstate missing from tree")
	ErrMissingPayload               = errors.New("payload missing from exported tree")
)

type SnapshotFormat = snapshot.Format

const (
	SnapshotFormatUnspecified     = snapshot.Format_FORMAT_UNSPECIFIED
	SnapshotFormatProto           = snapshot.Format_FORMAT_PROTO
	SnapshotFormatProtoCompressed = snapshot.Format_FORMAT_PROTO_COMPRESSED
	SnapshotFormatJSON            = snapshot.Format_FORMAT_JSON
)

type RawChunk struct {
	Nr     uint32
	Data   []byte
	Height uint64
	Format SnapshotFormat
}

func SnapshotFromTM(tms *tmtypes.Snapshot) (*Snapshot, error) {
	snap := Snapshot{
		Height:     tms.Height,
		Format:     SnapshotFormat(tms.Format),
		Chunks:     tms.Chunks,
		Hash:       tms.Hash,
		ByteChunks: make([][]byte, int(tms.Chunks)), // have the chunk slice ready for loading
	}
	meta := &snapshot.Metadata{}
	if err := proto.Unmarshal(tms.Metadata, meta); err != nil {
		return nil, err
	}
	md, err := MetadataFromProto(meta)
	if err != nil {
		return nil, err
	}
	snap.Meta = md
	return &snap, nil
}

func (s Snapshot) ToTM() (*tmtypes.Snapshot, error) {
	md, err := proto.Marshal(s.Meta.IntoProto())
	if err != nil {
		return nil, err
	}
	return &tmtypes.Snapshot{
		Height:   s.Height,
		Format:   uint32(s.Format),
		Chunks:   s.Chunks,
		Hash:     s.Hash,
		Metadata: md,
	}, nil
}

func AppStateFromTree(tree *iavl.ImmutableTree) (*PayloadAppState, error) {
	appState := &Payload{
		Data: &PayloadAppState{AppState: &AppState{}},
	}
	key := appState.GetTreeKey()
	data, _ := tree.Get([]byte(key))
	if data == nil {
		return nil, ErrSnapshotKeyDoesNotExist
	}
	prp := appState.IntoProto()
	if err := proto.Unmarshal(data, prp); err != nil {
		return nil, err
	}
	appState = PayloadFromProto(prp)
	return appState.GetAppState(), nil
}

// TreeFromSnapshot takes the given snapshot data and creates a avl tree from it.
func (s *Snapshot) TreeFromSnapshot(tree *iavl.MutableTree) error {
	importer, err := tree.Import(s.Meta.Version)
	if err != nil {
		return fmt.Errorf("failed instantiate AVL tree importer: %w", err)
	}
	defer importer.Close()

	// Convert slice into map for quick lookup
	payloads := map[string]*Payload{}
	for _, pl := range s.Nodes {
		payloads[pl.GetTreeKey()] = pl
	}

	// Add the nodes in in the order they were exported in SnapshotFromTree
	for _, n := range s.Meta.NodeHashes {
		var value []byte
		if n.IsLeaf {
			// value is the snapshot payload
			payload, ok := payloads[n.Key]
			if !ok {
				return ErrMissingPayload
			}
			value, err = proto.Marshal(payload.IntoProto())
			if err != nil {
				return err
			}
		} else {
			// it is very important that this is a nil-slice and not an empty-non-nil slice for importer.Add()
			// to work which is why I've made it explicit and left this comment. An empty slice means there is
			// a node value but its empty, a nil slice means there is no value.
			value = nil
		}

		// Reconstruct exported node and add it to the important
		importer.Add(
			&iavl.ExportNode{
				Key:     []byte(n.Key),
				Value:   value,
				Height:  int8(n.Height), // this is the height of the node in thre tree
				Version: n.Version,      // this is the version of the node in the tree (it is incremented if that node's value is updated)
			})
	}

	// validate the import and commit it into the tree
	err = importer.Commit()
	if err != nil {
		return fmt.Errorf("could not commit imported tree: %w", err)
	}

	return nil
}

// SnapshotFromTree traverses the given avl tree and represents it as a Snapshot.
func SnapshotFromTree(tree *iavl.ImmutableTree) (*Snapshot, error) {
	hash, err := tree.Hash()
	if err != nil {
		return nil, err
	}
	snap := Snapshot{
		Hash: hash,
		Meta: &Metadata{
			Version:     tree.Version(),
			NodeHashes:  []*NodeHash{}, // a slice of the data for each node in the tree without the payload value, just its hash
			ChunkHashes: []string{},
		},

		// a slice of payloads that correspond to the nodehashes. Note that len(NodeHashes) != len(Nodes) since
		// only the leaf nodes of the tree contain payload data, the sub-tree roots only exist for the merkle-hash.
		Nodes: []*Payload{},
	}

	exporter, err := tree.Export()
	if err != nil {
		return nil, fmt.Errorf("could not export the current AVL tree: %w", err)
	}
	defer exporter.Close()

	exportedNode, err := exporter.Next()
	for err == nil {
		hash := hex.EncodeToString(crypto.Hash(exportedNode.Value))
		node := &NodeHash{
			Hash:    hash,
			Height:  int32(exportedNode.Height),
			Version: exportedNode.Version,
			Key:     string(exportedNode.Key),
			IsLeaf:  exportedNode.Value != nil,
		}

		snap.Meta.NodeHashes = append(snap.Meta.NodeHashes, node)

		// its only the nodes at the end of the tree which have the payload data
		// all intermediary nodes have empty values and just make up the merkle-tree
		// if we are a payload-less node just step again
		if !node.IsLeaf {
			exportedNode, err = exporter.Next()
			continue
		}

		// sort out the payload for this node
		pl := &snapshot.Payload{}
		if perr := proto.Unmarshal(exportedNode.Value, pl); err != nil {
			return nil, perr
		}

		payload := PayloadFromProto(pl)
		payload.raw = exportedNode.Value[:]

		snap.Nodes = append(snap.Nodes, payload)

		// if it happens to be the appstate payload grab the snapshot height while we're there
		if payload.Namespace() == AppSnapshot {
			p, _ := payload.Data.(*PayloadAppState)
			snap.Height = p.AppState.Height
		}

		// move onto the next node
		exportedNode, err = exporter.Next()
	}

	if !errors.Is(err, iavl.ErrorExportDone) {
		// either an error occurred while traversing, or we never reached the end
		return nil, fmt.Errorf("failed to export AVL tree: %w", err)
	}

	if snap.Height == 0 {
		return nil, fmt.Errorf("failed to export AVL tree: %w", ErrMissingAppstateNode)
	}

	// set chunks, ready to send in case we need it
	snap.nodesToChunks()
	return &snap, nil
}

func (s *Snapshot) ValidateMeta(other *Snapshot) error {
	if len(s.Meta.ChunkHashes) != len(other.Meta.ChunkHashes) || len(s.Meta.NodeHashes) != len(other.Meta.NodeHashes) {
		return ErrSnapshotMetaMismatch
	}
	for i := range s.Meta.ChunkHashes {
		if other.Meta.ChunkHashes[i] != s.Meta.ChunkHashes[i] {
			return ErrSnapshotMetaMismatch
		}
	}
	for i := range s.Meta.NodeHashes {
		if other.Meta.NodeHashes[i].Hash != s.Meta.NodeHashes[i].Hash {
			return ErrSnapshotMetaMismatch
		}
	}
	return nil
}

func (s *Snapshot) nodesToChunks() {
	all := &Chunk{
		Data: s.Nodes[:],
		Nr:   1,
		Of:   1,
	}

	b, _ := proto.Marshal(all.IntoProto())
	if len(b) < MaxChunkSize {
		s.DataChunks = []*Chunk{
			all,
		}
		s.hashChunks()
		return
	}
	parts := len(b) / IdealChunkSize
	if t := parts * IdealChunkSize; t != len(b) {
		parts++
	}
	s.ByteChunks = make([][]byte, 0, parts)
	step := len(b) / parts
	for i := 0; i < len(b); i += step {
		end := i + step
		if end > len(b) {
			end = len(b)
		}
		s.ByteChunks = append(s.ByteChunks, b[i:end])
	}

	s.hashByteChunks()
}

func (s *Snapshot) hashByteChunks() {
	s.Meta.ChunkHashes = make([]string, 0, len(s.ByteChunks))
	for _, b := range s.ByteChunks {
		s.Meta.ChunkHashes = append(s.Meta.ChunkHashes, hex.EncodeToString(crypto.Hash(b)))
		s.Chunks++
	}
}

func (s *Snapshot) hashChunks() {
	s.Meta.ChunkHashes = make([]string, 0, len(s.DataChunks))
	s.ByteChunks = make([][]byte, 0, len(s.DataChunks))
	for _, c := range s.DataChunks {
		pc := c.IntoProto()
		b, _ := proto.Marshal(pc)
		s.Meta.ChunkHashes = append(s.Meta.ChunkHashes, hex.EncodeToString(crypto.Hash(b)))
		s.ByteChunks = append(s.ByteChunks, b)
		s.Chunks++
	}
}

func (s *Snapshot) LoadChunk(chunk *RawChunk) error {
	if chunk.Nr > s.Chunks {
		return ErrChunkOutOfRange
	}
	if len(s.ByteChunks) == 0 {
		s.ByteChunks = make([][]byte, int(s.Chunks))
	}
	i := int(chunk.Nr)
	if len(s.Meta.ChunkHashes) <= i {
		return ErrChunkOutOfRange
	}
	hash := hex.EncodeToString(crypto.Hash(chunk.Data))
	if s.Meta.ChunkHashes[i] != hash {
		return ErrChunkHashMismatch
	}
	s.ByteChunks[i] = chunk.Data
	s.byteLen += len(chunk.Data)
	s.ChunksSeen++
	chunk.Height = s.Height
	chunk.Format = s.Format
	if s.Chunks == s.ChunksSeen {
		return s.unmarshalChunks()
	}
	// this ought to be the last one, but we're clearly missing some
	if j := i + 1; j == int(s.Chunks) {
		return ErrMissingChunks
	}
	return nil
}

func (s Snapshot) GetMissing() []uint32 {
	// no need to check seen vs expected, seen will always be smaller
	ids := make([]uint32, 0, int(s.Chunks-s.ChunksSeen))
	for i := range s.ByteChunks {
		if len(s.ByteChunks) == 0 {
			ids = append(ids, uint32(i))
		}
	}
	return ids
}

func (s *Snapshot) unmarshalChunks() error {
	data := make([]byte, 0, s.byteLen)
	for _, b := range s.ByteChunks {
		data = append(data, b...)
	}
	sChunk := &snapshot.Chunk{}
	if err := proto.Unmarshal(data, sChunk); err != nil {
		return err
	}
	s.DataChunks = []*Chunk{
		ChunkFromProto(sChunk),
	}
	s.Nodes = make([]*Payload, 0, len(sChunk.Data))
	for _, pl := range sChunk.Data {
		s.Nodes = append(s.Nodes, PayloadFromProto(pl))
	}
	return nil
}

func (s Snapshot) Ready() bool {
	return s.ChunksSeen == s.Chunks
}

func (n SnapshotNamespace) String() string {
	return string(n)
}

func SnapshotFormatFromU32(f uint32) (SnapshotFormat, error) {
	i32 := int32(f)
	if _, ok := snapshot.Format_name[i32]; !ok {
		return SnapshotFormatUnspecified, ErrInvalidSnapshotFormat
	}
	return SnapshotFormat(i32), nil
}
