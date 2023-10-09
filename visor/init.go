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

	visorConf := config.DefaultVisorConfig(log, homePath)

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
