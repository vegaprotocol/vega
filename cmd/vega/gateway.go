package main

import (
	"code.vegaprotocol.io/vega/internal/fsutil"
	"code.vegaprotocol.io/vega/internal/logging"
	"github.com/spf13/cobra"
)

type gatewayCommand struct {
	command

	rootPath string
	Log      *logging.Logger
}

func (g *gatewayCommand) Init(c *Cli) {
	g.cli = c
	g.cmd = &cobra.Command{
		Use:   "gateway",
		Short: "Start the vega gateway",
		Long:  "Start up the vega gateway to the node api (rest and graphql)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return g.runGateway(args)
		},
	}

	fs := g.cmd.Flags()
	fs.StringVarP(&g.rootPath, "c", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
}

func (g *gatewayCommand) runGateway(args []string) error {
	return nil
}
