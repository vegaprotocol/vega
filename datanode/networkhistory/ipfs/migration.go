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
	"fmt"

	"code.vegaprotocol.io/vega/logging"

	mg12 "github.com/ipfs/fs-repo-migrations/fs-repo-12-to-13/migration"
	mg13 "github.com/ipfs/fs-repo-migrations/fs-repo-13-to-14/migration"
	mg14 "github.com/ipfs/fs-repo-migrations/fs-repo-14-to-15/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/ipfs/kubo/repo/fsrepo/migrations"
)

// LatestSupportedVersion returns the latest version supported by the kubo library.
func latestSupportedVersion() int {
	// TODO: Maybe We should hardcode it to be safe and control when the migration happens?
	return fsrepo.RepoVersion
}

// MigrateIpfsStorageVersion migrates the IPFS store to the latest supported by the
// library version.
func MigrateIpfsStorageVersion(log *logging.Logger, ipfsDir string) error {
	repoVersion, err := migrations.RepoVersion(ipfsDir)
	if err != nil {
		return fmt.Errorf("failed to check version for the %s IPFS repository: %w", ipfsDir, err)
	}

	// migration not needed
	if repoVersion >= latestSupportedVersion() {
		if log != nil {
			log.Info("The IPFS for the network-history is up to date. Migration not needed")
		}

		return nil
	}

	localIpfsDir, err := migrations.IpfsDir(ipfsDir)
	if err != nil {
		return fmt.Errorf("failed to find local ipfs directory: %w", err)
	}

	// fetcher := createFetcher("", localIpfsDir)
	// err = migrations.RunMigration(context.Background(), fetcher, latestSupportedVersion(), localIpfsDir, false)
	if err := runMigrations(repoVersion, localIpfsDir); err != nil {
		return fmt.Errorf("failed to execute the ipfs migration: %w", err)
	}

	return nil
}

func runMigrations(currentVersion int, ipfsDir string) error {
	migrationsSteps, err := requiredMigrations(currentVersion)
	if err != nil {
		return fmt.Errorf("failed to determine required migrations: %w", err)
	}

	for _, m := range migrationsSteps {
		if err := m.Apply(migrate.Options{
			Flags: migrate.Flags{
				// Force: true,
				Revert:   false,
				Path:     ipfsDir,
				Verbose:  true,
				NoRevert: true,
			},
			Verbose: true,
		}); err != nil {
			return fmt.Errorf("failed to run migration for %s: %w", m.Versions(), err)
		}
	}

	return nil
}

func requiredMigrations(currentVersion int) ([]migrate.Migration, error) {
	availableMigrations := []migrate.Migration{
		// We do not care about older versions. We are migrating repository
		// for vega 0.73. `Vega@v0.73` has `kubo@v0.20.0` which should contain
		// repository v13 (as per https://github.com/ipfs/fs-repo-migrations/tree/master?tab=readme-ov-file#when-should-i-migrate).
		// Older versions depends on ancient versions of packages that causes
		// issue for go 1.21...
		nil,               // 0-1
		nil,               // 1-2
		nil,               // 2-3
		nil,               // 3-4
		nil,               // 4-5
		nil,               // 5-6
		nil,               // 6-7
		nil,               // 7-8
		nil,               // 8-9
		nil,               // 9-10
		nil,               // 10-11
		nil,               // 11-12
		&mg12.Migration{}, // 12-13
		&mg13.Migration{}, // 13-14
		&mg14.Migration{}, // 14-15
	}

	requiredMigrations := []migrate.Migration{}

	// no migration required
	if currentVersion >= len(availableMigrations) {
		return requiredMigrations, nil
	}

	for fromVersion := currentVersion; fromVersion < len(availableMigrations); fromVersion++ {
		if availableMigrations[fromVersion] == nil {
			return nil, fmt.Errorf("migration from version %d is not supported: minimum supported version to migrate the ipfs repo from is 13", fromVersion)
		}
		requiredMigrations = append(requiredMigrations, availableMigrations[fromVersion])
	}

	return requiredMigrations, nil
}
