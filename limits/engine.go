package limits

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/logging"
)

type Engine struct {
	log *logging.Logger
	cfg Config

	blockCount uint16

	// are these action possible?
	canProposeMarket, canProposeAsset, bootstrapFinished bool

	// Settings from the genesis state
	proposeMarketEnabled, proposeAssetEnabled         bool
	proposeMarketEnabledFrom, proposeAssetEnabledFrom time.Time
	bootstrapBlockCount                               uint16

	genesisLoaded bool

	// snapshot state
	lss *limitsSnapshotState
}

func New(log *logging.Logger, cfg Config) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Engine{
		log: log,
		cfg: cfg,
		lss: &limitsSnapshotState{changed: true},
	}
}

// UponGenesis load the limits from the genesis state.
func (e *Engine) UponGenesis(ctx context.Context, rawState []byte) (err error) {
	e.log.Debug("Entering limits.Engine.UponGenesis")
	defer func() {
		if err != nil {
			e.log.Debug("Failure in limits.Engine.UponGenesis", logging.Error(err))
		} else {
			e.log.Debug("Leaving limits.Engine.UponGenesis without error")
		}
		e.genesisLoaded = true
		e.lss.changed = true
	}()

	state, err := LoadGenesisState(rawState)
	if err != nil && err != ErrNoLimitsGenesisState {
		e.log.Error("unable to load genesis state",
			logging.Error(err))
		return err
	}

	if err == ErrNoLimitsGenesisState {
		defaultState := DefaultGenesisState()
		state = &defaultState
	}

	// set enabled by default if not genesis state
	if state == nil {
		e.proposeAssetEnabled = true
		e.proposeMarketEnabled = true
		return nil
	}

	e.proposeAssetEnabled = state.ProposeAssetEnabled
	e.proposeMarketEnabled = state.ProposeMarketEnabled
	e.proposeAssetEnabledFrom = timeFromPtr(state.ProposeAssetEnabledFrom)
	e.proposeMarketEnabledFrom = timeFromPtr(state.ProposeMarketEnabledFrom)
	e.bootstrapBlockCount = state.BootstrapBlockCount

	e.log.Info("loaded limits genesis state",
		logging.String("state", fmt.Sprintf("%#v", *state)))

	return nil
}

func (e *Engine) OnTick(_ context.Context, t time.Time) {
	if !e.genesisLoaded || (e.bootstrapFinished && e.canProposeAsset && e.canProposeMarket) {
		return
	}

	if !e.bootstrapFinished {
		e.blockCount++
		if e.blockCount > e.bootstrapBlockCount {
			e.log.Info("bootstraping period finished, transactions are now allowed")
			e.bootstrapFinished = true
		}
		e.lss.changed = true
	}

	if !e.canProposeMarket && e.bootstrapFinished && e.proposeMarketEnabled && t.After(e.proposeMarketEnabledFrom) {
		e.log.Info("all required conditions are met, proposing markets is now allowed")
		e.canProposeMarket = true
		e.lss.changed = true
	}
	if !e.canProposeAsset && e.bootstrapFinished && e.proposeAssetEnabled && t.After(e.proposeAssetEnabledFrom) {
		e.log.Info("all required conditions are met, proposing assets is now allowed")
		e.canProposeAsset = true
		e.lss.changed = true
	}
}

func (e *Engine) CanProposeMarket() bool {
	return e.canProposeMarket
}

func (e *Engine) CanProposeAsset() bool {
	return e.canProposeAsset
}

func (e *Engine) CanTrade() bool {
	return e.canProposeAsset && e.canProposeMarket
}

func (e *Engine) BootstrapFinished() bool {
	return e.bootstrapFinished
}

func timeFromPtr(tptr *time.Time) time.Time {
	var t time.Time
	if tptr != nil {
		t = *tptr
	}
	return t
}
