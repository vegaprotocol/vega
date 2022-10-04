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
	"fmt"

	"code.vegaprotocol.io/vega/blockexplorer/config"
	"code.vegaprotocol.io/vega/blockexplorer/store"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jessevdk/go-flags"
)

type InitDBCmd struct {
	config.VegaHomeFlag
}

func (opts *InitDBCmd) Execute(_ []string) error {
	logger := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer logger.AtExit()

	config, err := loadConfig(logger, opts.VegaHome)
	if err != nil {
		return err
	}

	err = store.MigrateToLatestSchema(logger, config.Store)
	if err != nil {
		return fmt.Errorf("creating db schema: %w", err)
	}

	return nil
}

var initDBCmd InitDBCmd

func InitDB(ctx context.Context, parser *flags.Parser) error {
	initDBCmd = InitDBCmd{}

	short := "Initialize / update database schema"
	long := "Creates, (or updates) database tables and views according to the schema required for the tendermint psql indexer"

	_, err := parser.AddCommand("init-db", short, long, &initDBCmd)
	return err
}
