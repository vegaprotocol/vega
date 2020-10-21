package main

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/gateway"
	"code.vegaprotocol.io/vega/gateway/server"
	"code.vegaprotocol.io/vega/logging"

	"github.com/jessevdk/go-flags"
)

type gatewayCmd struct {
	ctx context.Context
	gateway.Config
	config.RootPathFlag
}

func (opts *gatewayCmd) Execute(args []string) error {
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
	if err := srv.Start(); err != nil {
		return err
	}
	defer srv.Stop()

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
