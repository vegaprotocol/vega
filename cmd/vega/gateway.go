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

	gty, err := startGateway(g.Log, configPath)
	if err != nil {
		return err
	}

	waitSig(ctx, g.Log)
	gty.Stop()

	return nil
}

type Gateway struct {
	gqlSrv  gatewaySrv
	restSrv gatewaySrv
}

func startGateway(log *logging.Logger, configPath string) (*Gateway, error) {
	// load config
	buf, err := ioutil.ReadFile(filepath.Join(configPath, "gateway.toml"))
	if err != nil {
		return nil, err
	}

	cfg := gateway.NewDefaultConfig()
	_, err = toml.Decode(string(buf), &cfg)
	if err != nil {
		return nil, err
	}

	var restSrv, gqlSrv gatewaySrv

	if cfg.REST.Enabled {
		restSrv = rest.NewRestProxyServer(log, cfg)
	}

	if cfg.GraphQL.Enabled {
		gqlSrv, err = gql.New(log, cfg)
	}

	if restSrv != nil {
		go restSrv.Start()
	}
	if gqlSrv != nil {
		go gqlSrv.Start()
	}

	return &Gateway{
		gqlSrv:  gqlSrv,
		restSrv: restSrv,
	}, nil

}

func (g *Gateway) Stop() {
	if g.restSrv != nil {
		g.restSrv.Stop()
	}
	if g.gqlSrv != nil {
		g.gqlSrv.Stop()
	}
}
