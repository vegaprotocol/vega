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

package config

import (
	"context"
	"fmt"
	"path"
	"runtime"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"github.com/fsnotify/fsnotify"
	"golang.org/x/sync/errgroup"
)

const (
	currentFolder      = "current"
	genesisFolder      = "genesis"
	configFileName     = "config.toml"
	RunConfigFileName  = "run-config.toml"
	VegaBinaryName     = "vega"
	DataNodeBinaryName = "data-node"
)

/*
description: Defines the name of the asset to be downloaded.
*/
type AssetsConfig struct {
	// description: Name of the asset file on Github.
	Name string `toml:"name"`
	/*
		description: |
			Name of the binary if the asset is a zip file and the binary is included inside of it.
	*/
	BinaryName *string `toml:"binaryName"`
}

func (a AssetsConfig) GetBinaryPath() string {
	if a.BinaryName != nil {
		return *a.BinaryName
	}

	return a.Name
}

/*
description: Determines if the assets should be automatically downloaded and installed. If so this defines the assets that should be downloaded from GitHub for a specific release.

example:

	type: toml
	value: |
		[autoInstall]
			enabled = true
			repositoryOwner = "vegaprotocol"
			repository = "vega"
			[autoInstall.asset]
				name = "vega-darwin-amd64.zip"
				binaryName = "vega"
*/
type AutoInstallConfig struct {
	/*
		description: Auto Install flag
		default: true
	*/
	Enabled bool `toml:"enabled"`
	/*
		description: Owner of the repository from where the assets should be downloaded.
		default: vegaprotocol
	*/
	GithubRepositoryOwner string `toml:"repositoryOwner"`
	/*
		description: Name of the repository from where the assets should be downloaded.
		default: vega
	*/
	GithubRepository string `toml:"repository"`
	/*
		description: |
			Definition of the asset that should be downloaded from the GitHub repository.
			If the asset is contained in a zip file, the name of the binary is given.
		example:
			type: toml
			value: |
				[autoInstall.asset]
					name = "vega-darwin-amd64.zip"
					binaryName = "vega"
	*/
	Asset AssetsConfig `toml:"asset"`
}

/*
description: Root of the config file
example:

	type: toml
	value: |
		maxNumberOfRestarts = 3
		restartsDelaySeconds = 5

		[upgradeFolders]
			"vX.X.X" = "vX.X.X"

		[autoInstall]
			enabled = false
*/
type VisorConfigFile struct {
	/*
		description: |
			Visor communicates with the core node via RPC API.
			This variable allows a validator to specify how many times Visor should try to establish a connection to the core node before the Visor process fails.
			The `maxNumberOfFirstConnectionRetries` is only taken into account during the first start up of the Core node process - not restarts.
		note: |
			There is a 2 second delay between each attempt. Setting the max retry number to 5 means Visor will try to establish a connection 5 times in 10 seconds.
		default: 10
	*/
	MaxNumberOfFirstConnectionRetries int `toml:"maxNumberOfFirstConnectionRetries,optional"`
	/*
		description: |
			Defines the maximum number of restarts in case any of
			the processes have failed before the Visor process fails.
		note: |
			The amount of time Visor waits between restarts can be set by `restartsDelaySeconds`.
		default: 3
	*/
	MaxNumberOfRestarts int `toml:"maxNumberOfRestarts,optional"`
	/*
		description: |
			Number of seconds that Visor waits before it tries to re-start the processes.
		default: 5
	*/
	RestartsDelaySeconds int `toml:"restartsDelaySeconds,optional"`
	/*
		description: |
			Number of seconds that Visor waits before it sends a termination signal (SIGTERM) to running processes
			that are ready for upgrade.
			After the time has elapsed Visor stops the running binaries (SIGTERM).
		default: 0
	*/
	StopDelaySeconds int `toml:"stopDelaySeconds,optional"`
	/*
		description: |
			Number of seconds that Visor waits after it sends termination signal (SIGTERM) to running processes.
			After the time has elapsed Visor force-kills (SIGKILL) any running processes.
		default: 15
	*/
	StopSignalTimeoutSeconds int `toml:"stopSignalTimeoutSeconds,optional"`

	/*
		description: |
			During the upgrade, by default Visor looks for a folder with a name identical to the upgrade version.
			The default behaviour can be changed by providing mapping between `version` and `custom_folder_name`.
			If a custom mapping is provided, during the upgrade Visor uses the folder given in the mapping for specific version.

		example:
			type: toml
			value: |
				[upgradeFolders]
					"v99.9.9" = "custom_upgrade_folder_name"
	*/
	UpgradeFolders map[string]string `toml:"upgradeFolders,optional"`

	/*
		description: |
			Defines the assets that should be automatically downloaded from Github for a specific release.

		example:
			type: toml
			value: |
				[autoInstall]
					enabled = true
					repositoryOwner = "vegaprotocol"
					repository = "vega"
					[autoInstall.assets]
						[autoInstall.assets.vega]
							asset_name = "vega-darwin-amd64.zip"
							binary_name = "vega"

	*/
	AutoInstall AutoInstallConfig `toml:"autoInstall"`
}

func parseAndValidateVisorConfigFile(path string) (*VisorConfigFile, error) {
	conf := VisorConfigFile{}
	if err := paths.ReadStructuredFile(path, &conf); err != nil {
		return nil, fmt.Errorf("failed to parse VisorConfig: %w", err)
	}

	return &conf, nil
}

type VisorConfig struct {
	mut        sync.RWMutex
	configPath string
	homePath   string
	data       *VisorConfigFile
	log        *logging.Logger
}

func DefaultVisorConfig(log *logging.Logger, homePath string) *VisorConfig {
	return &VisorConfig{
		log:        log,
		homePath:   homePath,
		configPath: path.Join(homePath, configFileName),
		data: &VisorConfigFile{
			UpgradeFolders:                    map[string]string{"vX.X.X": "vX.X.X"},
			MaxNumberOfRestarts:               3,
			MaxNumberOfFirstConnectionRetries: 10,
			RestartsDelaySeconds:              5,
			StopDelaySeconds:                  0,
			StopSignalTimeoutSeconds:          15,
			AutoInstall: AutoInstallConfig{
				Enabled:               true,
				GithubRepositoryOwner: "vegaprotocol",
				GithubRepository:      "vega",
				Asset: AssetsConfig{
					Name:       fmt.Sprintf("vega-%s-%s.zip", runtime.GOOS, "amd64"),
					BinaryName: toPointer("vega"),
				},
			},
		},
	}
}

func NewVisorConfig(log *logging.Logger, homePath string) (*VisorConfig, error) {
	configPath := path.Join(homePath, configFileName)

	dataFile, err := parseAndValidateVisorConfigFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &VisorConfig{
		configPath: configPath,
		homePath:   homePath,
		data:       dataFile,
		log:        log,
	}, nil
}

func (pc *VisorConfig) reload() error {
	pc.log.Info("Reloading config")
	dataFile, err := parseAndValidateVisorConfigFile(pc.configPath)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	pc.mut.Lock()
	pc.data.MaxNumberOfFirstConnectionRetries = dataFile.MaxNumberOfFirstConnectionRetries
	pc.data.MaxNumberOfRestarts = dataFile.MaxNumberOfRestarts
	pc.data.RestartsDelaySeconds = dataFile.RestartsDelaySeconds
	pc.data.StopSignalTimeoutSeconds = dataFile.StopSignalTimeoutSeconds
	pc.data.StopDelaySeconds = dataFile.StopDelaySeconds
	pc.data.UpgradeFolders = dataFile.UpgradeFolders
	pc.data.AutoInstall = dataFile.AutoInstall
	pc.mut.Unlock()

	pc.log.Info("Reloading config success")

	return nil
}

func (pc *VisorConfig) WatchForUpdate(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	var eg errgroup.Group
	eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				if event.Has(fsnotify.Write) {
					// add a small sleep here in order to handle vi
					// vi do not send a write event / edit the file in place,
					// it always create a temporary file, then delete the original one,
					// and then rename the temp file with the name of the original file.
					// if we try to update the conf as soon as we get the event, the file is not
					// always created and we get a no such file or directory error
					time.Sleep(50 * time.Millisecond)

					pc.reload()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
				return err
			}
		}
	})

	if err := watcher.Add(pc.configPath); err != nil {
		return err
	}

	return eg.Wait()
}

func (pc *VisorConfig) CurrentFolder() string {
	pc.mut.RLock()
	defer pc.mut.RUnlock()

	return path.Join(pc.homePath, currentFolder)
}

func (pc *VisorConfig) CurrentRunConfigPath() string {
	pc.mut.RLock()
	defer pc.mut.RUnlock()

	return path.Join(pc.CurrentFolder(), RunConfigFileName)
}

func (pc *VisorConfig) GenesisFolder() string {
	pc.mut.RLock()
	defer pc.mut.RUnlock()

	return path.Join(pc.homePath, genesisFolder)
}

func (pc *VisorConfig) UpgradeFolder(releaseTag string) string {
	pc.mut.RLock()
	defer pc.mut.RUnlock()

	if folderName, ok := pc.data.UpgradeFolders[releaseTag]; ok {
		return path.Join(pc.homePath, folderName)
	}

	return path.Join(pc.homePath, releaseTag)
}

func (pc *VisorConfig) MaxNumberOfRestarts() int {
	pc.mut.RLock()
	defer pc.mut.RUnlock()

	return pc.data.MaxNumberOfRestarts
}

func (pc *VisorConfig) MaxNumberOfFirstConnectionRetries() int {
	pc.mut.RLock()
	defer pc.mut.RUnlock()

	return pc.data.MaxNumberOfFirstConnectionRetries
}

func (pc *VisorConfig) RestartsDelaySeconds() int {
	pc.mut.RLock()
	defer pc.mut.RUnlock()

	return pc.data.RestartsDelaySeconds
}

func (pc *VisorConfig) StopSignalTimeoutSeconds() int {
	pc.mut.RLock()
	defer pc.mut.RUnlock()

	return pc.data.StopSignalTimeoutSeconds
}

func (pc *VisorConfig) StopDelaySeconds() int {
	pc.mut.RLock()
	defer pc.mut.RUnlock()

	return pc.data.StopDelaySeconds
}

func (pc *VisorConfig) AutoInstall() AutoInstallConfig {
	pc.mut.RLock()
	defer pc.mut.RUnlock()

	return pc.data.AutoInstall
}

func (pc *VisorConfig) WriteToFile() error {
	pc.mut.RLock()
	defer pc.mut.RUnlock()

	return paths.WriteStructuredFile(pc.configPath, pc.data)
}

func toPointer[T any](val T) *T {
	return &val
}
