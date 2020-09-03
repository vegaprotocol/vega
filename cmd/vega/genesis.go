package main

import (
	"fmt"

	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"

	"github.com/spf13/cobra"
)

type genesisCommand struct {
	command

	log     *logging.Logger
	inPlace string
}

func (g *genesisCommand) Init(c *Cli) {
	g.cli = c
	g.cmd = &cobra.Command{
		Use:   "genesis",
		Short: "The genesis subcommand",
		Long:  "Generate a default genesis state for a vega network",
		RunE:  g.Run,
	}

	g.cmd.Flags().StringVarP(&g.inPlace, "in-place", "i", "", "The path to the tendermint configuration, will re-write it with the vega genesis state")
}

func (g *genesisCommand) Run(cmd *cobra.Command, args []string) error {
	if len(g.inPlace) <= 0 {
		dump, err := genesis.DumpDefault()
		if err != nil {
			return err
		}
		fmt.Printf("%v\n", dump)
		return nil
	}
	return genesis.UpdateInPlaceDefault(g.inPlace)
}
