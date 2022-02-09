package evtforward

import (
	"fmt"

	"code.vegaprotocol.io/vega/evtforward/ethereum"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
)

const (
	topEngineLogger = "event-forwarder"
	ethereumLogger  = "ethereum"
)

type Engine struct {
	log *logging.Logger

	ethEngine *ethereum.Engine
}

func NewEngine(log *logging.Logger, config Config) *Engine {
	topEngineLogger := log.Named(topEngineLogger)
	topEngineLogger.SetLevel(config.Level.Get())

	return &Engine{
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

	e.ethEngine.ReloadConf(config.Ethereum)
}

func (e *Engine) StartEthereumEngine(
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
		ethLogger,
		client,
		ethCfg.CollateralBridge(),
		ethCfg.StakingBridge(),
		ethCfg.VestingBridge(),
		assets,
	)
	if err != nil {
		return fmt.Errorf("couldn't create the log filterer: %w", err)
	}

	e.ethEngine = ethereum.NewEngine(
		ethLogger,
		filterer,
		forwarder,
		ethCfg.StakingBridge(),
		ethCfg.VestingBridge(),
	)

	go func() {
		e.log.Info("Starting the Ethereum Event Forwarder")
		e.ethEngine.Start()
	}()

	return nil
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

func (e *NoopEngine) StartEthereumEngine(
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

func (e *NoopEngine) Stop() {
	if e.log.IsDebug() {
		e.log.Debug("Stopping Ethereum configuration is a no-op")
	}
}
