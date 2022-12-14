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

type versionCommandOutput struct {
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

type BinariesRunner struct {
	mut         sync.RWMutex
	running     map[int]*exec.Cmd
	binsFolder  string
	log         *logging.Logger
	stopTimeout time.Duration
	releaseInfo *types.ReleaseInfo
}

func NewBinariesRunner(log *logging.Logger, binsFolder string, stopTimeout time.Duration, rInfo *types.ReleaseInfo) *BinariesRunner {
	return &BinariesRunner{
		binsFolder:  binsFolder,
		running:     map[int]*exec.Cmd{},
		log:         log,
		stopTimeout: stopTimeout,
		releaseInfo: rInfo,
	}
}

func ensureBinaryVersion(binary, version string) error {
	var output versionCommandOutput
	if _, err := utils.ExecuteBinary(binary, []string{"version", "--output", "json"}, &output); err != nil {
		return err
	}

	if output.Version != version {
		return fmt.Errorf("wrong binary version provided - provided: %s, want: %s", output.Version, version)
	}

	return nil
}

func (r *BinariesRunner) runBinary(ctx context.Context, binPath string, args []string) error {
	if !filepath.IsAbs(binPath) {
		binPath = path.Join(r.binsFolder, binPath)
	}

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
		return fmt.Errorf("failed to execute binary %s %v: %w", binPath, args, err)
	}

	// Ensures that if one binary failes all of them are killed
	go func() {
		<-ctx.Done()

		if cmd.Process == nil {
			return
		}

		// Process has already exited - no need to kill it
		if cmd.ProcessState != nil {
			return
		}

		r.log.Debug("Killing binary", logging.String("binaryPath", binPath))

		if err := cmd.Process.Kill(); err != nil {
			r.log.Debug("Failed to kill binary",
				logging.String("binaryPath", binPath),
				logging.Error(err),
			)
		}
	}()

	processID := cmd.Process.Pid

	r.mut.Lock()
	r.running[processID] = cmd
	r.mut.Unlock()

	defer func() {
		r.mut.Lock()
		delete(r.running, processID)
		r.mut.Unlock()
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to execute binary %s %v: %w", binPath, args, err)
	}

	return nil
}

func (r *BinariesRunner) Run(ctx context.Context, runConf *config.RunConfig) chan error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		// TODO consider moving this logic somewhere else
		args := Args(runConf.Vega.Binary.Args)
		if r.releaseInfo != nil {
			args.Set(snapshotBlockHeightFlagName, strconv.FormatUint(r.releaseInfo.UpgradeBlockHeight, 10))
		}

		return r.runBinary(ctx, runConf.Vega.Binary.Path, args)
	})

	if runConf.DataNode != nil {
		eg.Go(func() error {
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
	for _, c := range r.running {
		r.log.Info("Stopping process",
			logging.String("binaryName", c.Path),
			logging.Strings("args", c.Args),
		)

		if err := c.Process.Kill(); err != nil {
			r.log.Error("Failed to signal running binary",
				logging.String("binaryPath", c.Path),
				logging.Strings("args", c.Args),
				logging.Error(err),
			)
		}
	}

	return nil
}

func (r *BinariesRunner) Kill() error {
	return r.signal(syscall.SIGKILL)
}
