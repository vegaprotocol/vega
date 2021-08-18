package genesis

import (
	"context"

	"github.com/jessevdk/go-flags"
)

type newCmd struct {
	Validator newValidatorCmd `command:"validator" description:"Show information to become validator"`
	Help      bool            `short:"h" long:"help" description:"Show this help message"`
}

func initNewCmd(ctx context.Context, parentCmd *flags.Command) error {
	cmd := newCmd{
		Validator: newValidatorCmd{
			TmRoot: "$HOME/.tendermint",
		},
	}

	var (
		short = "Create a resource"
		long  = "Create a resource"
	)

	_, err := parentCmd.AddCommand("new", short, long, &cmd)
	return err
}
