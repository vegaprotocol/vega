// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
