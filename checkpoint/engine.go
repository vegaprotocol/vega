package checkpoint

import (
	"bytes"
	"errors"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/types"
)

var (
	ErrUnknownCheckpointName      = errors.New("component for checkpoint not registered")
	ErrComponentWithDuplicateName = errors.New("multiple components with the same name")
	ErrSnapshotHashIncorrect      = errors.New("the hash and snapshot data do not match")

	cpOrder = []types.CheckpointName{
		types.NetParamsCheckpoint,  // net params should go first
		types.AssetsCheckpoint,     // assets are required for collateral to work
		types.CollateralCheckpoint, // without balances, governance (proposals, bonds) are difficult
		types.GovernanceCheckpoint, // depends on all of the above
	}
)

// State interface represents system components that need snapshotting
// Name returns the component name (key in engine map)
// Hash returns, obviously, the state hash
// @TODO adding func to get the actual data
//go:generate go run github.com/golang/mock/mockgen -destination mocks/state_mock.go -package mocks code.vegaprotocol.io/vega/checkpoint State
type State interface {
	Name() types.CheckpointName
	Checkpoint() ([]byte, error)
	Load(checkpoint []byte) error
}

type Engine struct {
	components map[types.CheckpointName]State
	ordered    []string
}

func New(components ...State) (*Engine, error) {
	e := &Engine{
		components: make(map[types.CheckpointName]State, len(components)),
	}
	for _, c := range components {
		if err := e.addComponent(c); err != nil {
			return nil, err
		}
	}
	return e, nil
}

// Add used to add/register components after the engine has been instantiated already
// this is mainly used to make testing easier
func (e *Engine) Add(comps ...State) error {
	for _, c := range comps {
		if err := e.addComponent(c); err != nil {
			return err
		}
	}
	return nil
}

// add component, but check for duplicate names
func (e *Engine) addComponent(comp State) error {
	name := comp.Name()
	c, ok := e.components[name]
	if !ok {
		e.components[name] = comp
		return nil
	}
	if c != comp {
		return ErrComponentWithDuplicateName
	}
	// component was registered already
	return nil
}

// Checkpoint returns the overall checkpoint
func (e *Engine) Checkpoint() ([]byte, error) {
	cp := &types.Checkpoint{}
	for _, k := range cpOrder {
		comp, ok := e.components[k]
		if !ok {
			continue
		}
		data, err := comp.Checkpoint()
		if err != nil {
			return nil, err
		}
		// set the correct field
		cp.Set(k, data)
	}
	b, err := vega.Marshal(cp.IntoProto())
	if err != nil {
		return nil, err
	}

	return vega.Marshal(&vega.Snapshot{
		State: b,
		Hash:  crypto.Hash(cp.HashBytes()),
	})
}

// Load - loads checkpoint data for all components by name
func (e *Engine) Load(data []byte) error {
	snap := &vega.Snapshot{}
	if err := vega.Unmarshal(data, snap); err != nil {
		return err
	}
	vcp := &vega.Checkpoint{}
	if err := vega.Unmarshal(snap.State, vcp); err != nil {
		return err
	}
	cp := types.NewCheckpointFromProto(vcp)
	// check the hash
	if hash := crypto.Hash(cp.HashBytes()); bytes.Compare(hash, snap.Hash) != 0 {
		return ErrSnapshotHashIncorrect
	}
	for _, k := range cpOrder {
		cpData := cp.Get(k)
		if len(cpData) == 0 {
			continue
		}
		c, ok := e.components[k]
		if !ok {
			return ErrUnknownCheckpointName // data cannot be restored
		}
		if err := c.Load(cpData); err != nil {
			return err
		}
	}
	return nil
}
