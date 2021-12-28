package nodewallet

import (
	"fmt"

	vgjson "code.vegaprotocol.io/shared/libs/json"
	"code.vegaprotocol.io/shared/paths"

	"code.vegaprotocol.io/vega/config"
	vgfmt "code.vegaprotocol.io/vega/libs/fmt"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"

	"github.com/jessevdk/go-flags"
)

type generateCmd struct {
	config.OutputFlag

	Config nodewallets.Config

	WalletPassphrase config.Passphrase `long:"wallet-passphrase-file"`

	Chain string `short:"c" long:"chain" required:"true" description:"The chain to be imported (vega, ethereum)"`
	Force bool   `long:"force" description:"Should the command generate a new wallet on top of an existing one"`
}

const (
	ethereumChain = "ethereum"
	vegaChain     = "vega"
)

func (opts *generateCmd) Execute(_ []string) error {
	output, err := opts.GetOutput()
	if err != nil {
		return err
	}

	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	registryPass, err := rootCmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(rootCmd.VegaHome)

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
		var walletPass string
		if opts.Config.ETH.ClefAddress == "" {
			walletPass, err = opts.WalletPassphrase.Get("blockchain wallet", true)
			if err != nil {
				return err
			}
		} else if output.IsHuman() {
			fmt.Println(yellow("Warning: Generating a new account in Clef has to be manually approved, and only the Key Store backend is supported. \nPlease consider using the 'import' command instead."))
		}

		data, err = nodewallets.GenerateEthereumWallet(
			opts.Config.ETH,
			vegaPaths,
			registryPass,
			walletPass,
			opts.Force,
		)
		if err != nil {
			return fmt.Errorf("couldn't generate Ethereum node wallet: %w", err)
		}
	case vegaChain:
		walletPass, err := opts.WalletPassphrase.Get("blockchain wallet", true)
		if err != nil {
			return err
		}

		data, err = nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletPass, opts.Force)
		if err != nil {
			return fmt.Errorf("couldn't generate Vega node wallet: %w", err)
		}
	default:
		return fmt.Errorf("chain %q is not supported", opts.Chain)
	}

	if output.IsHuman() {
		fmt.Println(green("generation successful:"))
		vgfmt.PrettyPrint(data)
	} else if output.IsJSON() {
		if err := vgjson.Print(data); err != nil {
			return err
		}
	}

	return nil
}
