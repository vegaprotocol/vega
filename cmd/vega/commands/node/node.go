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

package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/core/admin"
	"code.vegaprotocol.io/vega/core/api"
	"code.vegaprotocol.io/vega/core/api/rest"
	"code.vegaprotocol.io/vega/core/blockchain"
	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/blockchain/nullchain"
	ethclient "code.vegaprotocol.io/vega/core/client/eth"
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/coreapi"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/protocol"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"code.vegaprotocol.io/vega/version"

	"github.com/cometbft/cometbft/abci/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"google.golang.org/grpc"
)

var ErrUnknownChainProvider = errors.New("unknown chain provider")

type Command struct {
	ctx    context.Context
	cancel context.CancelFunc

	Log *logging.Logger

	pproffhandlr *pprof.Pprofhandler
	stats        *stats.Stats

	conf        config.Config
	confWatcher *config.Watcher

	nullBlockchain   *nullchain.NullBlockchain
	blockchainServer *blockchain.Server
	blockchainClient *blockchain.Client

	nodeWallets          *nodewallets.NodeWallets
	nodeWalletPassphrase string

	vegaPaths paths.Paths

	ethClient        *ethclient.Client
	ethConfirmations *ethclient.EthereumConfirmations
	l2Clients        *ethclient.L2Clients

	abciApp  *appW
	protocol *protocol.Protocol

	// APIs
	grpcServer  *api.GRPC
	proxyServer *rest.ProxyServer
	adminServer *admin.Server
	coreService *coreapi.Service

	tmNode *abci.TmNode
}

func (n *Command) Run(
	confWatcher *config.Watcher,
	vegaPaths paths.Paths,
	nodeWalletPassphrase, tmHome, networkURL, network string,
	log *logging.Logger,
) error {
	n.Log.Info("starting vega",
		logging.String("version", version.Get()),
		logging.String("commit-hash", version.GetCommitHash()),
	)

	n.confWatcher = confWatcher
	n.nodeWalletPassphrase = nodeWalletPassphrase

	n.conf = confWatcher.Get()
	n.vegaPaths = vegaPaths

	if err := n.setupCommon(); err != nil {
		return err
	}

	if err := n.loadNodeWallets(); err != nil {
		return err
	}

	if err := n.startBlockchainClients(); err != nil {
		return err
	}

	// TODO(): later we will want to select what version of the protocol
	// to run, most likely via configuration, so we can use legacy or current
	var err error
	n.protocol, err = protocol.New(
		n.ctx,
		n.confWatcher,
		n.Log,
		n.cancel,
		n.stopBlockchain,
		n.nodeWallets,
		n.ethClient,
		n.ethConfirmations,
		n.blockchainClient,
		vegaPaths,
		n.stats,
		n.l2Clients,
	)
	if err != nil {
		return err
	}

	if err := n.startAPIs(); err != nil {
		return fmt.Errorf("could not start the core APIs: %w", err)
	}

	// The protocol must be started after the API, otherwise nobody is listening
	// to the internal events emitted during that phase (like during the state
	// restoration), which will cause issues to APIs consumer like system tests.
	if err := n.protocol.Start(n.ctx); err != nil {
		return fmt.Errorf("could not start the core: %w", err)
	}

	// if a chain is being replayed tendermint does this during the initial handshake with the
	// app and does so synchronously. We to need to set this off in a goroutine so we can catch any
	// SIGTERM during that replay and shutdown properly
	errCh := make(chan error)
	go func() {
		defer func() {
			// if a consensus failure happens during replay tendermint panics
			// we need to catch it so we can call shutdown and then re-panic
			if r := recover(); r != nil {
				n.Stop()
				panic(r)
			}
		}()
		if err := n.startBlockchain(log, tmHome, network, networkURL); err != nil {
			errCh <- err
		}
		// start the nullblockchain if we are in that mode, it *needs* to be after we've started the gRPC server
		// otherwise it'll start calling init-chain and all the way before we're ready.
		if n.conf.Blockchain.ChainProvider == blockchain.ProviderNullChain {
			if err := n.nullBlockchain.StartServer(); err != nil {
				errCh <- err
			}
		}
	}()

	// at this point all is good, and we should be started, we can
	// just wait for signals or whatever
	n.Log.Info("Vega startup complete",
		logging.String("node-mode", string(n.conf.NodeMode)))

	// wait for possible protocol upgrade, or user exit
	if err := n.wait(errCh); err != nil {
		return err
	}

	return n.Stop()
}

func (n *Command) wait(errCh <-chan error) error {
	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)
	for {
		select {
		case sig := <-gracefulStop:
			n.Log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
			return nil
		case e := <-errCh:
			n.Log.Error("problem starting blockchain", logging.Error(e))
			return e
		case <-n.ctx.Done():
			// nothing to do
			return nil
		}
	}
}

func (n *Command) stopBlockchain() error {
	if n.blockchainServer == nil {
		return nil
	}
	return n.blockchainServer.Stop()
}

func (n *Command) Stop() error {
	upStatus := n.protocol.GetProtocolUpgradeService().GetUpgradeStatus()

	// Blockchain server has been already stopped by the app during the upgrade.
	// Calling stop again would block forever.
	if n.blockchainServer != nil && !upStatus.ReadyToUpgrade {
		n.blockchainServer.Stop()
	}
	if n.protocol != nil {
		n.protocol.Stop()
	}
	if n.grpcServer != nil {
		n.grpcServer.Stop()
	}
	if n.proxyServer != nil {
		n.proxyServer.Stop()
	}
	if n.adminServer != nil {
		n.adminServer.Stop()
	}

	if n.conf.IsValidator() {
		if err := n.nodeWallets.Ethereum.Cleanup(); err != nil {
			n.Log.Error("couldn't clean up Ethereum node wallet", logging.Error(err))
		}
	}

	var err error
	if n.pproffhandlr != nil {
		err = n.pproffhandlr.Stop()
	}

	n.Log.Info("Vega shutdown complete",
		logging.String("version", version.Get()),
		logging.String("version-hash", version.GetCommitHash()))

	n.Log.Sync()
	n.cancel()

	// Blockchain server need to be killed as it is stuck in BeginBlock function.
	if upStatus.ReadyToUpgrade {
		return kill()
	}

	return err
}

func (n *Command) startAPIs() error {
	n.grpcServer = api.NewGRPC(
		n.Log,
		n.conf.API,
		n.stats,
		n.blockchainClient,
		n.protocol.GetEventForwarder(),
		n.protocol.GetTimeService(),
		n.protocol.GetEventService(),
		n.protocol.GetPoW(),
		n.protocol.GetSpamEngine(),
		n.protocol.GetPowEngine(),
	)

	n.coreService = coreapi.NewService(n.ctx, n.Log, n.conf.CoreAPI, n.protocol.GetBroker())
	n.grpcServer.RegisterService(func(server *grpc.Server) {
		apipb.RegisterCoreStateServiceServer(server, n.coreService)
	})

	// watch configs
	n.confWatcher.OnConfigUpdate(
		func(cfg config.Config) { n.grpcServer.ReloadConf(cfg.API) },
	)

	n.proxyServer = rest.NewProxyServer(n.Log, n.conf.API)

	if n.conf.IsValidator() {
		adminServer, err := admin.NewValidatorServer(n.Log, n.conf.Admin, n.vegaPaths, n.nodeWalletPassphrase, n.nodeWallets, n.protocol.GetProtocolUpgradeService())
		if err != nil {
			return err
		}
		n.adminServer = adminServer
	} else {
		adminServer, err := admin.NewNonValidatorServer(n.Log, n.conf.Admin, n.protocol.GetProtocolUpgradeService())
		if err != nil {
			return err
		}
		n.adminServer = adminServer
	}

	go n.grpcServer.Start()
	go n.proxyServer.Start()

	if n.adminServer != nil {
		go n.adminServer.Start()
	}

	return nil
}

func (n *Command) startBlockchain(log *logging.Logger, tmHome, network, networkURL string) error {
	// make sure any env variable is resolved
	tmHome = os.ExpandEnv(tmHome)
	n.abciApp = newAppW(n.protocol.Abci())

	switch n.conf.Blockchain.ChainProvider {
	case blockchain.ProviderTendermint:
		var err error
		// initialise the node
		n.tmNode, err = n.startABCI(log, n.abciApp, tmHome, network, networkURL)
		if err != nil {
			return err
		}
		n.blockchainServer = blockchain.NewServer(n.Log, n.tmNode)
		// initialise the client
		client, err := n.tmNode.GetClient()
		if err != nil {
			return err
		}
		n.blockchainClient.Set(client, n.tmNode.MempoolSize)
	case blockchain.ProviderNullChain:
		// nullchain acts as both the client and the server because its does everything
		n.nullBlockchain = nullchain.NewClient(
			n.Log,
			n.conf.Blockchain.Null,
			n.protocol.GetTimeService(), // if we've loaded from a snapshot we need to be able to ask the protocol what time its at
		)
		n.nullBlockchain.SetABCIApp(n.abciApp)
		n.blockchainServer = blockchain.NewServer(n.Log, n.nullBlockchain)
		// n.blockchainClient = blockchain.NewClient(n.nullBlockchain)
		n.blockchainClient.Set(n.nullBlockchain, 100*1024*1024)

	default:
		return ErrUnknownChainProvider
	}

	n.confWatcher.OnConfigUpdate(
		func(cfg config.Config) { n.blockchainServer.ReloadConf(cfg.Blockchain) },
	)

	if err := n.blockchainServer.Start(); err != nil {
		return err
	}

	if err := n.blockchainClient.Start(); err != nil {
		return err
	}

	return nil
}

func (n *Command) setupCommon() (err error) {
	// this shouldn't happen, the context is initialized in here
	if n.cancel != nil {
		n.cancel()
	}

	// ensure we cancel the context on error
	defer func() {
		if err != nil {
			n.cancel()
		}
	}()

	// initialize the application context
	n.ctx, n.cancel = context.WithCancel(context.Background())

	// get the configuration, this have been loaded by the root
	conf := n.confWatcher.Get()

	// reload logger with the setup from configuration
	n.Log = logging.NewLoggerFromConfig(conf.Logging).Named(n.Log.GetName())

	// enable pprof if necessary
	if conf.Pprof.Enabled {
		n.Log.Info("vega is starting with pprof profile, this is not a recommended setting for production")
		n.pproffhandlr, err = pprof.New(n.Log, conf.Pprof)
		if err != nil {
			return err
		}
		n.confWatcher.OnConfigUpdate(
			func(cfg config.Config) { n.pproffhandlr.ReloadConf(cfg.Pprof) },
		)
	}

	n.stats = stats.New(n.Log, n.conf.Stats)

	// start prometheus stuff
	metrics.Start(n.conf.Metrics)

	return err
}

func (n *Command) loadNodeWallets() (err error) {
	// if we are a non-validator, nothing needs to be done here
	if !n.conf.IsValidator() {
		return nil
	}

	n.nodeWallets, err = nodewallets.GetNodeWallets(n.conf.NodeWallet, n.vegaPaths, n.nodeWalletPassphrase)
	if err != nil {
		return fmt.Errorf("couldn't get node wallets: %w", err)
	}

	return n.nodeWallets.Verify()
}

func (n *Command) startABCI(log *logging.Logger, app types.Application, tmHome string, network string, networkURL string) (*abci.TmNode, error) {
	var (
		genesisDoc *tmtypes.GenesisDoc
		err        error
	)
	if len(network) > 0 {
		genesisDoc, err = httpGenesisDocProvider(network)
	} else if len(networkURL) > 0 {
		genesisDoc, err = genesisDocHTTPFromURL(networkURL)
	}
	if err != nil {
		return nil, err
	}

	return abci.NewTmNode(
		n.conf.Blockchain,
		log,
		tmHome,
		app,
		genesisDoc,
	)
}

func (n *Command) startBlockchainClients() error {
	// just intantiate the client here, we'll setup the actual impl later on
	// when the null blockchain or tendermint is started.
	n.blockchainClient = blockchain.NewClient()

	// if we are a non-validator, nothing needs to be done here
	if !n.conf.IsValidator() {
		return nil
	}

	// We may not need ethereum client initialized when we have not
	// provided the ethereum endpoint. We skip creating client here
	// when RPCEnpoint is empty and the nullchain present.
	if len(n.conf.Ethereum.RPCEndpoint) < 1 && n.conf.Blockchain.ChainProvider == blockchain.ProviderNullChain {
		return nil
	}

	var err error
	n.l2Clients, err = ethclient.NewL2Clients(n.ctx, n.Log, n.conf.Ethereum)
	if err != nil {
		return fmt.Errorf("could not instantiate ethereum l2 clients: %w", err)
	}

	n.ethClient, err = ethclient.Dial(n.ctx, n.conf.Ethereum)
	if err != nil {
		return fmt.Errorf("could not instantiate ethereum client: %w", err)
	}
	n.ethConfirmations = ethclient.NewEthereumConfirmations(n.conf.Ethereum, n.ethClient, nil)

	return nil
}

// kill the running process by signaling itself with SIGKILL.
func kill() error {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return p.Signal(syscall.SIGKILL)
}
