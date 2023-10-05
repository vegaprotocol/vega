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
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jessevdk/go-flags"
	"go.uber.org/zap"

	"code.vegaprotocol.io/vega/blockexplorer"
	"code.vegaprotocol.io/vega/blockexplorer/config"
	"code.vegaprotocol.io/vega/logging"
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
