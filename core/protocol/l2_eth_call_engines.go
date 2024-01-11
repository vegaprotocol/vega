package protocol

// circling around import cycle here...

import (
	"context"

	"code.vegaprotocol.io/vega/core/client/eth"
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	"code.vegaprotocol.io/vega/core/datasource/external/ethverifier"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
)

type SpecActivationListener func(listener spec.SpecActivationsListener)

type L2EthCallEngines struct {
	log         *logging.Logger
	cfg         ethcall.Config
	isValidator bool
	clients     *eth.L2Clients
	forwarder   ethcall.Forwarder

	// chain id -> engine
	engines                map[string]*ethcall.Engine
	specActivationListener SpecActivationListener
}

func NewL2EthCallEngines(log *logging.Logger, cfg ethcall.Config, isValidator bool, clients *eth.L2Clients, forwarder ethcall.Forwarder, specActivationListener SpecActivationListener) *L2EthCallEngines {
	return &L2EthCallEngines{
		log:                    log,
		cfg:                    cfg,
		isValidator:            isValidator,
		clients:                clients,
		forwarder:              forwarder,
		engines:                map[string]*ethcall.Engine{},
		specActivationListener: specActivationListener,
	}
}

func (v *L2EthCallEngines) GetOrInstantiate(chainID string) (ethverifier.EthCallEngine, error) {

	if e, ok := v.engines[chainID]; ok {
		return e, nil
	}

	v.log.Panic("should be instantiated by now really?")
	return nil, nil
}

func (v *L2EthCallEngines) OnEthereumL2ConfigsUpdated(
	ctx context.Context, ethCfg *types.EthereumL2Configs,
) error {
	// new L2 configured, instatiate the verifier for it.
	for _, c := range ethCfg.Configs {
		// if already exists, do nothing
		if _, ok := v.engines[c.ChainID]; ok {
			continue
		}

		clt, _, ok := v.clients.Get(c.NetworkID)
		if !ok {
			v.log.Panic("ethereum client not configured for L2",
				logging.String("chain-id", c.ChainID),
				logging.String("network-id", c.NetworkID),
			)
		}

		e := ethcall.NewEngine(v.log, v.cfg, v.isValidator, clt, v.forwarder)
		v.engines[c.ChainID] = e

		// start it here?
		e.Start()

		// setup activation listener
		v.specActivationListener(v.engines[c.ChainID])
	}

	return nil
}
