package gateway

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"code.vegaprotocol.io/vega/basecmd"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/gateway"
	gql "code.vegaprotocol.io/vega/gateway/graphql"
	"code.vegaprotocol.io/vega/gateway/rest"
	"code.vegaprotocol.io/vega/logging"
)

var (
	Command basecmd.Command

	configPath string
)

type gatewaySrv interface {
	Start()
	Stop()
}

func init() {
	Command.Name = "gateway"
	Command.Short = "Start a new vega gateway"

	cmd := flag.NewFlagSet("gateway", flag.ContinueOnError)
	cmd.StringVar(&configPath, "config-path", fsutil.DefaultVegaDir(), "file path in which the configuration will be located")

	cmd.Usage = func() {
		fmt.Fprintf(cmd.Output(), "%v\n\n", helpGateway())
		cmd.PrintDefaults()
	}

	Command.FlagSet = cmd
	Command.Usage = Command.FlagSet.Usage
	Command.Run = runCommand
}

func helpGateway() string {
	helpStr := `
Usage: vega gateway [options]
`
	return strings.TrimSpace(helpStr)
}

func runCommand(log *logging.Logger, args []string) int {
	if err := Command.FlagSet.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(Command.FlagSet.Output(), "%v\n", err)
		return 1
	}

	if len(configPath) <= 0 {
		fmt.Fprintln(os.Stderr, "vega: config path cannot be empty")
		return 1
	}

	if err := runGateway(log, configPath); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	return 0
}

func runGateway(log *logging.Logger, configPath string) error {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()

	if configPath == "" {
		// Use configPath from ENV
		configPath = basecmd.EnvConfigPath()
		if configPath == "" {
			// Default directory ($HOME/.vega)
			configPath = fsutil.DefaultVegaDir()
		}
	}

	// VEGA config (holds all package level configs)
	cfgwatchr, err := config.NewFromFile(ctx, log, configPath, configPath)
	if err != nil {
		log.Error("unable to start config watcher", logging.Error(err))
		return errors.New("unable to start config watcher")
	}
	conf := cfgwatchr.Get()

	gty, err := Start(log, conf.Gateway)
	if err != nil {
		return err
	}

	basecmd.WaitSig(ctx, log)
	gty.Stop()

	return nil
}

// Gateway contains all the gateway objects, currently GraphQL and REST.
type Gateway struct {
	gqlSrv  gatewaySrv
	restSrv gatewaySrv
}

func Start(log *logging.Logger, cfg gateway.Config) (*Gateway, error) {
	var (
		restSrv, gqlSrv gatewaySrv
		err             error
	)
	if cfg.REST.Enabled {
		restSrv = rest.NewProxyServer(log, cfg)
	}

	if cfg.GraphQL.Enabled {
		gqlSrv, err = gql.New(log, cfg)
		if err != nil {
			return nil, err
		}
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
