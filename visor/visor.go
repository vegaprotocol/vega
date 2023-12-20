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

package visor

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/visor/client"
	"code.vegaprotocol.io/vega/visor/config"
	"code.vegaprotocol.io/vega/visor/utils"
)

const (
	upgradeAPICallTickerDuration = time.Second * 2
	maxUpgradeStatusErrs         = 10
	namedLogger                  = "visor"
)

type Visor struct {
	conf          *config.VisorConfig
	clientFactory client.Factory
	log           *logging.Logger
}

func NewVisor(ctx context.Context, log *logging.Logger, clientFactory client.Factory, homePath string) (*Visor, error) {
	homePath, err := utils.AbsPath(homePath)
	if err != nil {
		return nil, err
	}

	homeExists, err := utils.PathExists(homePath)
	if err != nil {
		return nil, err
	}

	if !homeExists {
		return nil, fmt.Errorf("visor is not initialized, call the `init` command first")
	}

	visorConf, err := config.NewVisorConfig(log, homePath)
	if err != nil {
		// Do not wrap error as underlying errors are meaningful enough.
		return nil, err
	}

	currentFolderExists, err := utils.PathExists(visorConf.CurrentRunConfigPath())
	if err != nil {
		return nil, err
	}

	v := &Visor{
		conf:          visorConf,
		clientFactory: clientFactory,
		log:           log.Named(namedLogger),
	}

	if !currentFolderExists {
		if err := v.setCurrentFolder(visorConf.GenesisFolder(), visorConf.CurrentFolder()); err != nil {
			return nil, fmt.Errorf("failed to set current folder to %q: %w", visorConf.CurrentFolder(), err)
		}
	}

	go v.watchForConfigUpdates(ctx)

	return v, nil
}

func (v *Visor) watchForConfigUpdates(ctx context.Context) {
	for {
		v.log.Debug("starting config file watcher")
		if err := v.conf.WatchForUpdate(ctx); err != nil {
			v.log.Error("config file watcher has failed", logging.Error(err))
		}
	}
}

func (v *Visor) Run(ctx context.Context) error {
	numOfRestarts := 0
	var currentReleaseInfo *types.ReleaseInfo

	upgradeTicker := time.NewTicker(upgradeAPICallTickerDuration)
	defer upgradeTicker.Stop()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var isRestarting bool

	for {
		runConf, err := config.ParseRunConfig(v.conf.CurrentRunConfigPath())
		if err != nil {
			return fmt.Errorf("failed to parse run config: %w", err)
		}

		c := v.clientFactory.GetClient(
			runConf.Vega.RCP.SocketPath,
			runConf.Vega.RCP.HTTPPath,
		)

		maxNumberOfFirstConnectionRetries := v.conf.MaxNumberOfFirstConnectionRetries()

		numOfUpgradeStatusErrs := 0
		maxNumRestarts := v.conf.MaxNumberOfRestarts()
		restartsDelay := time.Second * time.Duration(v.conf.RestartsDelaySeconds())

		if isRestarting {
			v.log.Info("Restarting binaries")
		} else {
			v.log.Info("Starting binaries")
		}

		binRunner := NewBinariesRunner(
			v.log,
			v.conf.CurrentFolder(),
			time.Second*time.Duration(v.conf.StopDelaySeconds()),
			time.Second*time.Duration(v.conf.StopSignalTimeoutSeconds()),
			currentReleaseInfo,
		)
		binErrs := binRunner.Run(ctx, runConf, isRestarting)

		upgradeTicker.Reset(upgradeAPICallTickerDuration)
		isRestarting = false

	CheckLoop:
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-binErrs:
				v.log.Error("Binaries executions has failed", logging.Error(err))

				if numOfRestarts >= maxNumRestarts {
					return fmt.Errorf("maximum number of possible restarts has been reached: %w", err)
				}

				numOfRestarts++
				v.log.Info("Binaries restart is scheduled", logging.Duration("restartDelay", restartsDelay))
				time.Sleep(restartsDelay)
				v.log.Info("Restarting binaries", logging.Int("remainingRestarts", maxNumRestarts-numOfRestarts))

				isRestarting = true

				break CheckLoop
			case <-upgradeTicker.C:
				upStatus, err := c.UpgradeStatus(ctx)
				if err != nil {
					// Binary has not started yet - waiting for first startup
					if numOfRestarts == 0 {
						if numOfUpgradeStatusErrs > maxNumberOfFirstConnectionRetries {
							return failedToGetStatusErr(maxNumberOfFirstConnectionRetries, err)
						}
					} else { // Binary has been started already. Something has failed after the startup
						if numOfUpgradeStatusErrs > maxUpgradeStatusErrs {
							return failedToGetStatusErr(maxUpgradeStatusErrs, err)
						}
					}

					v.log.Debug("Failed to get upgrade status from API", logging.Error(err))

					numOfUpgradeStatusErrs++

					v.log.Info("Still waiting for vega to start...", logging.Int("attemptLeft", maxUpgradeStatusErrs-numOfUpgradeStatusErrs))

					break
				}

				if !upStatus.ReadyToUpgrade {
					break
				}

				currentReleaseInfo = upStatus.AcceptedReleaseInfo

				v.log.Info("Preparing upgrade")

				if err := binRunner.Stop(); err != nil {
					v.log.Info("Failed to stop binaries, resorting to force kill", logging.Error(err))
					if err := binRunner.Kill(); err != nil {
						return fmt.Errorf("failed to force kill the running processes: %w", err)
					}
				}

				v.log.Info("Starting upgrade")

				if err := v.prepareNextUpgradeFolder(ctx, currentReleaseInfo.VegaReleaseTag); err != nil {
					return fmt.Errorf("failed to prepare next upgrade folder: %w", err)
				}

				numOfRestarts = 0
				numOfUpgradeStatusErrs = 0

				break CheckLoop
			}
		}
	}
}

func failedToGetStatusErr(numberOfErrs int, err error) error {
	return fmt.Errorf("failed to get upgrade status for maximum amount of %d times: %w", numberOfErrs, err)
}
