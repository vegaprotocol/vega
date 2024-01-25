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

package store

import (
	"time"

	"code.vegaprotocol.io/vega/libs/config"
)

var namedLogger = "postgres.store"

type Config struct {
	Postgres             config.PostgresConnection `group:"database" namespace:"postgres"`
	MigrateData          bool                      `default:"true"   description:"Migrate data from the old database"                                          group:"database" namespace:"postgres"`
	MigrateBlockDuration time.Duration             `default:"1h"     description:"Amount of data to migrate at a time, in duration, i.e. 1h, 4h etc."          group:"database" namespace:"postgres"`
	MigratePauseInterval time.Duration             `default:"1m"     description:"Pause migrations between dates to prevent block explorer from being blocked" group:"database" namespace:"postgres"`
}

func NewDefaultConfig() Config {
	return Config{
		Postgres: config.PostgresConnection{
			Host:            "localhost",
			Port:            5432,
			Database:        "tendermint_indexer_db",
			Username:        "vega",
			Password:        "vega",
			ApplicationName: "vega block explorer",
		},
		MigrateData:          true,
		MigrateBlockDuration: time.Hour,
		MigratePauseInterval: time.Minute,
	}
}
