package main

import (
	"log"
	"os"

	"github.com/jessevdk/go-flags"
)

// mainOptions act as global options.
type mainOptions struct {
}

// Subcommand is the signature of a sub command that can be registered.
type Subcommand func(*flags.Parser) error

// Register registers one or more subcommands.
func Register(parser *flags.Parser, cmds ...Subcommand) error {
	for _, fn := range cmds {
		if err := fn(parser); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	parser := flags.NewParser(&mainOptions{}, flags.Default)

	Register(parser,
		Gateway,
		// other sub-cmds goes here
	)

	if _, err := parser.Parse(); err != nil {
		log.Printf("err = %+v\n", err)
		switch t := err.(type) {
		case *flags.Error:
			if t.Type != flags.ErrHelp {
				parser.WriteHelp(os.Stdout)
			}
			os.Exit(-1)
		default:
			log.Printf("err = %+v\n", err)
		}
	}
}
