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
	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/api/rest"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/checkpoint"
	ethclient "code.vegaprotocol.io/vega/client/eth"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/coreapi"
	"code.vegaprotocol.io/vega/delegation"
	"code.vegaprotocol.io/vega/epochtime"
	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/limits"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/notary"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/oracles/adaptors"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/rewards"
	"code.vegaprotocol.io/vega/snapshot"
	"code.vegaprotocol.io/vega/spam"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/statevar"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/validators/erc20multisig"
	"code.vegaprotocol.io/vega/vegatime"

	"google.golang.org/grpc"
)

// NodeCommand use to implement 'node' command.
type NodeCommand struct {
	ctx    context.Context
	cancel context.CancelFunc

	// blockchain
	// protocol
	// protocolUpgrade

	broker *broker.Broker

	timeService  *vegatime.Svc
	epochService *epochtime.Svc
	eventService *subscribers.Service

	blockchainServer *blockchain.Server
	blockchainClient *blockchain.Client

	pproffhandlr *pprof.Pprofhandler
	stats        *stats.Stats
	Log          *logging.Logger

	vegaPaths   paths.Paths
	conf        config.Config
	confWatcher *config.Watcher

	feesTracker          *execution.FeesTracker
	statevar             *statevar.Engine
	snapshot             *snapshot.Engine
	executionEngine      *execution.Engine
	governance           *governance.Engine
	collateral           *collateral.Engine
	oracle               *oracles.Engine
	oracleAdaptors       *adaptors.Adaptors
	netParams            *netparams.Store
	delegation           *delegation.Engine
	limits               *limits.Engine
	rewards              *rewards.Engine
	checkpoint           *checkpoint.Engine
	spam                 *spam.Engine
	nodeWallets          *nodewallets.NodeWallets
	nodeWalletPassphrase string
	builtinOracle        *oracles.Builtin

	assets               *assets.Service
	topology             *validators.Topology
	notary               *notary.SnapshotNotary
	eventForwarder       *evtforward.Forwarder
	eventForwarderEngine EventForwarderEngine
	witness              *validators.Witness
	banking              *banking.Engine
	genesisHandler       *genesis.Handler

	// plugins
	settlePlugin     *plugins.Positions
	notaryPlugin     *plugins.Notary
	assetPlugin      *plugins.Asset
	withdrawalPlugin *plugins.Withdrawal
	depositPlugin    *plugins.Deposit

	// staking
	ethClient             *ethclient.Client
	ethConfirmations      *ethclient.EthereumConfirmations
	stakingAccounts       *staking.Accounting
	stakeVerifier         *staking.StakeVerifier
	stakeCheckpoint       *staking.Checkpoint
	erc20MultiSigTopology *erc20multisig.Topology

	commander *nodewallets.Commander

	app *processor.App

	Version     string
	VersionHash string
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

	tmCfg := n.conf.Blockchain.Tendermint
	if tmCfg.ABCIRecordDir != "" && tmCfg.ABCIReplayFile != "" {
		return errors.New("you can't specify both abci-record and abci-replay flags")
	}

	stages := []func([]string) error{
		n.setupCommon,
		n.startBlockchainConnections,
		n.loadNodeWallets,
		n.startServices,
		n.runNode,
		n.stopServices,
		n.postRun,
		n.persistentPost,
	}
	for _, fn := range stages {
		if err := fn(args); err != nil {
			return err
		}
	}

	return nil
}

// runNode is the entry of node command.
func (n *NodeCommand) runNode(args []string) error {
	defer n.cancel()
	defer func() {
		if n.conf.IsValidator() {
			if err := n.nodeWallets.Ethereum.Cleanup(); err != nil {
				n.Log.Error("couldn't clean up Ethereum node wallet", logging.Error(err))
			}
		}
	}()

	statusChecker := monitoring.New(n.Log, n.conf.Monitoring, n.blockchainClient)
	statusChecker.OnChainDisconnect(n.cancel)
	statusChecker.OnChainVersionObtained(
		func(v string) { n.stats.SetChainVersion(v) },
	)

	grpcServer := api.NewGRPC(
		n.Log,
		n.conf.API,
		n.stats,
		n.blockchainClient,
		n.eventForwarder,
		n.timeService,
		n.eventService,
		statusChecker,
	)

	grpcServer.RegisterService(func(server *grpc.Server) {
		svc := coreapi.NewService(n.ctx, n.Log, n.conf.CoreAPI, n.broker)
		apipb.RegisterCoreStateServiceServer(server, svc)
	})

	// watch configs
	n.confWatcher.OnConfigUpdate(
		func(cfg config.Config) { grpcServer.ReloadConf(cfg.API) },
		func(cfg config.Config) { statusChecker.ReloadConf(cfg.Monitoring) },
	)

	proxyServer := rest.NewProxyServer(n.Log, n.conf.API)

	// start the grpc server
	go grpcServer.Start()
	go proxyServer.Start()
	metrics.Start(n.conf.Metrics)

	// some clients need to start after the rpc-server is up
	err := n.blockchainClient.Start()

	if err == nil {
		n.Log.Info("Vega startup complete",
			logging.String("node-mode", string(n.conf.NodeMode)))
		waitSig(n.ctx, n.Log)
	}

	// Clean up and close resources
	grpcServer.Stop()
	n.blockchainServer.Stop()
	statusChecker.Stop()
	proxyServer.Stop()

	return err
}

// waitSig will wait for a sigterm or sigint interrupt.
func waitSig(ctx context.Context, log *logging.Logger) {
	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
	case <-ctx.Done():
		// nothing to do
	}
}
