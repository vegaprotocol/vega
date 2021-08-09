package nodewallet

import (
	"context"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/nodewallet"
	"github.com/jessevdk/go-flags"
)

type RootCmd struct {
	// Global options
	config.RootPathFlag
	config.PassphraseFlag

	// Subcommands
	Generate generateCmd `command:"generate" description:"Generate and register a wallet into the nodewallet"`
	Import   importCmd   `command:"import" description:"Import the configuration of a wallet required by the vega node"`
	Verify   verifyCmd   `command:"verify" description:"Verify the configuration imported in the nodewallet"`
	Help     bool        `short:"h" long:"help" description:"Show this help message"`
}

var rootCmd RootCmd

func NodeWallet(ctx context.Context, parser *flags.Parser) error {
	root := config.NewRootPathFlag()
	rootCmd = RootCmd{
		RootPathFlag: root,
		Generate: generateCmd{
			Config: nodewallet.NewDefaultConfig(),
		},
		Import: importCmd{
			Config: nodewallet.NewDefaultConfig(),
		},
		Verify: verifyCmd{
			Config: nodewallet.NewDefaultConfig(),
		},
	}

	var (
		short = "Manages the node wallet"
		long  = `The nodewallet is a wallet owned by the vega node, it contains all
	the information to login to other wallets from external blockchain that
	vega will need to run properly (e.g and ethereum wallet, which allow vega
	to sign transaction to be verified on the ethereum blockchain) available
	wallet: eth, vega`
	)

	_, err := parser.AddCommand("nodewallet", short, long, &rootCmd)
	return err
}
