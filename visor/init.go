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
	"fmt"
	"os"
	"path"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/visor/config"
	"code.vegaprotocol.io/vega/visor/utils"
)

func Init(log *logging.Logger, homeFolder string, withDataNode bool) error {
	homePath, err := utils.AbsPath(homeFolder)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %q: %w", homeFolder, err)
	}

	homeExists, err := utils.PathExists(homePath)
	if err != nil {
		return err
	}

	if homeExists {
		return fmt.Errorf("home folder %q already exists", homePath)
	}

	visorConf := config.DefaultVisorConfig(log, homePath, withDataNode)

	log.Info("Initiating genesis folder")
	if err := initDefaultFolder(visorConf.GenesisFolder(), "genesis", withDataNode); err != nil {
		return err
	}

	log.Info("Initiating vX.X.X upgrade folder")
	if err := initDefaultFolder(visorConf.UpgradeFolder("vX.X.X"), "vX.X.X", withDataNode); err != nil {
		return err
	}

	log.Info("Saving default config file")
	if err := visorConf.WriteToFile(); err != nil {
		return fmt.Errorf("failed to write config to file: %w", err)
	}

	return nil
}

func initDefaultFolder(folderPath, name string, withDataNode bool) error {
	if err := os.MkdirAll(folderPath, 0o755); err != nil {
		return fmt.Errorf("failed to create %q folder: %w", name, err)
	}

	if err := config.ExampleRunConfig(name, withDataNode).WriteToFile(path.Join(folderPath, config.RunConfigFileName)); err != nil {
		return fmt.Errorf("failed to write example config file for %q: %w", name, err)
	}

	return nil
}
