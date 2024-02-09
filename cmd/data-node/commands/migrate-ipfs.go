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
	"path/filepath"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/networkhistory/ipfs"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
)

type MigrateIpfsCmd struct {
	config.VegaHomeFlag
}

func MigrateIpfs(ctx context.Context, parser *flags.Parser) error {
	migrateIpfsCmd = MigrateIpfsCmd{}

	_, err := parser.AddCommand("migrate-ipfs", "Update IPFS store version", "Migrate IPFS store to the latest version supported by Vega", &migrateIpfsCmd)

	return err
}

var migrateIpfsCmd MigrateIpfsCmd

func (cmd *MigrateIpfsCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	vegaPaths := paths.New(cmd.VegaHome)
	ipfsDir := filepath.Join(vegaPaths.StatePathFor(paths.DataNodeNetworkHistoryHome), "store", "ipfs")

	return ipfs.MigrateIpfsStorageVersion(log, ipfsDir)
}
