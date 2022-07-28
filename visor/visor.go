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
	"log"
	"os"
	"path"
	"time"

	"code.vegaprotocol.io/vega/visor/config"
	"code.vegaprotocol.io/vega/visor/utils"
)

const upgradeApiCallTickerDuration = time.Second * 2

type Visor struct {
	conf *config.VisorConfig
}

func NewVisor(ctx context.Context, homePath string) (*Visor, error) {
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

	visorConf, err := config.NewVisorConfig(homePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	currentFolderExists, err := utils.PathExists(visorConf.CurrentRunConfigPath())
	if err != nil {
		return nil, err
	}

	if !currentFolderExists {
		if err := setCurrentFolder(visorConf.GenesisFolder(), visorConf.CurrentFolder()); err != nil {
			return nil, fmt.Errorf("failed to set current folder to %q: %w", visorConf.CurrentFolder(), err)
		}
	}

	r := &Visor{
		conf: visorConf,
	}

	go r.watchForConfigUpdates(ctx)

	return r, nil
}

func (r *Visor) watchForConfigUpdates(ctx context.Context) {
	for {
		if err := r.conf.WatchForUpdate(ctx); err != nil {
			// TODO - notify the run thread that this has failed
			log.Printf("config file watcher has failed: %s", err)
		}
	}
}

func (r *Visor) Run(ctx context.Context) error {
	numOfRestarts := 0

	upgradeTicker := time.NewTicker(upgradeApiCallTickerDuration)
	defer upgradeTicker.Stop()

	for {
		runConf, err := config.ParseRunConfig(r.conf.CurrentRunConfigPath())
		if err != nil {
			return fmt.Errorf("failed to parse run config: %w", err)
		}

		log.Printf("Running with run conf %+v", runConf)

		upgradeTicker.Reset(upgradeApiCallTickerDuration)

		maxNumRestarts := r.conf.MaxNumberOfRestarts()
		restartsDelay := time.Second * time.Duration(r.conf.RestartsDelaySeconds())

		binRunner := NewBinariesRunner(r.conf.CurrentFolder())
		errs := binRunner.Run(ctx, runConf.Binaries)

	CheckLoop:
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-errs:
				log.Printf("binaries executions has failed: %s", err)

				if numOfRestarts >= maxNumRestarts {
					return fmt.Errorf("reached maximum number of possible restarts: %w", err)
				}

				numOfRestarts++
				log.Printf("restart is scheduled, wating for %s seconds", restartsDelay)
				time.Sleep(restartsDelay)

				log.Printf("restarting, remaining num of restarts %d", maxNumRestarts-numOfRestarts)

				break CheckLoop
			case <-upgradeTicker.C: // TODO fail to process if the upgrade check is failing for a long time
				upStatus, err := UpgradeStatus(runConf.RCPAddress)
				if err != nil {
					log.Printf("failed to fetch update: %s", err)
					continue
				}

				if !upStatus.Result.ReadyToUpgrade {
					continue
				}

				if err := binRunner.Stop(); err != nil {
					// Force to kill if fails grateful stop fails
					if err := binRunner.Kill(); err != nil {
						return fmt.Errorf("failed to force kill the running processes: %w", err)
					}
				}

				log.Printf(
					"starting upgrade to Vega %q and Data Node %q",
					upStatus.Result.AcceptedReleaseInfo.VegaReleaseTag,
					upStatus.Result.AcceptedReleaseInfo.DatanodeReleaseTag,
				)

				if err := setCurrentFolder(r.conf.GenesisFolder(), r.conf.CurrentFolder()); err != nil {
					return fmt.Errorf("failed to set current folder to %q: %w", r.conf.CurrentFolder(), err)
				}

				break CheckLoop
			}
		}
	}
}

func setCurrentFolder(sourceFolder, currentFolder string) error {
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
