package nodewallet

import (
	"fmt"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jessevdk/go-flags"
)

type verifyCmd struct {
	Config nodewallet.Config
	Help   bool `short:"h" long:"help" description:"Show this help message"`
}

func (opts *verifyCmd) Execute(_ []string) error {
	if opts.Help {
		return &flags.Error{
			Type:    flags.ErrHelp,
			Message: "vega nodewallet verify subcommand help",
		}
	}

	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	if ok, err := fsutil.PathExists(rootCmd.RootPath); !ok {
		return fmt.Errorf("invalid root directory path: %w", err)
	}

	pass, err := rootCmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return err
	}

	conf, err := config.Read(rootCmd.RootPath)
	if err != nil {
		return err
	}
	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	ethClient, err := ethclient.Dial(conf.NodeWallet.ETH.Address)
	if err != nil {
		return err
	}

	nw, err := nodewallet.New(log, conf.NodeWallet, pass, ethClient, rootCmd.RootPath)
	if err != nil {
		return err
	}

	err = nw.Verify()
	if err != nil {
		return err
	}

	fmt.Printf("ok\n")
	return nil
}
