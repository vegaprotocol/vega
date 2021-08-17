package types

import "code.vegaprotocol.io/protos/vega"

type CheckpointName string

const (
	GovernanceCheckpoint CheckpointName = "governance"
	AssetsCheckpoint     CheckpointName = "assets"
	CollateralCheckpoint CheckpointName = "collateral"
	NetParamsCheckpoint  CheckpointName = "netparams"
)

type Checkpoint struct {
	Governance        []byte
	Assets            []byte
	Collateral        []byte
	NetworkParameters []byte
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
