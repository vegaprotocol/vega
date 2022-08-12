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
	"path"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/visor/client"
	"code.vegaprotocol.io/vega/visor/config"
	"code.vegaprotocol.io/vega/visor/utils"
)

const (
	upgradeApiCallTickerDuration = time.Second * 2
	maxUpgradeStatusErrs         = 10
)

type Visor struct {
	conf          *config.VisorConfig
	clientFactory client.ClientFactory
	log           *logging.Logger
}

func NewVisor(ctx context.Context, log *logging.Logger, clientFactory client.ClientFactory, homePath string) (*Visor, error) {
	homePath, err := utils.AbsPath(homePath)
	if err != nil {
		return nil, err
	}

	homeExists, err := utils.PathExists(homePath)
	if err != nil {
		return nil, err
	}
	if !homeExists {
		return nil, fmt.Errorf("home folder %q does not exists, it can initiated with init command", homePath)
	}

	visorConf, err := config.NewVisorConfig(log, homePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	currentFolderExists, err := utils.PathExists(visorConf.CurrentRunConfigPath())
	if err != nil {
		return nil, fmt.Errorf("failed to check if %q path exists: %w", visorConf.CurrentRunConfigPath(), err)
	}

	r := &Visor{
		conf:          visorConf,
		clientFactory: clientFactory,
		log:           log,
	}

	if !currentFolderExists {
		if err := r.setCurrentFolder(visorConf.GenesisFolder(), visorConf.CurrentFolder()); err != nil {
			return nil, fmt.Errorf("failed to set current folder to %q: %w", visorConf.CurrentFolder(), err)
		}
	}

	go r.watchForConfigUpdates(ctx)

	return r, nil
}

func (r *Visor) watchForConfigUpdates(ctx context.Context) {
	for {
		r.log.Debug("starting config file watcher")
		if err := r.conf.WatchForUpdate(ctx); err != nil {
			r.log.Error("config file watcher has failed", logging.Error(err))
		}
	}
}

func (r *Visor) setCurrentFolder(sourceFolder, currentFolder string) error {
	r.log.Debug("setting current folder",
		logging.String("sourceFolder", sourceFolder),
		logging.String("currentFolder", currentFolder),
	)

	runConfPath := path.Join(sourceFolder, config.RunConfigFileName)
	runConfExists, err := utils.PathExists(runConfPath)
	if err != nil {
		return err
	}

	if !runConfExists {
		return fmt.Errorf("missing run config in %q folder", runConfPath)
	}

	if err := os.RemoveAll(currentFolder); err != nil {
		return fmt.Errorf("failed to remove current folder: %w", err)
	}

	if err := os.Symlink(sourceFolder, currentFolder); err != nil {
		return fmt.Errorf("failed to set current folder as current: %w", err)
	}

	return nil
}

func (r *Visor) Run(ctx context.Context) error {
	numOfRestarts := 0
	var currentRelaseInfo *types.ReleaseInfo

	upgradeTicker := time.NewTicker(upgradeApiCallTickerDuration)
	defer upgradeTicker.Stop()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		runConf, err := config.ParseRunConfig(r.conf.CurrentRunConfigPath())
		if err != nil {
			return fmt.Errorf("failed to parse run config: %w", err)
		}

		client := r.clientFactory.GetClient(
			runConf.Vega.RCP.SocketPath,
			runConf.Vega.RCP.HttpPath,
		)

		numOfUpgradeStatusErrs := 0
		maxNumRestarts := r.conf.MaxNumberOfRestarts()
		restartsDelay := time.Second * time.Duration(r.conf.RestartsDelaySeconds())

		r.log.Info("Starting binaries")
		binRunner := NewBinariesRunner(r.log, r.conf.CurrentFolder(), time.Second*time.Duration(r.conf.StopSignalTimeoutSeconds()))
		binErrs := binRunner.Run(ctx, runConf, currentRelaseInfo)

		upgradeTicker.Reset(upgradeApiCallTickerDuration)

	CheckLoop:
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-binErrs:
				r.log.Error("Binaries executions has failed", logging.Error(err))

				if numOfRestarts >= maxNumRestarts {
					return fmt.Errorf("maximum number of possible restarts has been reached: %w", err)
				}

				numOfRestarts++
				r.log.Info("Binaries restart is scheduled", logging.Duration("restartDelay", restartsDelay))
				time.Sleep(restartsDelay)
				r.log.Info("Restarting binaries", logging.Int("remainingRestarts", maxNumRestarts-numOfRestarts))

				break CheckLoop
			case <-upgradeTicker.C:
				upStatus, err := client.UpgradeStatus(ctx)
				if err != nil {
					if numOfUpgradeStatusErrs > maxUpgradeStatusErrs {
						return fmt.Errorf("failed to upgrade status for maximum amount of %d times: %w", maxUpgradeStatusErrs, err)
					}

					r.log.Debug("failed to get upgrade status from API", logging.Error(err))
					numOfUpgradeStatusErrs++

					break
				}

				if !upStatus.ReadyToUpgrade {
					break
				}

				currentRelaseInfo = upStatus.AcceptedReleaseInfo

				r.log.Info("Preparing upgrade")

				if err := binRunner.Stop(); err != nil {
					// Force to kill if fails grateful stop fails
					if err := binRunner.Kill(); err != nil {
						return fmt.Errorf("failed to force kill the running processes: %w", err)
					}
				}

				r.log.Info("Starting upgrade")

				if err := r.setCurrentFolder(r.conf.UpgradeFolder(), r.conf.CurrentFolder()); err != nil {
					return fmt.Errorf("failed to set current folder to %q: %w", r.conf.CurrentFolder(), err)
				}

				numOfRestarts = 0

				break CheckLoop
			}
		}
	}
}
