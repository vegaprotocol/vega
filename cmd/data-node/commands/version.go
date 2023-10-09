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

package commands

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/config"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/version"

	"github.com/jessevdk/go-flags"
)

type VersionCmd struct {
	version string
	hash    string
	config.OutputFlag
}

func (cmd *VersionCmd) Execute(_ []string) error {
	if cmd.Output.IsJSON() {
		return vgjson.Print(struct {
			Version string `json:"version"`
			Hash    string `json:"hash"`
		}{
			Version: cmd.version,
			Hash:    cmd.hash,
		})
	}

	fmt.Printf("Vega Datanode CLI %s (%s)\n", cmd.version, cmd.hash)
	return nil
}

var versionCmd VersionCmd

func Version(ctx context.Context, parser *flags.Parser) error {
	versionCmd = VersionCmd{
		version: version.Get(),
		hash:    version.GetCommitHash(),
	}

	_, err := parser.AddCommand("version", "Show version info", "Show version info", &versionCmd)
	return err
}
