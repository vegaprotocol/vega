package node

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"code.vegaprotocol.io/vega/accounts"
	"code.vegaprotocol.io/vega/api"
	apiv2 "code.vegaprotocol.io/vega/api/v2"
	"code.vegaprotocol.io/vega/auth"
	"code.vegaprotocol.io/vega/basecmd"
	"code.vegaprotocol.io/vega/basecmd/gateway"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/candles"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/parties"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/plugins/positions"
	"code.vegaprotocol.io/vega/pprof"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/transfers"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/pkg/errors"
)

type AccountStore interface {
	buffer.AccountStore
	accounts.AccountStore
	Close() error
	ReloadConf(storage.Config)
}

type CandleStore interface {
	buffer.CandleStore
	candles.CandleStore
	Close() error
	ReloadConf(storage.Config)
}

type OrderStore interface {
	buffer.OrderStore
	orders.OrderStore
	GetMarketDepth(context.Context, string) (*proto.MarketDepth, error)
	Close() error
	ReloadConf(storage.Config)
}

type TradeStore interface {
	buffer.TradeStore
	trades.TradeStore
	Close() error
	ReloadConf(storage.Config)
}

// Node use to implement 'node' command.
type Node struct {
	ctx    context.Context
	cancel context.CancelFunc

	accounts              AccountStore
	candleStore           CandleStore
	orderStore            OrderStore
	marketStore           *storage.Market
	marketDataStore       *storage.MarketData
	tradeStore            TradeStore
	partyStore            *storage.Party
	riskStore             *storage.Risk
	transferResponseStore *storage.TransferResponse

	orderBuf        *buffer.Order
	tradeBuf        *buffer.Trade
	partyBuf        *buffer.Party
	transferBuf     *buffer.TransferResponse
	marketBuf       *buffer.Market
	accountBuf      *buffer.Account
	candleBuf       *buffer.Candle
	marketDataBuf   *buffer.MarketData
	marginLevelsBuf *buffer.MarginLevels
	settleBuf       *buffer.Settlement

	candleService    *candles.Svc
	tradeService     *trades.Svc
	marketService    *markets.Svc
	orderService     *orders.Svc
	partyService     *parties.Svc
	timeService      *vegatime.Svc
	auth             *auth.Svc
	accountsService  *accounts.Svc
	transfersService *transfers.Svc
	riskService      *risk.Svc

	blockchain       *blockchain.Blockchain
	blockchainClient *blockchain.Client

	pproffhandlr *pprof.Pprofhandler
	configPath   string
	conf         config.Config
	stats        *stats.Stats
	withPPROF    bool
	noChain      bool
	noStores     bool
	Log          *logging.Logger
	cfgwatchr    *config.Watcher

	executionEngine *execution.Engine
	mktscfg         []proto.Market

	// plugins
	settlePlugin *positions.Pos
	plugins      []plugins.Plugin
	srvv2        *apiv2.Server
	bufs         *buffer.Buffers
}

var (
	Command basecmd.Command

	configPath string
	noChain    bool
	noStores   bool
	withPprof  bool
)

func init() {
	Command.Name = "node"
	Command.Short = "Start a new vega node"

	cmd := flag.NewFlagSet("node", flag.ContinueOnError)
	cmd.StringVar(&configPath, "config-path", fsutil.DefaultVegaDir(), "file path to search for vega config file(s)")
	cmd.BoolVar(&noChain, "no-chain", false, "start the node using the noop chain")
	cmd.BoolVar(&noStores, "no-stores", false, "start the node without stores support")
	cmd.BoolVar(&withPprof, "with-pprof", false, "start the node with pprof support")

	cmd.Usage = func() {
		fmt.Fprintf(cmd.Output(), "%v\n\n", helpNode())
		cmd.PrintDefaults()
	}

	Command.FlagSet = cmd
	Command.Usage = Command.FlagSet.Usage
	Command.Run = runCommand
}

func helpNode() string {
	helpStr := `
Usage: vega node [options]
`
	return strings.TrimSpace(helpStr)
}

func runCommand(log *logging.Logger, args []string) int {
	if err := Command.FlagSet.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(Command.FlagSet.Output(), "%v\n", err)
		return 1
	}

	node := &Node{
		Log:        log,
		configPath: configPath,
	}
	err := node.persistentPre()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	err = node.preRun()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	// run the node
	err = node.runNode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	err = node.postRun()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	node.persistentPost()

	return 0
}

// runNode is the entry of node command.
func (l *Node) runNode() error {
	defer l.cancel()

	statusChecker := monitoring.New(l.Log, l.conf.Monitoring, l.blockchainClient)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { statusChecker.ReloadConf(cfg.Monitoring) })
	statusChecker.OnChainDisconnect(l.cancel)
	statusChecker.OnChainVersionObtained(func(v string) {
		l.stats.SetChainVersion(v)
	})

	var err error
	if l.conf.Auth.Enabled {
		l.auth, err = auth.New(l.ctx, l.Log, l.conf.Auth)
		if err != nil {
			return errors.Wrap(err, "unable to start auth service")
		}
		l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.auth.ReloadConf(cfg.Auth) })
	}

	// gRPC server
	grpcServer := api.NewGRPCServer(
		l.Log,
		l.conf.API,
		l.stats,
		l.blockchainClient,
		l.timeService,
		l.marketService,
		l.partyService,
		l.orderService,
		l.tradeService,
		l.candleService,
		l.accountsService,
		l.transfersService,
		l.riskService,
		statusChecker,
	)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { grpcServer.ReloadConf(cfg.API) })
	go grpcServer.Start()
	if l.conf.Auth.Enabled {
		l.auth.OnPartiesUpdated(grpcServer.OnPartiesUpdated)
	}
	metrics.Start(l.conf.Metrics)

	// start serverv2
	go func() {
		if err := l.srvv2.Start(); err != nil {
			l.Log.Error("error from api server",
				logging.Error(err))
		}
	}()

	// start gateway
	var gty *gateway.Gateway

	if l.conf.GatewayEnabled {
		gty, err = gateway.Start(l.Log, l.conf.Gateway)
		if err != nil {
			return err
		}
	}

	l.Log.Info("Vega startup complete")

	basecmd.WaitSig(l.ctx, l.Log)

	// Clean up and close resources
	grpcServer.Stop()
	l.blockchain.Stop()
	statusChecker.Stop()

	// cleanup gateway
	if l.conf.GatewayEnabled {
		if gty != nil {
			gty.Stop()
		}
	}

	return nil
}

func (n *Node) StartPlugins() {
	n.plugins = []plugins.Plugin{}
	for _, v := range n.conf.Plugins.Enabled {
		plugin, ok := plugins.Get(v)
		if !ok {
			n.Log.Error("tried to instanciated unknown plugin", logging.String("name", v))
		}
		p, err := plugin.New(n.Log, n.ctx, n.bufs, n.srvv2.GRPC(), n.conf.Plugins.Configs)
		if err != nil {
			n.Log.Error("unable to initialize plugin", logging.Error(err))
			continue
		}
		go p.Start()
		n.plugins = append(n.plugins, p)
	}
}
