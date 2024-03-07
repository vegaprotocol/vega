// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package limits

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
)

type Engine struct {
	log    *logging.Logger
	cfg    Config
	broker Broker

	timeService TimeService

	// are these action possible?
	canProposeMarket, canProposeAsset bool

	// Settings from the genesis state
	proposeMarketEnabled, proposeAssetEnabled, proposeSpotMarketEnabled, proposePerpsMarketEnabled, useAMMEnabled bool
	proposeMarketEnabledFrom, proposeAssetEnabledFrom                                                             time.Time

	genesisLoaded bool

	// snapshot state
	lss *limitsSnapshotState
}

type Broker interface {
	Send(event events.Event)
}

// TimeService provide the time of the vega node using the tm time.
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/core/limits TimeService
type TimeService interface {
	GetTimeNow() time.Time
}

func New(log *logging.Logger, cfg Config, tm TimeService, broker Broker) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Engine{
		log:         log,
		cfg:         cfg,
		lss:         &limitsSnapshotState{},
		broker:      broker,
		timeService: tm,
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
	}()

	state, err := LoadGenesisState(rawState)
	if err != nil && err != ErrNoLimitsGenesisState {
		e.log.Error("unable to load genesis state",
			logging.Error(err))
		return err
	}

	defer func() {
		e.sendEvent(ctx)
	}()

	if err == ErrNoLimitsGenesisState {
		defaultState := DefaultGenesisState()
		state = &defaultState
	}

	// set enabled by default if not genesis state
	if state == nil {
		e.proposeAssetEnabled = true
		e.proposeMarketEnabled = true
	} else {
		e.proposeAssetEnabled = state.ProposeAssetEnabled
		e.proposeMarketEnabled = state.ProposeMarketEnabled
	}

	// at this point we only know about the genesis state
	// of the limits, so we should set the can* fields to
	// this state
	e.canProposeAsset = e.proposeAssetEnabled
	e.canProposeMarket = e.proposeMarketEnabled

	e.log.Info("loaded limits genesis state",
		logging.String("state", fmt.Sprintf("%#v", *state)))

	return nil
}

func (e *Engine) OnLimitsProposeMarketEnabledFromUpdate(ctx context.Context, date string) error {
	// already validated by the netparams
	// no need to check it again, this is a valid date
	if len(date) <= 0 {
		e.proposeMarketEnabledFrom = time.Time{}
	} else {
		t, _ := time.Parse(time.RFC3339, date)
		e.proposeMarketEnabledFrom = t
	}
	e.onUpdate(e.timeService.GetTimeNow())
	e.sendEvent(ctx)

	return nil
}

func (e *Engine) OnLimitsProposeSpotMarketEnabledFromUpdate(ctx context.Context, enabled int64) error {
	e.proposeSpotMarketEnabled = enabled == 1
	e.sendEvent(ctx)
	return nil
}

func (e *Engine) OnLimitsProposePerpsMarketEnabledFromUpdate(ctx context.Context, enabled int64) error {
	e.proposePerpsMarketEnabled = enabled == 1
	e.sendEvent(ctx)
	return nil
}

func (e *Engine) OnLimitsProposeAMMEnabledUpdate(ctx context.Context, enabled int64) error {
	e.useAMMEnabled = enabled == 1
	e.sendEvent(ctx)
	return nil
}

func (e *Engine) OnLimitsProposeAssetEnabledFromUpdate(ctx context.Context, date string) error {
	// already validated by the netparams
	// no need to check it again, this is a valid date
	if len(date) <= 0 {
		e.proposeAssetEnabledFrom = time.Time{}
	} else {
		t, _ := time.Parse(time.RFC3339, date)
		e.proposeAssetEnabledFrom = t
	}

	e.onUpdate(e.timeService.GetTimeNow())
	e.sendEvent(ctx)

	return nil
}

func (e *Engine) OnTick(ctx context.Context, t time.Time) {
	canProposeAsset, canProposeMarket := e.canProposeAsset, e.canProposeMarket
	defer func() {
		if canProposeAsset != e.canProposeAsset || canProposeMarket != e.canProposeMarket {
			e.sendEvent(ctx)
		}
	}()
	e.onUpdate(t)
}

func (e *Engine) onUpdate(t time.Time) {
	//  if propose market enabled in genesis
	if e.proposeMarketEnabled {
		// we can propose a market and a new date have been set in the future
		if e.canProposeMarket && t.Before(e.proposeMarketEnabledFrom) {
			e.log.Info("proposing market is now disabled")
			e.canProposeMarket = false
		}

		// we can't propose a market for now, is the date in the past?
		if !e.canProposeMarket && t.After(e.proposeMarketEnabledFrom) {
			e.log.Info("all required conditions are met, proposing markets is now allowed")
			e.canProposeMarket = true
		}
	}

	//  if propose market enabled in genesis
	if e.proposeAssetEnabled {
		// we can propose a market and a new date have been set in the future
		if e.canProposeAsset && t.Before(e.proposeAssetEnabledFrom) {
			e.log.Info("proposing asset have been disabled")
			e.canProposeAsset = false
		}

		if !e.canProposeAsset && t.After(e.proposeAssetEnabledFrom) {
			e.log.Info("all required conditions are met, proposing assets is now allowed")
			e.canProposeAsset = true
		}
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

func (e *Engine) CanProposeSpotMarket() bool {
	return e.proposeSpotMarketEnabled
}

func (e *Engine) CanProposePerpsMarket() bool {
	return e.proposePerpsMarketEnabled
}

func (e *Engine) CanUseAMMPool() bool {
	return e.useAMMEnabled
}

func (e *Engine) sendEvent(ctx context.Context) {
	limits := vega.NetworkLimits{
		CanProposeMarket:          e.canProposeMarket,
		CanProposeAsset:           e.canProposeAsset,
		ProposeMarketEnabled:      e.proposeMarketEnabled,
		ProposeAssetEnabled:       e.proposeAssetEnabled,
		GenesisLoaded:             e.genesisLoaded,
		CanProposeSpotMarket:      e.proposeSpotMarketEnabled,
		CanProposePerpetualMarket: e.proposePerpsMarketEnabled,
		CanUseAmm:                 e.useAMMEnabled,
	}

	if !e.proposeMarketEnabledFrom.IsZero() {
		limits.ProposeMarketEnabledFrom = e.proposeAssetEnabledFrom.UnixNano()
	}

	if !e.proposeAssetEnabledFrom.IsZero() {
		limits.ProposeAssetEnabledFrom = e.proposeAssetEnabledFrom.UnixNano()
	}

	event := events.NewNetworkLimitsEvent(ctx, &limits)
	e.broker.Send(event)
}
