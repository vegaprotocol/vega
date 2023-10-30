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
