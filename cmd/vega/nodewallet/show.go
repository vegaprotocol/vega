package nodewallet

import (
	vgjson "code.vegaprotocol.io/shared/libs/json"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"

	"github.com/jessevdk/go-flags"
)

type showCmd struct {
	Config nodewallets.Config
}

func (opts *showCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	registryPass, err := rootCmd.PassphraseFile.Get("node wallet")
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

	registryLoader, err := nodewallets.NewRegistryLoader(vegaPaths, registryPass)
	if err != nil {
		return err
	}

	registry, err := registryLoader.GetRegistry(registryPass)
	if err != nil {
		return err
	}

	if err = vgjson.PrettyPrint(registry); err != nil {
		return err
	}
	return nil
}
