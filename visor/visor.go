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
		return nil, fmt.Errorf("home folder %q does not exists, it can initiated with init command", homePath)
	}

	visorConf, err := config.NewVisorConfig(log, homePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
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

					v.log.Debug("failed to get upgrade status from API", logging.Error(err))
					numOfUpgradeStatusErrs++

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
