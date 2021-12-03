package types

import (
	"bytes"
	"errors"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/protobuf/proto"
)

var (
	ErrCheckpointStateInvalid  = errors.New("state contained in the snapshot is invalid")
	ErrCheckpointHashIncorrect = errors.New("the hash and snapshot data do not match")
	ErrCheckpointHasNoState    = errors.New("there is no state set on the checkpoint")
)

type CheckpointName string

const (
	GovernanceCheckpoint     CheckpointName = "governance"
	AssetsCheckpoint         CheckpointName = "assets"
	CollateralCheckpoint     CheckpointName = "collateral"
	NetParamsCheckpoint      CheckpointName = "netparams"
	DelegationCheckpoint     CheckpointName = "delegation"
	EpochCheckpoint          CheckpointName = "epoch"
	BlockCheckpoint          CheckpointName = "block" // pseudo-checkpoint, really...
	PendingRewardsCheckpoint CheckpointName = "rewards"
)

type Block struct {
	Height int64
}

type CheckpointState struct {
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
	Block             []byte
	Rewards           []byte
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
	Auto    []string
}

func NewCheckpointStateFromProto(ps *checkpoint.CheckpointState) *CheckpointState {
	return &CheckpointState{
		State: ps.State,
		Hash:  ps.Hash,
	}
}

func (s CheckpointState) IntoProto() *checkpoint.CheckpointState {
	return &checkpoint.CheckpointState{
		Hash:  s.Hash,
		State: s.State,
	}
}

func (s CheckpointState) GetCheckpoint() (*Checkpoint, error) {
	pc := &checkpoint.Checkpoint{}
	if err := proto.Unmarshal(s.State, pc); err != nil {
		return nil, err
	}
	cp := NewCheckpointFromProto(pc)
	return cp, nil
}

func (s *CheckpointState) SetState(state []byte) error {
	cp := &checkpoint.Checkpoint{}
	if err := proto.Unmarshal(state, cp); err != nil {
		return err
	}
	c := NewCheckpointFromProto(cp)
	s.State = state
	s.Hash = crypto.Hash(c.HashBytes())
	return nil
}

func (s CheckpointState) GetBlockHeight() (int64, error) {
	if len(s.State) == 0 {
		return 0, ErrCheckpointHasNoState
	}
	cp := &checkpoint.Checkpoint{}
	if err := proto.Unmarshal(s.State, cp); err != nil {
		return 0, err
	}
	c := NewCheckpointFromProto(cp)
	return c.GetBlockHeight()
}

func (s *CheckpointState) SetCheckpoint(cp *Checkpoint) error {
	b, err := proto.Marshal(cp.IntoProto())
	if err != nil {
		return err
	}
	s.Hash = crypto.Hash(cp.HashBytes())
	s.State = b
	return nil
}

// Validate checks the hash, returns nil if valid.
func (s CheckpointState) Validate() error {
	cp, err := s.GetCheckpoint()
	if err != nil {
		return ErrCheckpointStateInvalid
	}
	if !bytes.Equal(crypto.Hash(cp.HashBytes()), s.Hash) {
		return ErrCheckpointHashIncorrect
	}
	return nil
}

func NewCheckpointFromProto(pc *checkpoint.Checkpoint) *Checkpoint {
	return &Checkpoint{
		Governance:        pc.Governance,
		Assets:            pc.Assets,
		Collateral:        pc.Collateral,
		NetworkParameters: pc.NetworkParameters,
		Delegation:        pc.Delegation,
		Epoch:             pc.Epoch,
		Block:             pc.Block,
		Rewards:           pc.Rewards,
	}
}

func (c Checkpoint) IntoProto() *checkpoint.Checkpoint {
	return &checkpoint.Checkpoint{
		Governance:        c.Governance,
		Assets:            c.Assets,
		Collateral:        c.Collateral,
		NetworkParameters: c.NetworkParameters,
		Delegation:        c.Delegation,
		Epoch:             c.Epoch,
		Block:             c.Block,
		Rewards:           c.Rewards,
	}
}

func (c *Checkpoint) SetBlockHeight(height int64) error {
	b := Block{
		Height: height,
	}
	bb, err := proto.Marshal(b.IntoProto())
	if err != nil {
		return err
	}
	c.Block = bb
	return nil
}

// HashBytes returns the data contained in the checkpoint as a []byte for hashing
// the order in which the data is added to the slice matters.
func (c Checkpoint) HashBytes() []byte {
	ret := make([]byte, 0, len(c.Governance)+len(c.Assets)+len(c.Collateral)+len(c.NetworkParameters)+len(c.Delegation)+len(c.Epoch)+len(c.Block))
	// the order in which we append is quite important
	ret = append(ret, c.NetworkParameters...)
	ret = append(ret, c.Assets...)
	ret = append(ret, c.Collateral...)
	ret = append(ret, c.Delegation...)
	ret = append(ret, c.Epoch...)
	ret = append(ret, c.Block...)
	ret = append(ret, c.Governance...)
	return append(ret, c.Rewards...)
}

// Set set a specific checkpoint value using the name the engine returns.
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
	case BlockCheckpoint:
		c.Block = val
	case PendingRewardsCheckpoint:
		c.Rewards = val
	}
}

// Get as the name suggests gets the data by checkpoint name.
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
	case BlockCheckpoint:
		return c.Block
	case PendingRewardsCheckpoint:
		return c.Rewards
	}
	return nil
}

func (c Checkpoint) GetBlockHeight() (int64, error) {
	pb := &checkpoint.Block{}
	if err := proto.Unmarshal(c.Block, pb); err != nil {
		return 0, err
	}
	return pb.Height, nil
}

func NewDelegationEntryFromProto(de *checkpoint.DelegateEntry) *DelegationEntry {
	amt, _ := num.UintFromString(de.Amount, 10)
	return &DelegationEntry{
		Party:      de.Party,
		Node:       de.Node,
		Amount:     amt,
		Undelegate: de.Undelegate,
		EpochSeq:   de.EpochSeq,
	}
}

func (d DelegationEntry) IntoProto() *checkpoint.DelegateEntry {
	return &checkpoint.DelegateEntry{
		Party:      d.Party,
		Node:       d.Node,
		Amount:     d.Amount.String(),
		Undelegate: d.Undelegate,
		EpochSeq:   d.EpochSeq,
	}
}

func NewDelegationCPFromProto(sd *checkpoint.Delegate) *DelegateCP {
	r := &DelegateCP{
		Active:  make([]*DelegationEntry, 0, len(sd.Active)),
		Pending: make([]*DelegationEntry, 0, len(sd.Pending)),
		Auto:    sd.AutoDelegation[:],
	}
	for _, a := range sd.Active {
		r.Active = append(r.Active, NewDelegationEntryFromProto(a))
	}
	for _, p := range sd.Pending {
		r.Pending = append(r.Pending, NewDelegationEntryFromProto(p))
	}
	return r
}

func (d DelegateCP) IntoProto() *checkpoint.Delegate {
	s := &checkpoint.Delegate{
		Active:         make([]*checkpoint.DelegateEntry, 0, len(d.Active)),
		Pending:        make([]*checkpoint.DelegateEntry, 0, len(d.Pending)),
		AutoDelegation: d.Auto[:],
	}
	for _, a := range d.Active {
		s.Active = append(s.Active, a.IntoProto())
	}
	for _, p := range d.Pending {
		s.Pending = append(s.Pending, p.IntoProto())
	}
	return s
}

func NewBlockFromProto(bp *checkpoint.Block) *Block {
	return &Block{
		Height: bp.Height,
	}
}

func (b Block) IntoProto() *checkpoint.Block {
	return &checkpoint.Block{
		Height: b.Height,
	}
}
