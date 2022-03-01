package node

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/blockchain"
	ethclient "code.vegaprotocol.io/vega/client/eth"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
)

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

	// Set ulimits
	if err = n.SetUlimits(); err != nil {
		n.Log.Warn("Unable to set ulimits", logging.Error(err))
	} else {
		n.Log.Debug("Set ulimits", logging.Uint64("nofile", n.conf.UlimitNOFile))
	}

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

func (n *NodeCommand) startBlockchainConnections(_ []string) error {
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
