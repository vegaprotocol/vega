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

package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/blockexplorer"
	"code.vegaprotocol.io/vega/blockexplorer/config"
	"code.vegaprotocol.io/vega/logging"

	"github.com/jessevdk/go-flags"
	"go.uber.org/zap"
)

type Start struct {
	config.VegaHomeFlag
	config.Config
}

func (opts *Start) Execute(_ []string) error {
	logger := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer logger.AtExit()

	cfg, err := loadConfig(logger, opts.VegaHome)
	if err != nil {
		return err
	}

	be := blockexplorer.NewFromConfig(*cfg)

	// Used to retrieve the error from the block explorer in the main thread.
	errCh := make(chan error, 1)
	defer close(errCh)

	// Use to shutdown the block explorer.
	beCtx, stopBlockExplorer := context.WithCancel(context.Background())

	blockExplorerStopped := make(chan any)
	go func() {
		if err := be.Run(beCtx); err != nil {
			errCh <- err
		}
		close(blockExplorerStopped)
	}()

	err = waitUntilInterruption(logger, errCh)

	stopBlockExplorer()
	<-blockExplorerStopped

	return err
}

func Run(_ context.Context, parser *flags.Parser) error {
	runCmd := Start{}

	short := "Start block explorer backend"
	long := "Start the various API grpc/rest APIs to query the tendermint postgres transaction index"

	_, err := parser.AddCommand("start", short, long, &runCmd)
	return err
}

// waitUntilInterruption will wait for a sigterm or sigint interrupt.
func waitUntilInterruption(logger *logging.Logger, errChan <-chan error) error {
	gracefulStop := make(chan os.Signal, 1)
	defer func() {
		signal.Stop(gracefulStop)
		close(gracefulStop)
	}()

	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	select {
	case sig := <-gracefulStop:
		logger.Info("OS signal received", zap.String("signal", fmt.Sprintf("%+v", sig)))
		return nil
	case err := <-errChan:
		logger.Error("Initiating shutdown due to an internal error reported by the block explorer", zap.Error(err))
		return err
	}
}
