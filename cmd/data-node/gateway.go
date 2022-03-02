package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/data-node/config"
	"code.vegaprotocol.io/data-node/gateway"
	"code.vegaprotocol.io/data-node/gateway/server"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/shared/paths"
	"golang.org/x/sync/errgroup"

	"github.com/jessevdk/go-flags"
)

type gatewayCmd struct {
	ctx context.Context
	gateway.Config
	config.VegaHomeFlag
}

func (opts *gatewayCmd) Execute(_ []string) error {
	eg, ctx := errgroup.WithContext(opts.ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	vegaPaths := paths.New(opts.VegaHome)

	cfgwatchr, err := config.NewWatcher(ctx, log, vegaPaths)
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

	// waitSig will wait for a sigterm or sigint interrupt.
	eg.Go(func() error {
		var gracefulStop = make(chan os.Signal, 1)
		signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

		select {
		case sig := <-gracefulStop:
			log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
			cancel()
		case <-ctx.Done():
			return ctx.Err()
		}

		return nil
	})

	eg.Go(func() error {
		srv := server.New(opts.Config, log, vegaPaths)
		if err := srv.Start(ctx); err != nil {
			return err
		}

		return nil
	})

	return eg.Wait()
}

func Gateway(ctx context.Context, parser *flags.Parser) error {
	opts := &gatewayCmd{
		ctx:    ctx,
		Config: gateway.NewDefaultConfig(),
	}

	_, err := parser.AddCommand("gateway", "The API gateway", "The gateway for all the vega APIs", opts)
	return err
}
