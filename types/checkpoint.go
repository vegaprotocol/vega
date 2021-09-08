package types

import (
	"bytes"
	"errors"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/protobuf/proto"
)

var (
	ErrSnapshotStateInvalid  = errors.New("state contained in the snapshot is invalid")
	ErrSnapshotHashIncorrect = errors.New("the hash and snapshot data do not match")
)

type CheckpointName string

const (
	GovernanceCheckpoint CheckpointName = "governance"
	AssetsCheckpoint     CheckpointName = "assets"
	CollateralCheckpoint CheckpointName = "collateral"
	NetParamsCheckpoint  CheckpointName = "netparams"
	DelegationCheckpoint CheckpointName = "delegation"
	EpochCheckpoint      CheckpointName = "epoch"
)

type Snapshot struct {
	State []byte
	Hash  []byte
}

type Checkpoint struct {
	Governance        []byte
	Assets            []byte
	Collateral        []byte
	NetworkParameters []byte
	Delegation        []byte
	Epoch             []byte
}

type DelegationEntry struct {
	Party      string
	Node       string
	Amount     *num.Uint
	Undelegate bool
	EpochSeq   uint64
}

type DelegateCP struct {
	Active  []*DelegationEntry
	Pending []*DelegationEntry
}

func NewSnapshotFromProto(ps *snapshot.Snapshot) *Snapshot {
	return &Snapshot{
		State: ps.State,
		Hash:  ps.Hash,
	}
}

func (s Snapshot) IntoProto() *snapshot.Snapshot {
	return &snapshot.Snapshot{
		Hash:  s.Hash,
		State: s.State,
	}
}

func (s Snapshot) GetCheckpoint() (*Checkpoint, error) {
	pc := &snapshot.Checkpoint{}
	if err := proto.Unmarshal(s.State, pc); err != nil {
		return nil, err
	}
	cp := NewCheckpointFromProto(pc)
	return cp, nil
}

func (s *Snapshot) SetState(state []byte) error {
	cp := &snapshot.Checkpoint{}
	if err := proto.Unmarshal(state, cp); err != nil {
		return err
	}
	c := NewCheckpointFromProto(cp)
	s.State = state
	s.Hash = crypto.Hash(c.HashBytes())
	return nil
}

func (s *Snapshot) SetCheckpoint(cp *Checkpoint) error {
	b, err := proto.Marshal(cp.IntoProto())
	if err != nil {
		return err
	}
	s.Hash = crypto.Hash(cp.HashBytes())
	s.State = b
	return nil
}

// Validate checks the hash, returns nil if valid
func (s Snapshot) Validate() error {
	cp, err := s.GetCheckpoint()
	if err != nil {
		return ErrSnapshotStateInvalid
	}
	if !bytes.Equal(crypto.Hash(cp.HashBytes()), s.Hash) {
		return ErrSnapshotHashIncorrect
	}
	return nil
}

func NewCheckpointFromProto(pc *snapshot.Checkpoint) *Checkpoint {
	return &Checkpoint{
		Governance:        pc.Governance,
		Assets:            pc.Assets,
		Collateral:        pc.Collateral,
		NetworkParameters: pc.NetworkParameters,
		Delegation:        pc.Delegation,
		Epoch:             pc.Epoch,
	}
}

func (c Checkpoint) IntoProto() *snapshot.Checkpoint {
	return &snapshot.Checkpoint{
		Governance:        c.Governance,
		Assets:            c.Assets,
		Collateral:        c.Collateral,
		NetworkParameters: c.NetworkParameters,
		Delegation:        c.Delegation,
		Epoch:             c.Epoch,
	}
}

// HashBytes returns the data contained in the snapshot as a []byte for hashing
// the order in which the data is added to the slice matters
func (c Checkpoint) HashBytes() []byte {
	ret := make([]byte, 0, len(c.Governance)+len(c.Assets)+len(c.Collateral)+len(c.NetworkParameters)+len(c.Delegation)+len(c.Epoch))
	// the order in which we append is quite important
	ret = append(ret, c.NetworkParameters...)
	ret = append(ret, c.Assets...)
	ret = append(ret, c.Collateral...)
	ret = append(ret, c.Delegation...)
	ret = append(ret, c.Epoch...)
	return append(ret, c.Governance...)
}

// Set set a specific checkpoint value using the name the engine returns
func (c *Checkpoint) Set(name CheckpointName, val []byte) {
	switch name {
	case GovernanceCheckpoint:
		c.Governance = val
	case AssetsCheckpoint:
		c.Assets = val
	case CollateralCheckpoint:
		c.Collateral = val
	case NetParamsCheckpoint:
		c.NetworkParameters = val
	case DelegationCheckpoint:
		c.Delegation = val
	case EpochCheckpoint:
		c.Epoch = val
	}
}

// Get as the name suggests gets the data by checkpoint name
func (c Checkpoint) Get(name CheckpointName) []byte {
	switch name {
	case GovernanceCheckpoint:
		return c.Governance
	case AssetsCheckpoint:
		return c.Assets
	case CollateralCheckpoint:
		return c.Collateral
	case NetParamsCheckpoint:
		return c.NetworkParameters
	case DelegationCheckpoint:
		return c.Delegation
	case EpochCheckpoint:
		return c.Epoch
	}
	return nil
}

func NewDelegationEntryFromProto(de *snapshot.DelegateEntry) *DelegationEntry {
	amt, _ := num.UintFromString(de.Amount, 10)
	return &DelegationEntry{
		Party:      de.Party,
		Node:       de.Node,
		Amount:     amt,
		Undelegate: de.Undelegate,
		EpochSeq:   de.EpochSeq,
	}
}

func (d DelegationEntry) IntoProto() *snapshot.DelegateEntry {
	return &snapshot.DelegateEntry{
		Party:      d.Party,
		Node:       d.Node,
		Amount:     d.Amount.String(),
		Undelegate: d.Undelegate,
		EpochSeq:   d.EpochSeq,
	}
}

func NewDelegationCPFromProto(sd *snapshot.Delegate) *DelegateCP {
	r := &DelegateCP{
		Active:  make([]*DelegationEntry, 0, len(sd.Active)),
		Pending: make([]*DelegationEntry, 0, len(sd.Pending)),
	}
	for _, a := range sd.Active {
		r.Active = append(r.Active, NewDelegationEntryFromProto(a))
	}
	for _, p := range sd.Pending {
		r.Pending = append(r.Pending, NewDelegationEntryFromProto(p))
	}
	return r
}

func (d DelegateCP) IntoProto() *snapshot.Delegate {
	s := &snapshot.Delegate{
		Active:  make([]*snapshot.DelegateEntry, 0, len(d.Active)),
		Pending: make([]*snapshot.DelegateEntry, 0, len(d.Pending)),
	}
	for _, a := range d.Active {
		s.Active = append(s.Active, a.IntoProto())
	}
	for _, p := range d.Pending {
		s.Pending = append(s.Pending, p.IntoProto())
	}
	return s
}
