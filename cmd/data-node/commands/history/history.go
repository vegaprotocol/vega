package history

import (
	"context"

	"github.com/jessevdk/go-flags"
)

type Cmd struct {
	// Subcommands
	Show showCmd `command:"show" description:"shows the block span of available history and the datanode's current history block span"`
	Load loadCmd `command:"load" description:"loads all the available history into the datanode"`
}

var historyCmd Cmd

func History(ctx context.Context, parser *flags.Parser) error {
	historyCmd = Cmd{
		Show: showCmd{},
		Load: loadCmd{},
	}

	desc := "Manage the datanode's history"
	_, err := parser.AddCommand("history", desc, desc, &historyCmd)
	if err != nil {
		return err
	}
	return nil
}
