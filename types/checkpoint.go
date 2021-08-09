package types

import (
	"bytes"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/crypto"
)

type CheckpointName string

const (
	GovernanceCheckpoint CheckpointName = "governance"
	AssetsCheckpoint     CheckpointName = "assets"
	CollateralCheckpoint CheckpointName = "collateral"
	NetParamsCheckpoint  CheckpointName = "netparams"
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
}

func NewSnapshotFromProto(ps *vega.Snapshot) *Snapshot {
	return &Snapshot{
		State: ps.State,
		Hash:  ps.Hash,
	}
}

func (s Snapshot) IntoProto() *vega.Snapshot {
	return &vega.Snapshot{
		Hash:  s.Hash,
		State: s.State,
	}
}

func (s Snapshot) GetCheckpoint() (*Checkpoint, error) {
	pc := &vega.Checkpoint{}
	if err := vega.Unmarshal(s.State, pc); err != nil {
		return nil, err
	}
	cp := NewCheckpointFromProto(pc)
	return cp, nil
}

func (s *Snapshot) SetState(state []byte) error {
	cp := &vega.Checkpoint{}
	if err := vega.Unmarshal(state, cp); err != nil {
		return err
	}
	c := NewCheckpointFromProto(cp)
	s.State = state
	s.Hash = crypto.Hash(c.HashBytes())
	return nil
}

func (s *Snapshot) SetCheckpoint(cp *Checkpoint) error {
	b, err := vega.Marshal(cp.IntoProto())
	if err != nil {
		return err
	}
	s.Hash = crypto.Hash(cp.HashBytes())
	s.State = b
	return nil
}

// IsValid checks the hash, returns false if the hash doesn't match
func (s Snapshot) IsValid() bool {
	cp, err := s.GetCheckpoint()
	if err != nil {
		return false
	}
	return bytes.Equal(crypto.Hash(cp.HashBytes()), s.Hash)
}

func NewCheckpointFromProto(pc *vega.Checkpoint) *Checkpoint {
	return &Checkpoint{
		Governance:        pc.Governance,
		Assets:            pc.Assets,
		Collateral:        pc.Collateral,
		NetworkParameters: pc.NetworkParameters,
	}
}

func (c Checkpoint) IntoProto() *vega.Checkpoint {
	return &vega.Checkpoint{
		Governance:        c.Governance,
		Assets:            c.Assets,
		Collateral:        c.Collateral,
		NetworkParameters: c.NetworkParameters,
	}
}

// HashBytes returns the data contained in the snapshot as a []byte for hashing
// the order in which the data is added to the slice matters
func (c Checkpoint) HashBytes() []byte {
	ret := make([]byte, 0, len(c.Governance)+len(c.Assets)+len(c.Collateral)+len(c.NetworkParameters))
	// the order in which we append is quite important
	ret = append(ret, c.NetworkParameters...)
	ret = append(ret, c.Assets...)
	ret = append(ret, c.Collateral...)
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
	}
	return nil
}
