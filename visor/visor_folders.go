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
	"os"
	"path"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/visor/config"
	"code.vegaprotocol.io/vega/visor/github"
	"code.vegaprotocol.io/vega/visor/utils"
)

var vegaDataNodeStartCmdArgs = []string{"datanode", "start"}

func (v *Visor) setCurrentFolder(sourceFolder, currentFolder string) error {
	v.log.Info("Setting current folder",
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

func (v *Visor) installUpgradeFolder(ctx context.Context, folder, releaseTag string, conf config.AutoInstallConfig) error {
	v.log.Info("Automatically installing upgrade folder")

	runConf, err := config.ParseRunConfig(v.conf.CurrentRunConfigPath())
	if err != nil {
		return err
	}

	if conf.Asset.Name == "" {
		return missingAutoInstallAssetError("vega")
	}

	if err := os.MkdirAll(folder, 0o755); err != nil {
		return fmt.Errorf("failed to create upgrade folder %q, %w", folder, err)
	}

	assetsFetcher := github.NewAssetsFetcher(
		conf.GithubRepositoryOwner,
		conf.GithubRepository,
		[]string{conf.Asset.Name},
	)

	v.log.Info("Downloading asset from Github", logging.String("asset", conf.Asset.Name))
	if err := assetsFetcher.Download(ctx, releaseTag, folder); err != nil {
		return fmt.Errorf("failed to download release assets for tag %q: %w", releaseTag, err)
	}

	runConf.Name = releaseTag
	runConf.Vega.Binary.Path = conf.Asset.GetBinaryPath()

	if runConf.DataNode != nil {
		runConf.DataNode.Binary.Path = conf.Asset.GetBinaryPath()

		if len(runConf.DataNode.Binary.Args) != 0 && runConf.DataNode.Binary.Args[0] != vegaDataNodeStartCmdArgs[0] {
			runConf.DataNode.Binary.Args = append(vegaDataNodeStartCmdArgs, runConf.DataNode.Binary.Args[1:]...)
		}
	}

	runConfPath := path.Join(folder, config.RunConfigFileName)
	if err := runConf.WriteToFile(runConfPath); err != nil {
		return fmt.Errorf("failed to create run config %q: %w", runConfPath, err)
	}

	return nil
}

func (v *Visor) prepareNextUpgradeFolder(ctx context.Context, releaseTag string) error {
	v.log.Debug("preparing next upgrade folder",
		logging.String("vegaTagVersion", releaseTag),
	)

	upgradeFolder := v.conf.UpgradeFolder(releaseTag)
	upgradeFolderExists, err := utils.PathExists(upgradeFolder)
	if err != nil {
		return err
	}

	if !upgradeFolderExists {
		autoInstallConf := v.conf.AutoInstall()
		if !autoInstallConf.Enabled {
			return fmt.Errorf("required upgrade folder %q is missing", upgradeFolder)
		}

		if err := v.installUpgradeFolder(ctx, upgradeFolder, releaseTag, autoInstallConf); err != nil {
			return fmt.Errorf("failed to auto install folder %q for release %q: %w", upgradeFolder, releaseTag, err)
		}
	}

	if err := v.setCurrentFolder(upgradeFolder, v.conf.CurrentFolder()); err != nil {
		return fmt.Errorf("failed to set current folder to %q: %w", v.conf.CurrentFolder(), err)
	}

	return nil
}

func missingAutoInstallAssetError(asset string) error {
	return fmt.Errorf("missing required auto install %s asset definition in Visor config", asset)
}
