package nodewallet

import (
	"fmt"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"

	"github.com/jessevdk/go-flags"
)

type verifyCmd struct {
	Config nodewallet.Config
}

func (opts *verifyCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	pass, err := rootCmd.PassphraseFile.Get("node wallet")
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

	nw, err := nodewallet.New(log, conf.NodeWallet, pass, nil, vegaPaths)
	if err != nil {
		return err
	}

	err = nw.Verify()
	if err != nil {
		return err
	}

	fmt.Println("ok")
	return nil
}
