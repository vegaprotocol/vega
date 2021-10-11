package nodewallet

import (
	"fmt"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"

	"github.com/jessevdk/go-flags"
)

type verifyCmd struct {
	Config nodewallets.Config
}

func (opts *verifyCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	registryPass, err := rootCmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.NewPaths(rootCmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	nw, err := nodewallets.GetNodeWallets(opts.Config, vegaPaths, registryPass)
	if err != nil {
		return fmt.Errorf("couldn't get node wallets: %w", err)
	}

	if err := nw.Verify(); err != nil {
		return err
	}

	fmt.Println(green("ok"))
	return nil
}
