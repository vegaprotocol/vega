package genesis

import (
	"context"

	"code.vegaprotocol.io/vega/config"
	"github.com/jessevdk/go-flags"
)

type RootCmd struct {
	// Global options
	config.RootPathFlag
	config.PassphraseFlag

	// Subcommands
	Generate generateCmd `command:"generate" description:"Generates the genesis file"`
	Update   updateCmd   `command:"update" description:"Update the genesis file with the app_state"`
	Help     bool        `short:"h" long:"help" description:"Show this help message"`
}

var rootCmd RootCmd

func Genesis(ctx context.Context, parser *flags.Parser) error {
	rootPath := config.NewRootPathFlag()
	rootCmd = RootCmd{
		RootPathFlag: rootPath,
		Generate: generateCmd{
			TmRoot: "$HOME/.tendermint",
		},
		Update: updateCmd{
			TmRoot: "$HOME/.tendermint",
		},
	}

	desc := "Manage the genesis file"
	_, err := parser.AddCommand("genesis", desc, desc, &rootCmd)
	return err
}
