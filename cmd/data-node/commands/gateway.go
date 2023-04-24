// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/datanode/gateway/server"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"golang.org/x/sync/errgroup"

	"github.com/jessevdk/go-flags"
)

type gatewayCmd struct {
	gateway.Config
	config.VegaHomeFlag
}

func (opts *gatewayCmd) Execute(_ []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eg, ctx := errgroup.WithContext(ctx)

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
		gracefulStop := make(chan os.Signal, 1)
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
		Config: gateway.NewDefaultConfig(),
	}

	_, err := parser.AddCommand("gateway", "The API gateway", "The gateway for all the vega APIs", opts)
	return err
}
