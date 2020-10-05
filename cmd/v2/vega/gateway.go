package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/gateway"
	gql "code.vegaprotocol.io/vega/gateway/graphql"
	"code.vegaprotocol.io/vega/gateway/rest"
	"code.vegaprotocol.io/vega/logging"

	"github.com/jessevdk/go-flags"
)

type gatewayOptions struct {
	gateway.Config
	RootPathOption
}

func Gateway(parser *flags.Parser) error {
	opts := &gatewayOptions{
		Config:         gateway.NewDefaultConfig(),
		RootPathOption: NewRootPathOption(),
	}

	_, err := parser.AddCommand("gateway", "short", "long", opts)
	return err
}

func (opts *gatewayOptions) Execute(args []string) error {
	ctx := context.Background()

	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	cfgwatchr, err := config.NewFromFile(ctx, log, opts.RootPath, opts.RootPath)
	if err != nil {
		log.Error("unable to start config watcher", logging.Error(err))
		return errors.New("unable to start config watcher")
	}

	conf := cfgwatchr.Get()

	// parse the remaining command line options again to ensure they
	// take precedence.
	flags.Parse(&conf)

	if conf.Gateway.REST.Enabled {
		srv := rest.NewProxyServer(log, conf.Gateway)
		go func() { srv.Start() }()
		defer srv.Stop()
	}

	if conf.Gateway.GraphQL.Enabled {
		srv, err := gql.New(log, conf.Gateway)
		if err != nil {
			return err
		}
		go func() { srv.Start() }()
		defer srv.Stop()
	}

	waitSig(ctx, log)
	return nil
}

// waitSig will wait for a sigterm or sigint interrupt.
func waitSig(ctx context.Context, log *logging.Logger) {
	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
	case <-ctx.Done():
		// nothing to do
	}
}
