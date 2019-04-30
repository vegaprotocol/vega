package main

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"code.vegaprotocol.io/vega/internal/fsutil"
	"code.vegaprotocol.io/vega/internal/gateway"
	gql "code.vegaprotocol.io/vega/internal/gateway/graphql"
	"code.vegaprotocol.io/vega/internal/gateway/rest"
	"code.vegaprotocol.io/vega/internal/logging"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

type gatewaySrv interface {
	Start()
	Stop()
}

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
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()

	configPath := g.rootPath
	if configPath == "" {
		// Use configPath from ENV
		configPath = envConfigPath()
		if configPath == "" {
			// Default directory ($HOME/.vega)
			configPath = fsutil.DefaultVegaDir()
		}
	}

	// load config
	buf, err := ioutil.ReadFile(filepath.Join(configPath, "gateway.toml"))
	if err != nil {
		return err
	}

	cfg := gateway.NewDefaultConfig()
	_, err = toml.Decode(string(buf), &cfg)
	if err != nil {
		return err
	}

	var restSrv, gqlSrv gatewaySrv

	if cfg.REST.Enabled {
		restSrv = rest.NewRestProxyServer(g.Log, cfg)
	}

	if cfg.GraphQL.Enabled {
		gqlSrv, err = gql.New(g.Log, cfg)
	}

	if restSrv != nil {
		go restSrv.Start()
	}
	if gqlSrv != nil {
		go gqlSrv.Start()
	}

	waitSig(ctx, g.Log)
	if restSrv != nil {
		restSrv.Stop()
	}
	if gqlSrv != nil {
		gqlSrv.Stop()
	}

	return nil
}
