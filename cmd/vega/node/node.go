// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	apipb "code.vegaprotocol.io/protos/vega/api/v1"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/admin"
	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/api/rest"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/blockchain/nullchain"
	ethclient "code.vegaprotocol.io/vega/client/eth"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/coreapi"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/protocol"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/version"

	"github.com/blang/semver"
	"github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"
	"google.golang.org/grpc"
)

var ErrUnknownChainProvider = errors.New("unknown chain provider")

type NodeCommand struct {
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

	abciApp  *appW
	protocol *protocol.Protocol

	// APIs
	grpcServer  *api.GRPC
	proxyServer *rest.ProxyServer
	adminServer *admin.Server
	coreService *coreapi.Service

	statusChecker *monitoring.Status

	protocolUpgrade <-chan string

	tmNode *abci.TmNode
}

func (n *NodeCommand) Run(
	confWatcher *config.Watcher,
	vegaPaths paths.Paths,
	nodeWalletPassphrase, tmHome, networkURL, network string,
	args []string,
) error {
	n.confWatcher = confWatcher
	n.nodeWalletPassphrase = nodeWalletPassphrase

	n.conf = confWatcher.Get()
	n.vegaPaths = vegaPaths

	if err := n.setupCommon(args); err != nil {
		return err
	}

	if err := n.loadNodeWallets(args); err != nil {
		return err
	}

	if err := n.startBlockchainClients(args); err != nil {
		return err
	}

	n.statusChecker = monitoring.New(n.Log, n.conf.Monitoring, n.blockchainClient)
	n.statusChecker.OnChainDisconnect(n.cancel)

	// TODO(): later we will want to select what version of the protocol
	// to run, most likely via configuration, so we can use legacy or current
	var err error
	n.protocol, err = protocol.New(
		n.ctx, n.confWatcher, n.Log, n.cancel, n.nodeWallets, n.ethClient, n.ethConfirmations, n.blockchainClient, vegaPaths, n.stats)
	if err != nil {
		return err
	}

	if err := n.startAPIs(); err != nil {
		return err
	}

	if err := n.startBlockchain(tmHome, network, networkURL); err != nil {
		return err
	}

	// at this point all is good, and we should be started, we can
	// just wait for signals or whatever
	n.Log.Info("Vega startup complete",
		logging.String("node-mode", string(n.conf.NodeMode)))

	// start the nullblockchain if we are in that mode, it *needs* to be after we've started the gRPC server
	// otherwise it'll start calling init-chain and all the way before we're ready.
	if n.conf.Blockchain.ChainProvider == blockchain.ProviderNullChain {
		n.nullBlockchain.StartServer()
	}

	// wait for possible protocol upgrade, or user exist
	n.wait()

	// cleanup
	n.Stop()

	return nil
}

func (n *NodeCommand) wait() {
	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case version := <-n.protocolUpgrade:
			n.startProtocolUpgrade(version)
		case sig := <-gracefulStop:
			n.Log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
			return
		case <-n.ctx.Done():
			// nothing to do
			return
		}
	}
}

func (n *NodeCommand) startProtocolUpgrade(version string) {
	semVersion, err := semver.Parse(version)
	if err != nil {
		n.Log.Error("invalid protocol version upgrade received, upgrade aborted",
			logging.String("version", version),
			logging.Error(err),
		)
		return
	}

	// first check if the request to upgrade match the version we know
	if !protocol.Version.EQ(semVersion) {
		n.Log.Error("unknown protocol version upgrade received, upgrade aborted",
			logging.String("version", version),
			logging.Error(err),
		)
		return
	}

	// TODO(): this is not final
	// then by instantiating the new version of the protocol
	// this is placeholder for now as not implemented.
	n.protocol, err = protocol.New(
		n.ctx, n.confWatcher, n.Log, n.cancel, n.nodeWallets, n.ethClient, n.ethConfirmations, n.blockchainClient, n.vegaPaths, n.stats)
	if err != nil {
		n.Log.Panic("protocol upgrade failure, could not instantiate the new version of the protocol", logging.Error(err))
	}

	n.abciApp.ScheduleUpgrade(n.protocol.Abci())

	// now we can update all the services used from the protocol
	n.updateAPIsServices()
}

func (n *NodeCommand) Stop() error {
	if n.protocol != nil {
		n.protocol.Stop()
	}
	if n.grpcServer != nil {
		n.grpcServer.Stop()
	}
	if n.blockchainServer != nil {
		n.blockchainServer.Stop()
	}
	if n.statusChecker != nil {
		n.statusChecker.Stop()
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

	return err
}

// updateAPIsServices is to be called when a new protocol is being loaded
// so the API services bind to the proper engines / services.
func (n *NodeCommand) updateAPIsServices() {
	n.grpcServer.UpdateProtocolServices(
		n.protocol.GetEventForwarder(),
		n.protocol.GetTimeService(),
		n.protocol.GetEventService(),
		n.protocol.GetPoW(),
	)

	n.coreService.UpdateBroker(n.protocol.GetBroker())
}

func (n *NodeCommand) startAPIs() error {
	n.grpcServer = api.NewGRPC(
		n.Log,
		n.conf.API,
		n.stats,
		n.blockchainClient,
		n.protocol.GetEventForwarder(),
		n.protocol.GetTimeService(),
		n.protocol.GetEventService(),
		n.statusChecker,
		n.protocol.GetPoW(),
	)

	n.coreService = coreapi.NewService(n.ctx, n.Log, n.conf.CoreAPI, n.protocol.GetBroker())
	n.grpcServer.RegisterService(func(server *grpc.Server) {
		apipb.RegisterCoreStateServiceServer(server, n.coreService)
	})

	// watch configs
	n.confWatcher.OnConfigUpdate(
		func(cfg config.Config) { n.grpcServer.ReloadConf(cfg.API) },
		func(cfg config.Config) { n.statusChecker.ReloadConf(cfg.Monitoring) },
	)

	n.proxyServer = rest.NewProxyServer(n.Log, n.conf.API)

	if bool(n.conf.Admin.Server.Enabled) && n.conf.IsValidator() {
		adminServer, err := admin.NewServer(n.Log, n.conf.Admin, n.vegaPaths, n.nodeWalletPassphrase, n.nodeWallets)
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

func (n *NodeCommand) startBlockchain(tmHome, network, networkURL string) error {
	// make sure any env variable is resolved
	tmHome = os.ExpandEnv(tmHome)
	n.abciApp = newAppW(n.protocol.Abci())

	switch n.conf.Blockchain.ChainProvider {
	case blockchain.ProviderTendermint:
		var err error
		// initialise the node
		n.tmNode, err = n.startABCI(n.ctx, n.abciApp, tmHome, network, networkURL)
		if err != nil {
			return err
		}
		n.blockchainServer = blockchain.NewServer(n.tmNode)
		// initialise the client
		client, err := n.tmNode.GetClient()
		if err != nil {
			return err
		}
		// n.blockchainClient = blockchain.NewClient(client)
		n.blockchainClient.Set(client)
	case blockchain.ProviderNullChain:
		// nullchain acts as both the client and the server because its does everything
		n.nullBlockchain = nullchain.NewClient(n.Log, n.conf.Blockchain.Null)
		n.nullBlockchain.SetABCIApp(n.abciApp)
		n.blockchainServer = blockchain.NewServer(n.nullBlockchain)
		// n.blockchainClient = blockchain.NewClient(n.nullBlockchain)
		n.blockchainClient.Set(n.nullBlockchain)

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

func (n *NodeCommand) setupCommon(_ []string) (err error) {
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

	// initialize the application contet
	n.ctx, n.cancel = context.WithCancel(context.Background())

	// get the configuration, this have been loaded by the root
	conf := n.confWatcher.Get()

	// reload logger with the setup from configuration
	n.Log = logging.NewLoggerFromConfig(conf.Logging)

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

func (n *NodeCommand) loadNodeWallets(_ []string) (err error) {
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

func (n *NodeCommand) startABCI(
	ctx context.Context,
	app types.Application,
	tmHome string,
	network string,
	networkURL string,
) (*abci.TmNode, error) {
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
		n.Log,
		tmHome,
		app,
		genesisDoc,
	)
}

func (n *NodeCommand) startBlockchainClients(_ []string) error {
	// just intantiate the client here, we'll setup the actual impl later on
	// when the null blockchain or tendermint is started.
	n.blockchainClient = blockchain.NewClient()

	// if we are a non-validator, nothing needs to be done here
	if !n.conf.IsValidator() {
		return nil
	}

	if n.conf.Blockchain.ChainProvider != blockchain.ProviderNullChain {
		var err error
		n.ethClient, err = ethclient.Dial(n.ctx, n.conf.Ethereum)
		if err != nil {
			return fmt.Errorf("could not instantiate ethereum client: %w", err)
		}
		n.ethConfirmations = ethclient.NewEthereumConfirmations(n.ethClient, nil)
	}

	return nil
}
