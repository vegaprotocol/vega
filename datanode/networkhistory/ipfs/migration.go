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

package ipfs

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/ipfs/kubo/repo/fsrepo/migrations"
)

func createFetcher(distPath string, ipfsDir string) migrations.Fetcher {
	const userAgent = "fs-repo-migrations"

	if distPath == "" {
		distPath = migrations.GetDistPathEnv(migrations.LatestIpfsDist)
	}

	return migrations.NewMultiFetcher(
		newIpfsFetcher(distPath, ipfsDir, 0),
		migrations.NewHttpFetcher(distPath, "", userAgent, 0))
}

// LatestSupportedVersion returns the latest version supported by the kubo library
func latestSupportedVersion() int {
	// TODO: Maybe We should hardcode it to be safe and control when the migration happens?
	return fsrepo.RepoVersion
}

// IsMigrationNeeded check if migration of the IPFS repository is needed
func isMigrationNeeded(ipfsDir string) (bool, error) {
	repoVersion, err := migrations.RepoVersion(ipfsDir)
	if err != nil {
		return false, fmt.Errorf("failed to check version for the %s IPFS repository: %w", ipfsDir, err)
	}

	return repoVersion < latestSupportedVersion(), nil
}

func MigrateIpfsStorageVersion(log *logging.Logger, ipfsDir string) error {
	isMigrationNeeded, err := isMigrationNeeded(ipfsDir)
	if err != nil {
		return fmt.Errorf("failed to check if the ipfs migration is needed: %w", err)
	}
	if !isMigrationNeeded {
		if log != nil {
			log.Info("The IPFS for the network-history is up to date. Migration not needed")
		}
		return nil
	}

	localIpfsDir, err := migrations.IpfsDir(ipfsDir)
	if err != nil {
		return fmt.Errorf("failed to find local ipfs directory: %w", err)
	}

	fetcher := createFetcher("", localIpfsDir)
	err = migrations.RunMigration(context.Background(), fetcher, latestSupportedVersion(), localIpfsDir, false)
	if err != nil {
		return fmt.Errorf("failed to execute the ipfs migration: %w", err)
	}

	return nil
}
