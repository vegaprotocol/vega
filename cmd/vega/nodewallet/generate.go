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

type generateCmd struct {
	Config nodewallet.Config

	WalletPassphrase config.Passphrase `short:"w" long:"wallet-passphrase"`

	Chain      string `short:"c" long:"chain" required:"true" description:"The chain to be imported (vega, ethereum)"`
	Force      bool   `long:"force" description:"Should the command generate a new wallet on top of an existing one"`
	Help       bool   `short:"h" long:"help" description:"Show this help message"`
}

func (opts *generateCmd) Execute(_ []string) error {
	if opts.Help {
		return &flags.Error{
			Type:    flags.ErrHelp,
			Message: "vega nodewallet generate subcommand help",
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

	walletPass, err := opts.WalletPassphrase.Get("blockchain wallet")
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

	ethClient, err := ethclient.Dial(opts.Config.ETH.Address)
	if err != nil {
		return err
	}

	nw, err := nodewallet.New(log, conf.NodeWallet, pass, ethClient, rootCmd.RootPath)
	if err != nil {
		return err
	}

	_, ok := nw.Get(nodewallet.Blockchain(opts.Chain))
	if ok && opts.Force {
		log.Warn("a wallet is already present for the current chain, this action will create a new one", logging.String("chain", opts.Chain))
	} else if ok {
		return fmt.Errorf("a wallet is already imported for the chain %v, please rerun with option --force to overwrite it", opts.Chain)
	}

	err = nw.Generate(opts.Chain, pass, walletPass)
	if err != nil {
		return err
	}

	fmt.Println("generation successful")
	return nil
}
