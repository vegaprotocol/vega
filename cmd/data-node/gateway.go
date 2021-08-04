package main

import (
	"context"
	"errors"

	"code.vegaprotocol.io/data-node/config"
	"code.vegaprotocol.io/data-node/gateway"
	"code.vegaprotocol.io/data-node/gateway/server"
	"code.vegaprotocol.io/data-node/logging"

	"github.com/jessevdk/go-flags"
)

type gatewayCmd struct {
	ctx context.Context
	gateway.Config
	config.RootPathFlag
	Help bool `short:"h" long:"help" description:"Show this help message"`
}

func (opts *gatewayCmd) Execute(_ []string) error {
	if opts.Help {
		return &flags.Error{
			Type:    flags.ErrHelp,
			Message: "vega gateway subcommand help",
		}
	}

	ctx, cancel := context.WithCancel(opts.ctx)
	defer cancel()

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

	srv := server.New(opts.Config, log)
	if err := srv.Start(ctx); err != nil {
		return err
	}

	waitSig(ctx, log)
	return nil
}

func Gateway(ctx context.Context, parser *flags.Parser) error {
	opts := &gatewayCmd{
		ctx:          ctx,
		Config:       gateway.NewDefaultConfig(),
		RootPathFlag: config.NewRootPathFlag(),
	}

	_, err := parser.AddCommand("gateway", "short", "long", opts)
	return err
}
