package node2

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	apipb "code.vegaprotocol.io/protos/vega/api/v1"
	"code.vegaprotocol.io/shared/paths"
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
	"github.com/blang/semver"
	"google.golang.org/grpc"
)

var ErrUnknownChainProvider = errors.New("unknown chain provider")

type NodeCommand struct {
	ctx    context.Context
	cancel context.CancelFunc

	Log         *logging.Logger
	Version     string
	VersionHash string

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
	coreService *coreapi.Service

	statusChecker *monitoring.Status

	protocolUpgrade <-chan string
}

func (n *NodeCommand) Run(
	confWatcher *config.Watcher,
	vegaPaths paths.Paths,
	nodeWalletPassphrase string,
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
	n.statusChecker.OnChainVersionObtained(
		func(v string) { n.stats.SetChainVersion(v) },
	)

	// TODO(): later we will want to select what version of the protocol
	// to run, most likely via configuration, so we can use legacy or current
	var err error
	n.protocol, err = protocol.New(
		n.ctx, n.confWatcher, n.Log, n.cancel, n.nodeWallets, n.ethClient, n.ethConfirmations, n.blockchainClient, vegaPaths, n.stats)
	if err != nil {
		return err
	}

	if err := n.startBlockchain(); err != nil {
		return err
	}

	if err := n.startAPIs(); err != nil {
		return err
	}

	// at this point all is good, and we should be started, we can
	// just wait for signals or whatever
	n.Log.Info("Vega startup complete",
		logging.String("node-mode", string(n.conf.NodeMode)))

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

func (n *NodeCommand) Stop() {
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

	if n.conf.IsValidator() {
		if err := n.nodeWallets.Ethereum.Cleanup(); err != nil {
			n.Log.Error("couldn't clean up Ethereum node wallet", logging.Error(err))
		}
	}
}

// updateAPIsServices is to be called when a new protocol is being loaded
// so the API services bind to the proper engines / services.
func (n *NodeCommand) updateAPIsServices() {
	n.grpcServer.UpdateProtocolServices(
		n.protocol.GetEventForwarder(),
		n.protocol.GetTimeService(),
		n.protocol.GetEventService(),
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
	)

	n.grpcServer.RegisterService(func(server *grpc.Server) {
		svc := coreapi.NewService(n.ctx, n.Log, n.conf.CoreAPI, n.protocol.GetBroker())
		apipb.RegisterCoreStateServiceServer(server, svc)
	})

	// watch configs
	n.confWatcher.OnConfigUpdate(
		func(cfg config.Config) { n.grpcServer.ReloadConf(cfg.API) },
		func(cfg config.Config) { n.statusChecker.ReloadConf(cfg.Monitoring) },
	)

	n.proxyServer = rest.NewProxyServer(n.Log, n.conf.API)

	go n.grpcServer.Start()
	go n.proxyServer.Start()

	return nil
}

func (n *NodeCommand) startBlockchain() error {
	n.abciApp = newAppW(n.protocol.Abci())

	switch n.conf.Blockchain.ChainProvider {
	case blockchain.ProviderTendermint:
		n.blockchainServer = blockchain.NewServer(
			abci.NewServer(n.Log, n.conf.Blockchain, n.abciApp),
		)
	case blockchain.ProviderNullChain:
		n.nullBlockchain.SetABCIApp(n.abciApp)
		// nullchain acts as both the client and the server because its does everything
		n.blockchainServer = blockchain.NewServer(n.nullBlockchain)
	default:
		return ErrUnknownChainProvider
	}

	n.confWatcher.OnConfigUpdate(
		func(cfg config.Config) { n.blockchainServer.ReloadConf(cfg.Blockchain) },
	)

	if err := n.blockchainServer.Start(); err != nil {
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

	n.stats = stats.New(n.Log, n.conf.Stats, n.Version, n.VersionHash)

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

func (n *NodeCommand) startBlockchainClients(_ []string) error {
	var null *nullchain.NullBlockchain
	switch n.conf.Blockchain.ChainProvider {
	case blockchain.ProviderTendermint:
		a, err := abci.NewClient(n.conf.Blockchain.Tendermint.ClientAddr)
		if err != nil {
			return err
		}
		n.blockchainClient = blockchain.NewClient(a)
	case blockchain.ProviderNullChain:
		n.nullBlockchain = nullchain.NewClient(n.Log, n.conf.Blockchain.Null)
		n.blockchainClient = blockchain.NewClient(null)
	}

	// if we are a non-validator, nothing needs to be done here
	if !n.conf.IsValidator() {
		return nil
	}

	if n.conf.Blockchain.ChainProvider != blockchain.ProviderNullChain {
		var err error
		n.ethClient, err = ethclient.Dial(n.ctx, n.conf.NodeWallet.ETH.Address)
		if err != nil {
			return fmt.Errorf("could not instantiate ethereum client: %w", err)
		}
		n.ethConfirmations = ethclient.NewEthereumConfirmations(n.ethClient, nil)
	}

	return nil
}
