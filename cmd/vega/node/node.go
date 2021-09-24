package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	coreapipb "code.vegaprotocol.io/protos/vega/coreapi/v1"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/api/rest"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"
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
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/notary"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/oracles/adaptors"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/rewards"
	"code.vegaprotocol.io/vega/spam"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/vegatime"

	"google.golang.org/grpc"
)

// NodeCommand use to implement 'node' command.
type NodeCommand struct {
	ctx    context.Context
	cancel context.CancelFunc

	broker *broker.Broker

	timeService  *vegatime.Svc
	epochService *epochtime.Svc
	eventService *subscribers.Service

	abciServer       *abci.Server
	blockchainClient *blockchain.Client

	pproffhandlr *pprof.Pprofhandler
	stats        *stats.Stats
	Log          *logging.Logger

	vegaPaths   paths.Paths
	conf        config.Config
	confWatcher *config.Watcher

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
	nodeWallet           *nodewallet.Service
	nodeWalletPassphrase string

	assets         *assets.Service
	topology       *validators.Topology
	notary         *notary.Notary
	evtfwd         *evtforward.EvtForwarder
	witness        *validators.Witness
	banking        *banking.Engine
	genesisHandler *genesis.Handler

	// plugins
	settlePlugin     *plugins.Positions
	notaryPlugin     *plugins.Notary
	assetPlugin      *plugins.Asset
	withdrawalPlugin *plugins.Withdrawal
	depositPlugin    *plugins.Deposit

	// staking
	ethClient       *ethclient.Client
	stakingAccounts *staking.Accounting
	stakeVerifier   *staking.StakeVerifier

	app *processor.App

	Version     string
	VersionHash string
}

func (l *NodeCommand) Run(confWatcher *config.Watcher, vegaPaths paths.Paths, nodeWalletPassphrase string, args []string) error {
	l.confWatcher = confWatcher
	l.nodeWalletPassphrase = nodeWalletPassphrase

	l.conf = confWatcher.Get()
	l.vegaPaths = vegaPaths

	tmCfg := l.conf.Blockchain.Tendermint
	if tmCfg.ABCIRecordDir != "" && tmCfg.ABCIReplayFile != "" {
		return errors.New("you can't specify both abci-record and abci-replay flags")
	}

	stages := []func([]string) error{
		l.persistentPre,
		l.preRun,
		l.runNode,
		l.postRun,
		l.persistentPost,
	}
	for _, fn := range stages {
		if err := fn(args); err != nil {
			return err
		}
	}

	return nil
}

// runNode is the entry of node command.
func (l *NodeCommand) runNode(args []string) error {
	defer l.cancel()
	defer func() {
		if err := l.nodeWallet.Cleanup(); err != nil {
			l.Log.Error("error cleaning up nodewallet", logging.Error(err))
		}
	}()

	statusChecker := monitoring.New(l.Log, l.conf.Monitoring, l.blockchainClient)
	statusChecker.OnChainDisconnect(l.cancel)
	statusChecker.OnChainVersionObtained(
		func(v string) { l.stats.SetChainVersion(v) },
	)

	grpcServer := api.NewGRPC(
		l.Log,
		l.conf.API,
		l.stats,
		l.blockchainClient,
		l.evtfwd,
		l.timeService,
		l.eventService,
		statusChecker,
	)

	grpcServer.RegisterService(func(server *grpc.Server) {
		svc := coreapi.NewService(l.ctx, l.Log, l.conf.CoreAPI, l.broker)
		coreapipb.RegisterCoreApiServiceServer(server, svc)
	})

	// watch configs
	l.confWatcher.OnConfigUpdate(
		func(cfg config.Config) { grpcServer.ReloadConf(cfg.API) },
		func(cfg config.Config) { statusChecker.ReloadConf(cfg.Monitoring) },
	)

	proxyServer := rest.NewProxyServer(l.Log, l.conf.API)

	// start the grpc server
	go grpcServer.Start()
	go proxyServer.Start()
	metrics.Start(l.conf.Metrics)

	l.Log.Info("Vega startup complete")
	waitSig(l.ctx, l.Log)

	// Clean up and close resources
	grpcServer.Stop()
	l.abciServer.Stop()
	statusChecker.Stop()
	proxyServer.Stop()

	return nil
}

// waitSig will wait for a sigterm or sigint interrupt.
func waitSig(ctx context.Context, log *logging.Logger) {
	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
	case <-ctx.Done():
		// nothing to do
	}
}

func flagProvided(flag string) bool {
	for _, v := range os.Args[1:] {
		if v == flag {
			return true
		}
	}

	return false
}
