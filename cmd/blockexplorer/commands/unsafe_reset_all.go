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

package commands

import (
	"context"

	"code.vegaprotocol.io/vega/blockexplorer/config"
	"code.vegaprotocol.io/vega/blockexplorer/store"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jessevdk/go-flags"
)

type UnsafeResetAllCmd struct {
	config.VegaHomeFlag
}

func (opts *UnsafeResetAllCmd) Execute(_ []string) error {
	logger := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer logger.AtExit()

	config, err := loadConfig(logger, opts.VegaHome)
	if err != nil {
		return err
	}

	err = store.DropAllTablesAndViews(logger, config.Store)
	if err != nil {
		return err
	}
	return nil
}

var unsafeResetAllCmd UnsafeResetAllCmd

func UnsafeResetAll(ctx context.Context, parser *flags.Parser) error {
	unsafeResetAllCmd = UnsafeResetAllCmd{}

	short := "Drop all data & schema from the database"
	long := "Delete all tables & views from the database (but not the database itself)"

	_, err := parser.AddCommand("unsafe-reset-all", short, long, &unsafeResetAllCmd)
	return err
}
