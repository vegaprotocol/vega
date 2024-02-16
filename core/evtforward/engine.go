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

package evtforward

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/evtforward/ethereum"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
)

const (
	topEngineLogger = "event-forwarder"
	ethereumLogger  = "ethereum"
)

type Engine struct {
	cfg Config
	log *logging.Logger

	ethEngine *ethereum.Engine

	stakingStartingBlock         uint64
	multisigControlStartingBlock uint64
}

func NewEngine(log *logging.Logger, config Config) *Engine {
	topEngineLogger := log.Named(topEngineLogger)
	topEngineLogger.SetLevel(config.Level.Get())

	return &Engine{
		cfg: config,
		log: topEngineLogger,
	}
}

// ReloadConf updates the internal configuration of the Event Forwarder engine.
func (e *Engine) ReloadConf(config Config) {
	e.log.Info("Reloading configuration")

	if e.log.GetLevel() != config.Level.Get() {
		e.log.Debug("Updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", config.Level.String()),
		)
		e.log.SetLevel(config.Level.Get())
	}
	if e.ethEngine != nil {
		e.ethEngine.ReloadConf(config.Ethereum)
	}
}

func (e *Engine) UpdateCollateralStartingBlock(b uint64) {
	e.ethEngine.UpdateCollateralStartingBlock(b)
}

func (e *Engine) UpdateStakingStartingBlock(b uint64) {
	e.stakingStartingBlock = b
	e.ethEngine.UpdateStakingStartingBlock(b)
}

func (e *Engine) UpdateMultisigControlStartingBlock(b uint64) {
	e.multisigControlStartingBlock = b
	e.ethEngine.UpdateMultiSigControlStartingBlock(b)
}

func (e *Engine) SetupEthereumEngine(
	client ethereum.Client,
	forwarder ethereum.Forwarder,
	config ethereum.Config,
	ethCfg *types.EthereumConfig,
	assets ethereum.Assets,
) error {
	if e.log.IsDebug() {
		e.log.Debug("Ethereum configuration has been loaded")
	}

	if e.ethEngine != nil {
		if e.log.IsDebug() {
			e.log.Debug("Stopping previous Ethereum Event Forwarder")
		}
		e.Stop()
	}

	if e.log.IsDebug() {
		e.log.Debug("Setting up the Ethereum Event Forwarder")
	}

	ethLogger := e.log.Named(ethereumLogger)
	ethLogger.SetLevel(config.Level.Get())

	filterer, err := ethereum.NewLogFilterer(
		e.cfg.Ethereum,
		ethLogger,
		client,
		ethCfg.CollateralBridge(),
		ethCfg.StakingBridge(),
		ethCfg.VestingBridge(),
		ethCfg.MultiSigControl(),
		assets,
	)
	if err != nil {
		return fmt.Errorf("couldn't create the log filterer: %w", err)
	}

	e.ethEngine = ethereum.NewEngine(
		e.cfg.Ethereum,
		ethLogger,
		filterer,
		forwarder,
		ethCfg.StakingBridge(),
		ethCfg.VestingBridge(),
		ethCfg.MultiSigControl(),
		ethCfg.ChainID(),
	)

	e.UpdateCollateralStartingBlock(filterer.CurrentHeight(context.Background()))

	if e.multisigControlStartingBlock != 0 {
		e.ethEngine.UpdateMultiSigControlStartingBlock(e.multisigControlStartingBlock)
	}
	if e.stakingStartingBlock != 0 {
		e.ethEngine.UpdateStakingStartingBlock(e.stakingStartingBlock)
	}

	e.Start()

	return nil
}

func (e *Engine) Start() {
	if e.ethEngine != nil {
		go func() {
			e.log.Info("Starting the Ethereum Event Forwarder")
			e.ethEngine.Start()
		}()
	}
}

func (e *Engine) Stop() {
	if e.ethEngine != nil {
		e.log.Info("Stopping the Ethereum Event Forwarder")
		e.ethEngine.Stop()
	}
	e.log.Info("The Event Forwarder engine stopped")
}

// NoopEngine can be use as a stub for the Engine. It does nothing.
type NoopEngine struct {
	log *logging.Logger
}

func NewNoopEngine(log *logging.Logger, config Config) *NoopEngine {
	topEngineLogger := log.Named(topEngineLogger)
	topEngineLogger.SetLevel(config.Level.Get())

	return &NoopEngine{
		log: topEngineLogger,
	}
}

func (e *NoopEngine) ReloadConf(_ Config) {
	if e.log.IsDebug() {
		e.log.Debug("Reloading Ethereum configuration is a no-op")
	}
}

func (e *NoopEngine) UpdateCollateralStartingBlock(b uint64) {}

func (e *NoopEngine) UpdateStakingStartingBlock(b uint64) {}

func (e *NoopEngine) UpdateMultisigControlStartingBlock(b uint64) {}

func (e *NoopEngine) SetupEthereumEngine(
	_ ethereum.Client,
	_ ethereum.Forwarder,
	_ ethereum.Config,
	_ *types.EthereumConfig,
	_ ethereum.Assets,
) error {
	if e.log.IsDebug() {
		e.log.Debug("Starting Ethereum configuration is a no-op")
	}

	return nil
}

func (e *NoopEngine) Start() {
	if e.log.IsDebug() {
		e.log.Debug("Starting Ethereum configuration is a no-op")
	}
}

func (e *NoopEngine) Stop() {
	if e.log.IsDebug() {
		e.log.Debug("Stopping Ethereum configuration is a no-op")
	}
}
