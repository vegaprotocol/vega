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

package main

import (
	"context"
	"fmt"

	"github.com/jessevdk/go-flags"
)

type VersionCmd struct {
	version string
	hash    string
}

func (cmd *VersionCmd) Execute(_ []string) error {
	fmt.Printf("Vega CLI %s (%s)\n", cmd.version, cmd.hash)
	return nil
}

var versionCmd VersionCmd

func Version(ctx context.Context, parser *flags.Parser) error {
	versionCmd = VersionCmd{
		version: CLIVersion,
		hash:    CLIVersionHash,
	}

	_, err := parser.AddCommand("version", "Show version info", "Show version info", &versionCmd)
	return err
}
