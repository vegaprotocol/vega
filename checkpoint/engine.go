package checkpoint

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

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

// AssetsState is a bit of a hacky way to get the assets that were enabled when checkpoint was reloaded, so we can enable them in the collateral engine
//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_state_mock.go -package mocks code.vegaprotocol.io/vega/checkpoint AssetsState
type AssetsState interface {
	State
	GetEnabledAssets() []*types.Asset
}

// CollateralState is part 2 of the hacky way to enable the assets required to load the collateral state
//go:generate go run github.com/golang/mock/mockgen -destination mocks/collateral_state_mock.go -package mocks code.vegaprotocol.io/vega/checkpoint CollateralState
type CollateralState interface {
	State
	EnableAsset(ctx context.Context, asset types.Asset) error
}

type Engine struct {
	components map[types.CheckpointName]State
	loadHash   []byte
	nextCP     time.Time
	delta      time.Duration
}

func New(components ...State) (*Engine, error) {
	e := &Engine{
		components: make(map[types.CheckpointName]State, len(components)),
		nextCP:     time.Time{},
	}
	for _, c := range components {
		if err := e.addComponent(c); err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e *Engine) UponGenesis(_ context.Context, data []byte) error {
	state, err := LoadGenesisState(data)
	if err != nil {
		return err
	}
	if len(state.CheckpointHash) != 0 {
		e.loadHash = state.CheckpointHash
	}
	return nil
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

// BalanceCheckpoint is used for deposits and withdrawals. We want a snapshot to be taken in those events
// but these snapshots should not affect the timing (delta, time between checkpoints). Currently, this call
// generates a full checkpoint, but we probably will change this to be a sparse checkpoint
// only containing changes in balances and (perhaps) network parameters...
func (e *Engine) BalanceCheckpoint() (*types.Snapshot, error) {
	// no time stuff here, for now we're just taking a full snapshot
	cp := e.makeCheckpoint()
	return cp, nil
}

// Checkpoint returns the overall checkpoint
func (e *Engine) Checkpoint(t time.Time) (*types.Snapshot, error) {
	// start time will be zero -> add delta to this time, and return
	if e.nextCP.IsZero() {
		e.nextCP = t.Add(e.delta)
		return nil, nil
	}
	if e.nextCP.After(t) {
		return nil, nil
	}
	e.nextCP = t.Add(e.delta)
	cp := e.makeCheckpoint()
	return cp, nil
}

func (e *Engine) makeCheckpoint() *types.Snapshot {
	cp := &types.Checkpoint{}
	for _, k := range cpOrder {
		comp, ok := e.components[k]
		if !ok {
			continue
		}
		data, err := comp.Checkpoint()
		if err != nil {
			panic(fmt.Errorf("failed to generate checkpoint: %w", err))
		}
		// set the correct field
		cp.Set(k, data)
	}
	snap := &types.Snapshot{}
	// setCheckpoint hides the vega type mess
	if err := snap.SetCheckpoint(cp); err != nil {
		panic(fmt.Errorf("checkpoint could not be created: %w", err))
	}

	return snap
}

// Load - loads checkpoint data for all components by name
func (e *Engine) Load(ctx context.Context, snap *types.Snapshot) error {
	// if no hash was specified, or the hash doesn't match, then don't even attempt to load the checkpoint
	if e.loadHash == nil || !bytes.Equal(e.loadHash, snap.Hash) {
		return nil
	}
	// we found the checkpoint we need to load, set value to nil
	// either the checkpoint was loaded successfully, or it wasn't
	// if this fails, the node goes down
	e.loadHash = nil
	cp, err := snap.GetCheckpoint()
	if err != nil {
		return err
	}
	// check the hash
	if err := snap.Validate(); err != nil {
		return err
	}
	var assets []*types.Asset
	for _, k := range cpOrder {
		cpData := cp.Get(k)
		if len(cpData) == 0 {
			continue
		}
		c, ok := e.components[k]
		if !ok {
			return ErrUnknownCheckpointName // data cannot be restored
		}
		if ac, ok := c.(AssetsState); ok {
			if err := c.Load(cpData); err != nil {
				return err
			}
			assets = ac.GetEnabledAssets()
			continue
		}
		// first enable assets, then load the state
		if cc, ok := c.(CollateralState); ok {
			for _, a := range assets {
				if err := cc.EnableAsset(ctx, *a); err != nil {
					return err
				}
			}
		}
		if err := c.Load(cpData); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) OnTimeElapsedUpdate(ctx context.Context, d time.Duration) error {
	if !e.nextCP.IsZero() {
		// update the time for the next cp
		e.nextCP = e.nextCP.Add(-e.delta).Add(d)
	}
	// update delta
	e.delta = d
	return nil
}
