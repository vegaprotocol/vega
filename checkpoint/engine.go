package checkpoint

import (
	"errors"

	"code.vegaprotocol.io/vega/types"
)

var (
	ErrUnknownCheckpointName      = errors.New("component for checkpoint not registered")
	ErrComponentWithDuplicateName = errors.New("multiple components with the same name")

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

func (e *Engine) Checkpoint() (*types.Checkpoint, error) {
	return nil, nil
}

func (e *Engine) GetCheckpoints() (map[string]Snapshot, error) {
	ret := make(map[string]Snapshot, len(e.components))
	for _, k := range cpOrder {
		// ensure we access components in the same order all the time
		c, ok := e.components[k]
		if !ok {
			continue
		}
		data, err := c.Checkpoint()
		if err != nil {
			return nil, err
		}
		sk := string(k)
		ret[sk] = Snapshot{
			name: sk,
			data: data,
		}
	}
	return ret, nil
}

// Load - loads checkpoint data for all components by name
func (e *Engine) Load(checkpoints map[string]Snapshot) error {
	// first ensure that all keys exist
	for k := range checkpoints {
		name := types.CheckpointName(k)
		if _, ok := e.components[name]; !ok {
			return ErrUnknownCheckpointName
		}
	}
	for _, k := range cpOrder {
		snapshot, ok := checkpoints[string(k)]
		if !ok {
			continue // no checkpoint data
		}
		comp, ok := e.components[k]
		if !ok {
			continue
		}
		if err := comp.Load(snapshot.data); err != nil {
			return err
		}
	}
	return nil
}
