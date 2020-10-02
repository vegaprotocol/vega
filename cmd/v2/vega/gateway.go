package main

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/gateway"
	"code.vegaprotocol.io/vega/logging"

	"github.com/jessevdk/go-flags"
)

type gatewayOptions struct {
	gateway.Config
	RootPath string `short:"c" long:"root-path" description:"Path of the root directory in which the configuration will be located" env:"VEGA_CONFIG"`
}

func (opts *gatewayOptions) Execute(args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	return nil
}

func Gateway(parser *flags.Parser) error {
	opts := &gatewayOptions{
		Config:   gateway.NewDefaultConfig(),
		RootPath: fsutil.DefaultVegaDir(),
	}

	// TODO: load config from path

	_, err := parser.AddCommand("gateway", "short", "long", opts)
	return err
}
