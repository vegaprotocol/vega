package start

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"

	"code.vegaprotocol.io/vega/cmd/vegaone/flags"
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

	"github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"
	"google.golang.org/grpc"
)

const namedLogger = "core"

type Core struct {
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

	vegaPaths      paths.Paths
	tendermintHome string

	ethClient        *ethclient.Client
	ethConfirmations *ethclient.EthereumConfirmations

	abciApp  *appW
	protocol *protocol.Protocol

	// APIs
	grpcServer  *api.GRPC
	proxyServer *rest.ProxyServer
	adminServer *admin.Server
	coreService *coreapi.Service

	genesisDoc *tmtypes.GenesisDoc
	tmNode     *abci.TmNode

	errCh chan error
}

func newCore(
	log *logging.Logger,
	vegaPaths paths.Paths,
	tendermintHome string,
	networkURL, network string,
	passphraseFile string,
) (*Core, error) {
	log = log.Named(namedLogger)

	confWatcher, err := config.NewWatcher(context.Background(), log, vegaPaths)
	if err != nil {
		return nil, err
	}

	// only try to get the passphrase if the node is started
	// as a validator
	var pass string
	if confWatcher.Get().IsValidator() {
		pass, err = flags.Passphrase(passphraseFile).Get("nodewallet passphrase", true)
		if err != nil {
			return nil, err
		}
	}

	genesisDoc, err := pullGenesis(log, networkURL, network)
	if err != nil {
		return nil, err
	}

	c := &Core{
		Log:                  log,
		conf:                 confWatcher.Get(),
		confWatcher:          confWatcher,
		vegaPaths:            vegaPaths,
		tendermintHome:       tendermintHome,
		genesisDoc:           genesisDoc,
		nodeWalletPassphrase: pass,
	}

	if err := c.setupCommon(); err != nil {
		return nil, err
	}

	return c, nil
}

func pullGenesis(log *logging.Logger, networkURL, network string) (*tmtypes.GenesisDoc, error) {
	genesisURL := networkURL
	if len(genesisURL) <= 0 && len(network) > 0 {
		genesisURL = httpGenesisDocURLFromNetwork(network)
	}

	var (
		genesisDoc *tmtypes.GenesisDoc
		err        error
	)
	if len(genesisURL) > 0 {
		log.Info("retrieving genesis file from", logging.String("url", genesisURL))
		if genesisDoc, err = genesisDocHTTPFromURL(genesisURL); err != nil {
			return nil, err
		}
		log.Info("genesis file loaded successfully", logging.String("url", genesisURL))
	}

	return genesisDoc, nil
}

var ErrUnknownChainProvider = errors.New("unknown chain provider")

func (c *Core) Start() error {
	c.Log.Info("starting vega",
		logging.String("version", version.Get()),
		logging.String("commit-hash", version.GetCommitHash()),
	)

	if err := c.loadNodeWallets(); err != nil {
		return err
	}

	if err := c.startBlockchainClients(); err != nil {
		return err
	}

	// TODO(): later we will want to select what version of the protocol
	// to run, most likely via configuration, so we can use legacy or current
	var err error
	c.protocol, err = protocol.New(
		c.ctx, c.confWatcher, c.Log, c.cancel, c.stopBlockchain, c.nodeWallets, c.ethClient, c.ethConfirmations, c.blockchainClient, c.vegaPaths, c.stats)
	if err != nil {
		return err
	}

	if err := c.startAPIs(); err != nil {
		return err
	}

	// if a chain is being replayed tendermint does this during the initial handshake with the
	// app and does so synchronously. We to need to set this off in a goroutine so we can catch any
	// SIGTERM during that replay and shutdown properly
	c.errCh = make(chan error)
	go func() {
		defer func() {
			// if a consensus failure happens during replay tendermint panics
			// we need to catch it so we can call shutdown and then re-panic
			if r := recover(); r != nil {
				c.Stop()
				panic(r)
			}
		}()
		if err := c.startBlockchain(); err != nil {
			c.errCh <- err
		}
		// start the nullblockchain if we are in that mode, it *needs* to be after we've started the gRPC server
		// otherwise it'll start calling init-chain and all the way before we're ready.
		if c.conf.Blockchain.ChainProvider == blockchain.ProviderNullChain {
			if err := c.nullBlockchain.StartServer(); err != nil {
				c.errCh <- err
			}
		}
	}()

	// at this point all is good, and we should be started, we can
	// just wait for signals or whatever
	c.Log.Info("Vega startup complete",
		logging.String("node-mode", string(c.conf.NodeMode)))

	return nil
}

func (c *Core) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *Core) Err() <-chan error {
	return c.errCh
}

func (c *Core) stopBlockchain() error {
	if c.blockchainServer == nil {
		return nil
	}
	return c.blockchainServer.Stop()
}

func (c *Core) Stop() error {
	upStatus := c.protocol.GetProtocolUpgradeService().GetUpgradeStatus()

	// Blockchain server has been already stopped by the app during the upgrade.
	// Calling stop again would block forever.
	if c.blockchainServer != nil && !upStatus.ReadyToUpgrade {
		c.blockchainServer.Stop()
	}
	if c.protocol != nil {
		c.protocol.Stop()
	}
	if c.grpcServer != nil {
		c.grpcServer.Stop()
	}
	if c.proxyServer != nil {
		c.proxyServer.Stop()
	}
	if c.adminServer != nil {
		c.adminServer.Stop()
	}

	if c.conf.IsValidator() {
		if err := c.nodeWallets.Ethereum.Cleanup(); err != nil {
			c.Log.Error("couldn't clean up Ethereum node wallet", logging.Error(err))
		}
	}

	var err error
	if c.pproffhandlr != nil {
		err = c.pproffhandlr.Stop()
	}

	c.Log.Info("Vega shutdown complete",
		logging.String("version", version.Get()),
		logging.String("version-hash", version.GetCommitHash()))

	c.Log.Sync()
	c.cancel()

	// Blockchain server need to be killed as it is stuck in BeginBlock function.
	if upStatus.ReadyToUpgrade {
		return kill()
	}

	return err
}

func (c *Core) startAPIs() error {
	c.grpcServer = api.NewGRPC(
		c.Log,
		c.conf.API,
		c.stats,
		c.blockchainClient,
		c.protocol.GetEventForwarder(),
		c.protocol.GetTimeService(),
		c.protocol.GetEventService(),
		c.protocol.GetPoW(),
		c.protocol.GetSpamEngine(),
		c.protocol.GetPowEngine(),
	)

	c.coreService = coreapi.NewService(c.ctx, c.Log, c.conf.CoreAPI, c.protocol.GetBroker())
	c.grpcServer.RegisterService(func(server *grpc.Server) {
		apipb.RegisterCoreStateServiceServer(server, c.coreService)
	})

	// watch configs
	c.confWatcher.OnConfigUpdate(
		func(cfg config.Config) { c.grpcServer.ReloadConf(cfg.API) },
	)

	c.proxyServer = rest.NewProxyServer(c.Log, c.conf.API)

	if c.conf.IsValidator() {
		adminServer, err := admin.NewValidatorServer(c.Log, c.conf.Admin, c.vegaPaths, c.nodeWalletPassphrase, c.nodeWallets, c.protocol.GetProtocolUpgradeService())
		if err != nil {
			return err
		}
		c.adminServer = adminServer
	} else {
		adminServer, err := admin.NewNonValidatorServer(c.Log, c.conf.Admin, c.protocol.GetProtocolUpgradeService())
		if err != nil {
			return err
		}
		c.adminServer = adminServer
	}

	go c.grpcServer.Start()
	go c.proxyServer.Start()

	if c.adminServer != nil {
		go c.adminServer.Start()
	}

	return nil
}

func (c *Core) startBlockchain() error {
	// make sure any env variable is resolved
	tmHome := os.ExpandEnv(c.tendermintHome)
	c.abciApp = newAppW(c.protocol.Abci())

	switch c.conf.Blockchain.ChainProvider {
	case blockchain.ProviderTendermint:
		var err error
		// initialise the node
		c.tmNode, err = c.startABCI(c.Log, c.abciApp, tmHome)
		if err != nil {
			return err
		}
		c.blockchainServer = blockchain.NewServer(c.Log, c.tmNode)
		// initialise the client
		client, err := c.tmNode.GetClient()
		if err != nil {
			return err
		}
		// n.blockchainClient = blockchain.NewClient(client)
		c.blockchainClient.Set(client)
	case blockchain.ProviderNullChain:
		// nullchain acts as both the client and the server because its does everything
		c.nullBlockchain = nullchain.NewClient(
			c.Log,
			c.conf.Blockchain.Null,
			c.protocol.GetTimeService(), // if we've loaded from a snapshot we need to be able to ask the protocol what time its at
		)
		c.nullBlockchain.SetABCIApp(c.abciApp)
		c.blockchainServer = blockchain.NewServer(c.Log, c.nullBlockchain)
		// n.blockchainClient = blockchain.NewClient(n.nullBlockchain)
		c.blockchainClient.Set(c.nullBlockchain)

	default:
		return ErrUnknownChainProvider
	}

	c.confWatcher.OnConfigUpdate(
		func(cfg config.Config) { c.blockchainServer.ReloadConf(cfg.Blockchain) },
	)

	if err := c.blockchainServer.Start(); err != nil {
		return err
	}

	if err := c.blockchainClient.Start(); err != nil {
		return err
	}

	return nil
}

func (n *Core) setupCommon() (err error) {
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
	// n.Log = logging.NewLoggerFromConfig(conf.Logging).Named(n.Log.GetName())

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

func (n *Core) loadNodeWallets() (err error) {
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

func (c *Core) startABCI(
	log *logging.Logger,
	app types.Application,
	tmHome string,
) (*abci.TmNode, error) {
	return abci.NewTmNode(
		c.conf.Blockchain,
		log,
		tmHome,
		app,
		c.genesisDoc,
	)
}

func (c *Core) startBlockchainClients() error {
	// just intantiate the client here, we'll setup the actual impl later on
	// when the null blockchain or tendermint is started.
	c.blockchainClient = blockchain.NewClient()

	// if we are a non-validator, nothing needs to be done here
	if !c.conf.IsValidator() {
		return nil
	}

	if c.conf.Blockchain.ChainProvider != blockchain.ProviderNullChain {
		var err error
		c.ethClient, err = ethclient.Dial(c.ctx, c.conf.Ethereum)
		if err != nil {
			return fmt.Errorf("could not instantiate ethereum client: %w", err)
		}
		c.ethConfirmations = ethclient.NewEthereumConfirmations(c.conf.Ethereum, c.ethClient, nil)
	}

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
