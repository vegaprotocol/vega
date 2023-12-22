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

package protocol

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/api"
	"code.vegaprotocol.io/vega/core/blockchain"
	"code.vegaprotocol.io/vega/core/broker"
	ethclient "code.vegaprotocol.io/vega/core/client/eth"
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/evtforward"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/processor"
	"code.vegaprotocol.io/vega/core/protocolupgrade"
	"code.vegaprotocol.io/vega/core/spam"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/vegatime"
	"code.vegaprotocol.io/vega/libs/subscribers"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/blang/semver"
)

var Version = semver.MustParse("0.1.0")

type Protocol struct {
	*processor.App

	log *logging.Logger

	confWatcher     *config.Watcher
	confListenerIDs []int

	services *allServices
}

const namedLogger = "protocol"

func New(
	ctx context.Context,
	confWatcher *config.Watcher,
	log *logging.Logger,
	cancel func(),
	stopBlockchain func() error,
	nodewallets *nodewallets.NodeWallets,
	ethClient *ethclient.Client,
	ethConfirmation *ethclient.EthereumConfirmations,
	blockchainClient *blockchain.Client,
	vegaPaths paths.Paths,
	stats *stats.Stats,
	l2Clients *ethclient.L2Clients,
) (p *Protocol, err error) {
	log = log.Named(namedLogger)

	defer func() {
		if err != nil {
			log.Error("unable to start protocol", logging.Error(err))
			return
		}

		ids := p.confWatcher.OnConfigUpdateWithID(
			func(cfg config.Config) { p.ReloadConf(cfg.Processor) },
		)
		p.confListenerIDs = ids
	}()

	svcs, err := newServices(
		ctx, log, confWatcher, nodewallets, ethClient, ethConfirmation, blockchainClient, vegaPaths, stats, l2Clients,
	)
	if err != nil {
		return nil, err
	}

	proto := &Protocol{
		App: processor.NewApp(
			log,
			svcs.vegaPaths,
			confWatcher.Get().Processor,
			cancel,
			stopBlockchain,
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
				Engine:                    svcs.oracle,
				Adaptors:                  svcs.oracleAdaptors,
				EthereumOraclesVerifier:   svcs.ethereumOraclesVerifier,
				EthereumL2OraclesVerifier: svcs.l2Verifiers,
			},
			svcs.delegation,
			svcs.limits,
			svcs.stakeVerifier,
			svcs.checkpoint,
			svcs.spam,
			svcs.pow,
			svcs.stakingAccounts,
			svcs.snapshotEngine,
			svcs.statevar,
			svcs.teamsEngine,
			svcs.referralProgram,
			svcs.volumeDiscount,
			svcs.blockchainClient,
			svcs.erc20MultiSigTopology,
			stats.GetVersion(),
			svcs.protocolUpgradeEngine,
			svcs.codec,
			svcs.gastimator,
			svcs.ethCallEngine,
			svcs.collateral,
		),
		log:         log,
		confWatcher: confWatcher,
		services:    svcs,
	}

	proto.services.netParams.Watch(
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMaxBatchSize,
			Watcher: proto.App.OnSpamProtectionMaxBatchSizeUpdate,
		},
	)

	return proto, nil
}

// Start will start the protocol, this means it's ready to process
// blocks from the blockchain.
func (n *Protocol) Start(ctx context.Context) error {
	if err := n.services.snapshotEngine.Start(ctx); err != nil {
		return fmt.Errorf("could not start the snapshot engine: %w", err)
	}
	return nil
}

// Stop will stop all services of the protocol.
func (n *Protocol) Stop() error {
	// unregister conf listeners
	n.log.Info("Stopping protocol services")
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

func (n *Protocol) GetPoW() api.ProofOfWorkParams {
	return n.services.pow
}

func (n *Protocol) GetProtocolUpgradeService() *protocolupgrade.Engine {
	return n.services.protocolUpgradeEngine
}

func (n *Protocol) GetSpamEngine() *spam.Engine {
	return n.services.spam
}

func (n *Protocol) GetPowEngine() processor.PoWEngine {
	return n.services.pow
}
