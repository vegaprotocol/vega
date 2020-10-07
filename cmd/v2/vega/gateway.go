package main

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/gateway"
	gql "code.vegaprotocol.io/vega/gateway/graphql"
	"code.vegaprotocol.io/vega/gateway/rest"
	"code.vegaprotocol.io/vega/logging"

	"github.com/jessevdk/go-flags"
)

type gatewayOptions struct {
	ctx context.Context
	gateway.Config
	RootPathOption
}

func Gateway(ctx context.Context, parser *flags.Parser) error {
	opts := &gatewayOptions{
		ctx:            ctx,
		Config:         gateway.NewDefaultConfig(),
		RootPathOption: NewRootPathOption(),
	}

	_, err := parser.AddCommand("gateway", "short", "long", opts)
	return err
}

func (opts *gatewayOptions) Execute(args []string) error {
	ctx := opts.ctx

	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	cfgwatchr, err := config.NewFromFile(ctx, log, opts.RootPath, opts.RootPath)
	if err != nil {
		log.Error("unable to start config watcher", logging.Error(err))
		return errors.New("unable to start config watcher")
	}

	conf := cfgwatchr.Get()
	opts.Config = conf.Gateway

	// parse the remaining command line options again to ensure they
	// take precedence.
	if _, err := flags.Parse(opts); err != nil {
		return err
	}

	if conf.Gateway.REST.Enabled {
		srv := rest.NewProxyServer(log, opts.Config)
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
