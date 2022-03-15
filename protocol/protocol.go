package protocol

import (
	"context"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/broker"
	ethclient "code.vegaprotocol.io/vega/client/eth"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/vegatime"
	"github.com/blang/semver"
)

var (
	Version = semver.MustParse("0.1.0")
)

type Protocol struct {
	*processor.App

	log *logging.Logger

	confWatcher     *config.Watcher
	confListenerIDs []int

	services *allServices
}

func New(
	ctx context.Context,
	confWatcher *config.Watcher,
	log *logging.Logger,
	cancel func(),
	nodewallets *nodewallets.NodeWallets,
	ethClient *ethclient.Client,
	ethConfirmation *ethclient.EthereumConfirmations,
	blockchainClient *blockchain.Client,
	vegaPaths paths.Paths,
	stats *stats.Stats,
) (p *Protocol, err error) {
	defer func() {
		if err != nil {
			ids := p.confWatcher.OnConfigUpdateWithID(
				func(cfg config.Config) { p.ReloadConf(cfg.Processor) },
			)
			p.confListenerIDs = ids
		}
	}()

	svcs, err := newServices(
		ctx, log, confWatcher, nodewallets, ethClient, ethConfirmation, blockchainClient, vegaPaths, stats,
	)
	if err != nil {
		return nil, err
	}

	return &Protocol{
		App: processor.NewApp(
			log,
			svcs.vegaPaths,
			confWatcher.Get().Processor,
			cancel,
			svcs.assets,
			svcs.banking,
			svcs.broker,
			svcs.witness,
			svcs.eventForwarder,
			svcs.executionEngine,
			svcs.genesisHandler,
			svcs.governance,
			svcs.notary,
			svcs.stats.Blockchain,
			svcs.timeService,
			svcs.epochService,
			svcs.topology,
			svcs.netParams,
			&processor.Oracle{
				Engine:   svcs.oracle,
				Adaptors: svcs.oracleAdaptors,
			},
			svcs.delegation,
			svcs.limits,
			svcs.stakeVerifier,
			svcs.checkpoint,
			svcs.spam,
			svcs.stakingAccounts,
			svcs.snapshot,
			svcs.statevar,
			svcs.blockchainClient,
			svcs.erc20MultiSigTopology,
			stats.GetVersion(),
		),
		log:         log,
		confWatcher: confWatcher,
		services:    svcs,
	}, nil
}

// Start will start the protocol, this means it's ready to process
// blocks from the blockchain
func (n *Protocol) Start() error {
	return nil
}

// Stop will stop all services of the protocol
func (n *Protocol) Stop() error {
	// unregister conf listeners
	n.confWatcher.Unregister(n.confListenerIDs)
	n.services.Stop()

	return nil
}

func (n *Protocol) Protocol() semver.Version {
	return Version
}

func (n *Protocol) GetEventForwarder() *evtforward.Forwarder {
	return n.services.eventForwarder
}

func (n *Protocol) GetTimeService() *vegatime.Svc {
	return n.services.timeService
}

func (n *Protocol) GetEventService() *subscribers.Service {
	return n.services.eventService
}

func (n *Protocol) GetBroker() *broker.Broker {
	return n.services.broker
}
