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

package visor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/visor/config"
	"code.vegaprotocol.io/vega/visor/utils"

	"golang.org/x/sync/errgroup"
)

const snapshotBlockHeightFlagName = "--snapshot.load-from-block-height"

type BinariesRunner struct {
	mut         sync.RWMutex
	running     map[int]*exec.Cmd
	binsFolder  string
	log         *logging.Logger
	stopDelay   time.Duration
	stopTimeout time.Duration
	releaseInfo *types.ReleaseInfo
}

func NewBinariesRunner(log *logging.Logger, binsFolder string, stopDelay, stopTimeout time.Duration, rInfo *types.ReleaseInfo) *BinariesRunner {
	return &BinariesRunner{
		binsFolder:  binsFolder,
		running:     map[int]*exec.Cmd{},
		log:         log,
		stopDelay:   stopDelay,
		stopTimeout: stopTimeout,
		releaseInfo: rInfo,
	}
}

func (r *BinariesRunner) cleanBinaryPath(binPath string) string {
	if !filepath.IsAbs(binPath) {
		return path.Join(r.binsFolder, binPath)
	}

	return binPath
}

func (r *BinariesRunner) runBinary(ctx context.Context, binPath string, args []string) error {
	binPath = r.cleanBinaryPath(binPath)

	if err := utils.EnsureBinary(binPath); err != nil {
		return fmt.Errorf("failed to locate binary %s %v: %w", binPath, args, err)
	}

	if r.releaseInfo != nil {
		if err := ensureBinaryVersion(binPath, r.releaseInfo.VegaReleaseTag); err != nil {
			return err
		}
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	r.log.Debug("Starting binary",
		logging.String("binaryPath", binPath),
		logging.Strings("args", args),
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start binary %s %v: %w", binPath, args, err)
	}

	processID := cmd.Process.Pid

	// Ensures that if one binary fails all of them are killed
	go func() {
		<-ctx.Done()

		if cmd.Process == nil {
			return
		}

		// Process has already exited - no need to kill it
		if cmd.ProcessState != nil {
			return
		}

		r.log.Debug("Stopping binary", logging.String("binaryPath", binPath))

		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			r.log.Debug("Failed to stop binary, resorting to force kill",
				logging.String("binaryPath", binPath),
				logging.Error(err),
			)
			if err := cmd.Process.Kill(); err != nil {
				r.log.Debug("Failed to force kill binary",
					logging.String("binaryPath", binPath),
					logging.Error(err),
				)
			}
		}
	}()

	r.mut.Lock()
	r.running[processID] = cmd
	r.mut.Unlock()

	defer func() {
		r.mut.Lock()
		delete(r.running, processID)
		r.mut.Unlock()
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed after waiting for binary %s %v: %w", binPath, args, err)
	}

	return nil
}

func (r *BinariesRunner) prepareVegaArgs(runConf *config.RunConfig, isRestart bool) (Args, error) {
	args := Args(runConf.Vega.Binary.Args)

	// if a node restart happens (not due protocol upgrade) and data node is present
	// we need to make sure that they will start on the block that data node has already processed.
	if isRestart && runConf.DataNode != nil {
		r.log.Debug("Getting latest history segment from data node (will lock the latest LevelDB snapshot!)")
		// this locks the levelDB file
		latestSegment, err := latestDataNodeHistorySegment(
			r.cleanBinaryPath(runConf.DataNode.Binary.Path),
			runConf.DataNode.Binary.Args,
		)
		r.log.Debug("Got latest history segment from data node", logging.Bool("success", err == nil))

		if err == nil {
			args.Set(snapshotBlockHeightFlagName, strconv.FormatUint(uint64(latestSegment.LatestSegment.Height), 10))
			return args, nil
		}

		// no segment was found - do not load from snapshot
		if errors.Is(err, ErrNoHistorySegmentFound) {
			return args, nil
		}

		return nil, fmt.Errorf("failed to get latest history segment from data node: %w", err)
	}

	if r.releaseInfo != nil {
		args.Set(snapshotBlockHeightFlagName, strconv.FormatUint(r.releaseInfo.UpgradeBlockHeight, 10))
	}

	return args, nil
}

func (r *BinariesRunner) Run(ctx context.Context, runConf *config.RunConfig, isRestart bool) chan error {
	r.log.Debug("Starting Vega binary")

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		args, err := r.prepareVegaArgs(runConf, isRestart)
		if err != nil {
			return fmt.Errorf("failed to prepare args for Vega binary: %w", err)
		}

		return r.runBinary(ctx, runConf.Vega.Binary.Path, args)
	})

	if runConf.DataNode != nil {
		eg.Go(func() error {
			r.log.Debug("Starting Data Node binary")
			return r.runBinary(ctx, runConf.DataNode.Binary.Path, runConf.DataNode.Binary.Args)
		})
	}

	errChan := make(chan error)

	go func() {
		err := eg.Wait()
		if err != nil {
			errChan <- err
		}
	}()

	return errChan
}

func (r *BinariesRunner) signal(signal syscall.Signal) error {
	r.mut.RLock()
	defer r.mut.RUnlock()

	var err error
	for _, c := range r.running {
		r.log.Info("Signaling process",
			logging.String("binaryName", c.Path),
			logging.String("signal", signal.String()),
			logging.Strings("args", c.Args),
		)

		err = c.Process.Signal(signal)
		if err != nil {
			r.log.Error("Failed to signal running binary",
				logging.String("binaryPath", c.Path),
				logging.Strings("args", c.Args),
				logging.Error(err),
			)
		}
	}

	return err
}

func (r *BinariesRunner) Stop() error {
	r.log.Info("Stopping binaries", logging.Duration("stop delay", r.stopDelay))

	time.Sleep(r.stopDelay)

	if err := r.signal(syscall.SIGTERM); err != nil {
		return err
	}

	r.mut.RLock()
	timeout := time.After(r.stopTimeout)
	r.mut.RUnlock()

	ticker := time.NewTicker(time.Second / 10)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("failed to gracefully shut down processes: timed out")
		case <-ticker.C:
			r.mut.RLock()
			if len(r.running) == 0 {
				r.mut.RUnlock()
				return nil
			}
			r.mut.RUnlock()
		}
	}
}

func (r *BinariesRunner) Kill() error {
	return r.signal(syscall.SIGKILL)
}
