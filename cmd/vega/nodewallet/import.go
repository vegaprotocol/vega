package nodewallet

import (
	"fmt"

	vgjson "code.vegaprotocol.io/shared/libs/json"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config"
	vgfmt "code.vegaprotocol.io/vega/libs/fmt"
	"code.vegaprotocol.io/vega/logging"
	nodewallet "code.vegaprotocol.io/vega/nodewallets"

	"github.com/jessevdk/go-flags"
)

type importCmd struct {
	config.OutputFlag

	Config nodewallet.Config

	WalletPassphrase config.Passphrase `long:"wallet-passphrase-file"`

	Chain      string `short:"c" long:"chain" required:"true" description:"The chain to be imported (vega, ethereum)"`
	WalletPath string `long:"wallet-path" required:"true" description:"The path to the wallet file to import"`
	Force      bool   `long:"force" description:"Should the command re-write an existing nodewallet file if it exists"`
}

func (opts *importCmd) Execute(_ []string) error {
	output, err := opts.GetOutput()
	if err != nil {
		return err
	}

	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	registryPass, err := rootCmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return err
	}

	walletPass, err := opts.WalletPassphrase.Get("blockchain wallet")
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

	var data map[string]string
	switch opts.Chain {
	case ethereumChain:
		data, err = nodewallet.ImportEthereumWallet(vegaPaths, registryPass, walletPass, opts.WalletPath, opts.Force)
		if err != nil {
			return fmt.Errorf("couldn't import Ethereum node wallet: %w", err)
		}
	case vegaChain:
		data, err = nodewallet.ImportVegaWallet(vegaPaths, registryPass, walletPass, opts.WalletPath, opts.Force)
		if err != nil {
			return fmt.Errorf("couldn't import Vega node wallet: %w", err)
		}
	}

	if output.IsHuman() {
		fmt.Println("import successful:")
		vgfmt.PrettyPrint(data)
	} else if output.IsJSON() {
		if err := vgjson.Print(data); err != nil {
			return err
		}
	}

	return nil
}
