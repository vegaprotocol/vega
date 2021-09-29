package nodewallet

import (
	"context"

	"code.vegaprotocol.io/vega/config"
	nodewallet "code.vegaprotocol.io/vega/nodewallets"

	"github.com/fatih/color"
	"github.com/jessevdk/go-flags"
)

var (
	yellow = color.New(color.FgYellow).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
)

type RootCmd struct {
	// Global options
	config.VegaHomeFlag
	config.PassphraseFlag

	// Subcommands
	Show     showCmd     `command:"show" description:"List the wallets registers into the nodewallet"`
	Generate generateCmd `command:"generate" description:"Generate and register a wallet into the nodewallet"`
	Import   importCmd   `command:"import" description:"Import the configuration of a wallet required by the vega node"`
	Verify   verifyCmd   `command:"verify" description:"Verify the configuration imported in the nodewallet"`
}

var rootCmd RootCmd

func NodeWallet(ctx context.Context, parser *flags.Parser) error {
	rootCmd = RootCmd{
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
