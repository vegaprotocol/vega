package checkpoint

import (
	"errors"
	"sort"
)

var (
	ErrUnknownCheckpointName      = errors.New("component for checkpoint not registered")
	ErrComponentWithDuplicateName = errors.New("multiple components with the same name")
)

// State interface represents system components that need snapshotting
// Name returns the component name (key in engine map)
// Hash returns, obviously, the state hash
// @TODO adding func to get the actual data
//go:generate go run github.com/golang/mock/mockgen -destination mocks/state_mock.go -package mocks code.vegaprotocol.io/vega/checkpoint State
type State interface {
	Name() string
	Hash() []byte
	Checkpoint() []byte
	Load(checkpoint, hash []byte) error
}

type Engine struct {
	components map[string]State
	ordered    []string
}

func New(components ...State) (*Engine, error) {
	e := &Engine{
		components: make(map[string]State, len(components)),
		ordered:    make([]string, 0, len(components)),
	}
	for _, c := range components {
		if err := e.addComponent(c); err != nil {
			return nil, err
		}
	}
	sort.Strings(e.ordered)
	return e, nil
}

// Add used to add/register components after the engine has been instantiated already
// this is mainly used to make testing easier
func (e *Engine) Add(comps ...State) error {
	e.ordered = append(make([]string, 0, len(e.ordered)+len(comps)), e.ordered...)
	for _, c := range comps {
		if err := e.addComponent(c); err != nil {
			return err
		}
	}
	sort.Strings(e.ordered)
	return nil
}

// add component, but check for duplicate names
func (e *Engine) addComponent(comp State) error {
	name := comp.Name()
	c, ok := e.components[name]
	if !ok {
		e.components[name] = comp
		e.ordered = append(e.ordered, name)
		return nil
	}
	if c != comp {
		return ErrComponentWithDuplicateName
	}
	// component was registered already
	return nil
}

func (e *Engine) GetCheckpoints() map[string]Snapshot {
	ret := make(map[string]Snapshot, len(e.components))
	for _, k := range e.ordered {
		// ensure we access components in the same order all the time
		c := e.components[k]
		ret[k] = Snapshot{
			name: k,
			hash: c.Hash(),
			data: c.Checkpoint(),
		}
	}
	return ret
}

// Load - loads checkpoint data for all components by name
func (e *Engine) Load(checkpoints map[string]Snapshot) error {
	// first ensure that all keys exist
	for k := range checkpoints {
		if _, ok := e.components[k]; !ok {
			return ErrUnknownCheckpointName
		}
	}
	for _, k := range e.ordered {
		snapshot, ok := checkpoints[k]
		if !ok {
			continue // no checkpoint data
		}
		comp := e.components[k] // we know this exists
		if err := comp.Load(snapshot.data, snapshot.hash); err != nil {
			return err
		}
	}
	return nil
}
