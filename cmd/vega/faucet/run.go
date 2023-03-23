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

package faucet

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/faucet"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
)

type faucetRun struct {
	ctx context.Context

	config.VegaHomeFlag
	config.PassphraseFlag
	faucet.Config
}

func (opts *faucetRun) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	pass, err := opts.PassphraseFile.Get("faucet wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(opts.VegaHome)

	faucetCfgLoader, err := faucet.InitialiseConfigLoader(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise faucet configuration loader: %w", err)
	}

	faucetCfg, err := faucetCfgLoader.GetConfig()
	if err != nil {
		return fmt.Errorf("couldn't get faucet configuration: %w", err)
	}

	if _, err := flags.NewParser(faucetCfg, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	faucetSvc, err := faucet.NewService(log, vegaPaths, *faucetCfg, pass)
	if err != nil {
		return fmt.Errorf("could not initialise the faucet service: %w", err)
	}

	ctx, cancel := context.WithCancel(opts.ctx)
	go func() {
		defer cancel()
		if err := faucetSvc.Start(); err != nil {
			log.Error("error starting faucet server", logging.Error(err))
		}
	}()

	waitSig(ctx, log)

	if err := faucetSvc.Stop(); err != nil {
		log.Error("error stopping faucet server", logging.Error(err))
	} else {
		log.Info("faucet server stopped with success")
	}

	return nil
}

// waitSig will wait for a sigterm or sigint interrupt.
func waitSig(ctx context.Context, log *logging.Logger) {
	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
	case <-ctx.Done():
		// nothing to do
	}
}
