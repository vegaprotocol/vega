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

package checkpoint

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vegactx "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrUnknownCheckpointName            = errors.New("component for checkpoint not registered")
	ErrComponentWithDuplicateName       = errors.New("multiple components with the same name")
	ErrNoCheckpointExpectedToBeRestored = errors.New("no checkpoint expected to be restored")
	ErrIncompatibleHashes               = errors.New("incompatible hashes")

	cpOrder = []types.CheckpointName{
		types.ValidatorsCheckpoint,            // validators information
		types.AssetsCheckpoint,                // assets are required for collateral to work, and the vote asset needs to be restored
		types.CollateralCheckpoint,            // without balances, governance (proposals, bonds) are difficult
		types.NetParamsCheckpoint,             // net params should go right after assets and collateral, so vote tokens are restored
		types.MarketActivityTrackerCheckpoint, // restore market activity information - needs to happen before governance
		types.ExecutionCheckpoint,             // we should have the parent market state restored before we start loading governance, so successor markets can inherit the correct state
		types.GovernanceCheckpoint,            // depends on all of the above
		types.EpochCheckpoint,                 // restore epoch information... so delegation sequence ID's make sense
		types.MultisigControlCheckpoint,       // restore the staking information, so delegation make sense
		types.StakingCheckpoint,               // restore the staking information, so delegation make sense
		types.DelegationCheckpoint,
		types.PendingRewardsCheckpoint, // pending rewards can basically be reloaded any time
		types.BankingCheckpoint,        // Banking checkpoint needs to be reload any time after collateral

	}
)

// State interface represents system components that need checkpointting
// Name returns the component name (key in engine map)
// Hash returns, obviously, the state hash
// @TODO adding func to get the actual data
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/state_mock.go -package mocks code.vegaprotocol.io/vega/core/checkpoint State
type State interface {
	Name() types.CheckpointName
	Checkpoint() ([]byte, error)
	Load(ctx context.Context, checkpoint []byte) error
}

// AssetsState is a bit of a hacky way to get the assets that were enabled when checkpoint was reloaded, so we can enable them in the collateral engine
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_state_mock.go -package mocks code.vegaprotocol.io/vega/core/checkpoint AssetsState
type AssetsState interface {
	State
	GetEnabledAssets() []*types.Asset
}

// CollateralState is part 2 of the hacky way to enable the assets required to load the collateral state
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/collateral_state_mock.go -package mocks code.vegaprotocol.io/vega/core/checkpoint CollateralState
type CollateralState interface {
	State
	EnableAsset(ctx context.Context, asset types.Asset) error
}

type Engine struct {
	log *logging.Logger

	components map[types.CheckpointName]State
	loadHash   []byte
	nextCP     time.Time
	delta      time.Duration

	// snapshot fields
	state   *types.PayloadCheckpoint
	data    []byte
	updated bool
	snapErr error
	poll    chan struct{}

	onCheckpointLoadedCB func(context.Context)
}

func New(log *logging.Logger, cfg Config, components ...State) (*Engine, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	e := &Engine{
		log:        log,
		components: make(map[types.CheckpointName]State, len(components)),
		nextCP:     time.Time{},
		state: &types.PayloadCheckpoint{
			Checkpoint: &types.CPState{},
		},
	}
	for _, c := range components {
		if err := e.addComponent(c); err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e *Engine) RegisterOnCheckpointLoaded(f func(context.Context)) {
	e.onCheckpointLoadedCB = f
}

func (e *Engine) UponGenesis(ctx context.Context, data []byte) (err error) {
	e.log.Debug("Entering checkpoint.Engine.UponGenesis")
	defer func() {
		if err != nil {
			e.log.Debug("Failure in checkpoint.Engine.UponGenesis", logging.Error(err))
		} else {
			e.log.Debug("Leaving checkpoint.Engine.UponGenesis without error")
		}
	}()

	state, err := LoadGenesisState(data)
	if err != nil {
		return err
	}

	// first is there a hash
	if state != nil && len(state.CheckpointHash) != 0 {
		e.loadHash, err = hex.DecodeString(state.CheckpointHash)
		e.log.Warn("Checkpoint restore enabled",
			logging.String("checkpoint-hash-str", state.CheckpointHash),
			logging.String("checkpoint-hex-encoded", hex.EncodeToString(e.loadHash)),
		)
		if err != nil {
			e.loadHash = nil
			e.log.Panic("Malformed restore hash in genesis file",
				logging.Error(err),
			)
		}
	}

	// a hash is set to be loaded
	if len(e.loadHash) > 0 {
		// no loadHash but a state specified.
		if len(state.CheckpointHash) <= 0 {
			e.log.Panic("invalid genesis file, hash specified without state")
		}

		buf, err := base64.StdEncoding.DecodeString(state.CheckpointState)
		if err != nil {
			return fmt.Errorf("invalid genesis file checkpoint.state: %w", err)
		}

		cpt := &types.CheckpointState{}
		if err := cpt.SetState(buf); err != nil {
			return fmt.Errorf("invalid restore checkpoint command: %w", err)
		}

		// now we can proceed with loading it.
		if err := e.load(ctx, cpt); err != nil {
			return fmt.Errorf("could not load checkpoint: %w", err)
		}
	}

	// if state nil, no checkpoint to load, let's just call
	// the onCheckPointloaded stuff to notify engine they don't have to wait for a
	// checkpoint to get in business
	if state == nil || len(state.CheckpointHash) <= 0 {
		e.onCheckpointLoaded(ctx)
	}

	return nil
}

// Add used to add/register components after the engine has been instantiated already
// this is mainly used to make testing easier.
func (e *Engine) Add(comps ...State) error {
	for _, c := range comps {
		if err := e.addComponent(c); err != nil {
			return err
		}
	}
	return nil
}

// add component, but check for duplicate names.
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

// BalanceCheckpoint is used for deposits and withdrawals. We want a checkpoint to be taken in those events
// but these checkpoints should not affect the timing (delta, time between checkpoints). Currently, this call
// generates a full checkpoint, but we probably will change this to be a sparse checkpoint
// only containing changes in balances and (perhaps) network parameters...
func (e *Engine) BalanceCheckpoint(ctx context.Context) (*types.CheckpointState, error) {
	// no time stuff here, for now we're just taking a full checkpoint
	cp := e.makeCheckpoint(ctx)
	return cp, nil
}

// Checkpoint returns the overall checkpoint.
func (e *Engine) Checkpoint(ctx context.Context, t time.Time) (*types.CheckpointState, error) {
	// start time will be zero -> add delta to this time, and return

	if e.nextCP.IsZero() {
		e.setNextCP(t.Add(e.delta))
		return nil, nil
	}
	if e.nextCP.After(t) {
		return nil, nil
	}
	e.setNextCP(t.Add(e.delta))
	cp := e.makeCheckpoint(ctx)
	return cp, nil
}

func (e *Engine) makeCheckpoint(ctx context.Context) *types.CheckpointState {
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
	// add block height to checkpoint
	h, _ := vegactx.BlockHeightFromContext(ctx)
	if err := cp.SetBlockHeight(int64(h)); err != nil {
		e.log.Panic("could not set block height", logging.Error(err))
	}
	cpState := &types.CheckpointState{}
	// setCheckpoint hides the vega type mess
	if err := cpState.SetCheckpoint(cp); err != nil {
		panic(fmt.Errorf("checkpoint could not be created: %w", err))
	}

	e.log.Debug("checkpoint taken", logging.Uint64("block-height", h))
	return cpState
}

// load - loads checkpoint data for all components by name.
func (e *Engine) load(ctx context.Context, cpt *types.CheckpointState) error {
	if len(e.loadHash) != 0 {
		hashDiff := bytes.Compare(e.loadHash, cpt.Hash)

		log := e.log.Info
		if hashDiff != 0 {
			log = e.log.Warn
		}
		log("Checkpoint hash reload requested",
			logging.String("hash-to-load", hex.EncodeToString(e.loadHash)),
			logging.String("checkpoint-hash", hex.EncodeToString(cpt.Hash)),
			logging.Int("hash-diff", hashDiff),
		)
	}

	if err := e.ValidateCheckpoint(cpt); err != nil {
		return err
	}
	// we found the checkpoint we need to load, set value to nil
	// either the checkpoint was loaded successfully, or it wasn't
	// if this fails, the node goes down
	e.loadHash = nil
	cp, err := cpt.GetCheckpoint()
	if err != nil {
		return err
	}
	// check the hash
	if err := cpt.Validate(); err != nil {
		return err
	}
	var (
		assets                 []*types.Asset
		doneAssets, doneCollat bool // just avoids type asserting all components
	)
	for _, k := range cpOrder {
		cpData := cp.Get(k)
		if len(cpData) == 0 {
			continue
		}
		c, ok := e.components[k]
		if !ok {
			return ErrUnknownCheckpointName // data cannot be restored
		}
		if !doneAssets {
			if ac, ok := c.(AssetsState); ok {
				if err := c.Load(ctx, cpData); err != nil {
					return err
				}
				assets = ac.GetEnabledAssets()
				doneAssets = true
				continue
			}
		}
		// first enable assets, then load the state
		if !doneCollat {
			if cc, ok := c.(CollateralState); ok {
				for _, a := range assets {
					// ignore this error, if the asset is already enabled, that's fine
					// we can carry on as though nothing happened
					if err := cc.EnableAsset(ctx, *a); err != nil {
						e.log.Debug("Asset already enabled",
							logging.String("asset-id", a.ID),
							logging.Error(err),
						)
					}
				}
				doneCollat = true
			}
		}
		if err := c.Load(ctx, cpData); err != nil {
			return err
		}
	}

	// seems like we went through it all without issue
	// we can execute the callback
	e.onCheckpointLoaded(ctx)

	return nil
}

func (e *Engine) ValidateCheckpoint(cpt *types.CheckpointState) error {
	// if no hash was specified, or the hash doesn't match, then don't even attempt to load the checkpoint
	if e.loadHash == nil {
		return ErrNoCheckpointExpectedToBeRestored
	}
	if !bytes.Equal(e.loadHash, cpt.Hash) {
		return fmt.Errorf("received(%v), expected(%v): %w", hex.EncodeToString(cpt.Hash), hex.EncodeToString(e.loadHash), ErrIncompatibleHashes)
	}
	return nil
}

func (e *Engine) OnTimeElapsedUpdate(ctx context.Context, d time.Duration) error {
	if !e.nextCP.IsZero() {
		// update the time for the next cp
		e.setNextCP(e.nextCP.Add(-e.delta).Add(d))
	}
	// update delta
	e.delta = d
	return nil
}

// onCheckpointLoaded will call the OnCheckpointLoaded method for
// all checkpoint providers (if it exists).
func (e *Engine) onCheckpointLoaded(ctx context.Context) {
	if e.onCheckpointLoadedCB != nil {
		e.onCheckpointLoadedCB(ctx)
	}
}
