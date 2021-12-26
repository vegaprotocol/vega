package node

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/blockchain/recorder"
	ethclient "code.vegaprotocol.io/vega/client/eth"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/prometheus/common/log"
	"github.com/spf13/afero"
	tmtypes "github.com/tendermint/tendermint/abci/types"
)

var (
	ErrUnknownChainProvider    = errors.New("unknown chain provider")
	ErrERC20AssetWithNullChain = errors.New("cannot use ERC20 asset with nullchain")
)

func (l *NodeCommand) persistentPre(_ []string) (err error) {
	// this shouldn't happen...
	if l.cancel != nil {
		l.cancel()
	}
	// ensure we cancel the context on error
	defer func() {
		if err != nil {
			l.cancel()
		}
	}()
	l.ctx, l.cancel = context.WithCancel(context.Background())

	conf := l.confWatcher.Get()

	// reload logger with the setup from configuration
	l.Log = logging.NewLoggerFromConfig(conf.Logging)

	if conf.Pprof.Enabled {
		l.Log.Info("vega is starting with pprof profile, this is not a recommended setting for production")
		l.pproffhandlr, err = pprof.New(l.Log, conf.Pprof)
		if err != nil {
			return
		}
		l.confWatcher.OnConfigUpdate(
			func(cfg config.Config) { l.pproffhandlr.ReloadConf(cfg.Pprof) },
		)
	}

	l.Log.Info("Starting Vega",
		logging.String("version", l.Version),
		logging.String("version-hash", l.VersionHash))

	// this doesn't fail
	l.timeService = vegatime.New(l.conf.Time)

	// Set ulimits
	if err = l.SetUlimits(); err != nil {
		l.Log.Warn("Unable to set ulimits",
			logging.Error(err))
	} else {
		l.Log.Debug("Set ulimits",
			logging.Uint64("nofile", l.conf.UlimitNOFile))
	}

	// this doesn't fail
	l.stats = stats.New(l.Log, l.conf.Stats, l.Version, l.VersionHash)

	if conf.Blockchain.ChainProvider != blockchain.ProviderNullChain {
		l.ethClient, err = ethclient.Dial(l.ctx, l.conf.NodeWallet.ETH.Address)
		if err != nil {
			return fmt.Errorf("could not instantiate ethereum client: %w", err)
		}
	}

	l.nodeWallets, err = nodewallets.GetNodeWallets(l.conf.NodeWallet, l.vegaPaths, l.nodeWalletPassphrase)
	if err != nil {
		return fmt.Errorf("couldn't get node wallets: %w", err)
	}

	return l.nodeWallets.Verify()
}

func (l *NodeCommand) startABCI(ctx context.Context, app *processor.App) (*abci.Server, error) {
	var abciApp tmtypes.Application
	tmCfg := l.conf.Blockchain.Tendermint
	if path := tmCfg.ABCIRecordDir; path != "" {
		rec, err := recorder.NewRecord(path, afero.NewOsFs())
		if err != nil {
			return nil, err
		}

		// closer
		go func() {
			<-ctx.Done()
			rec.Stop()
		}()

		abciApp = recorder.NewApp(app.Abci(), rec)
	} else {
		abciApp = app.Abci()
	}

	srv := abci.NewServer(l.Log, l.conf.Blockchain, abciApp)
	if err := srv.Start(); err != nil {
		return nil, err
	}

	if path := tmCfg.ABCIReplayFile; path != "" {
		rec, err := recorder.NewReplay(path, afero.NewOsFs())
		if err != nil {
			return nil, err
		}

		// closer
		go func() {
			<-ctx.Done()
			rec.Stop()
		}()

		go func() {
			if err := rec.Replay(abciApp); err != nil {
				log.Fatalf("replay: %v", err)
			}
		}()
	}

	return srv, nil
}
