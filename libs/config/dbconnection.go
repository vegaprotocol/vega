// Copyright (C) 2023  Gobalsky Labs Limited
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

package config

import (
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/libs/config/encoding"

	"github.com/jackc/pgx/v4/pgxpool"
)

type PostgresConnection struct {
	Host             string            `description:"hostname of postgres server"                                               long:"host"`
	Port             int               `description:"port postgres is running on"                                               long:"port"`
	Username         string            `description:"username to connect with"                                                  long:"username"`
	Password         string            `description:"password for user"                                                         long:"password"`
	Database         string            `description:"database name"                                                             long:"database"`
	SocketDir        string            `description:"location of postgres UNIX socket directory (used if host is empty string)" long:"socket-dir"`
	ApplicationName  string            `description:"identify the application to the database using this name"                  long:"application-name"`
	StatementTimeout encoding.Duration `description:"Terminate any database connections that take longer than this"             long:"statement-timeout"`
}

func (conf PostgresConnection) ToConnectionString() string {
	if conf.Host == "" {
		//nolint:nosprintfhostport
		return fmt.Sprintf("postgresql://%s:%s@/%s?host=%s&port=%d",
			conf.Username,
			conf.Password,
			conf.Database,
			conf.SocketDir,
			conf.Port)
	}
	//nolint:nosprintfhostport
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		conf.Username,
		conf.Password,
		conf.Host,
		conf.Port,
		conf.Database)
}

func (conf PostgresConnection) ToPgxPoolConfig() (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(conf.ToConnectionString())
	if err != nil {
		return nil, err
	}

	if conf.ApplicationName != "" {
		cfg.ConnConfig.RuntimeParams["application_name"] = "Vega Data Node"
	}

	if conf.StatementTimeout.Get() > 0 {
		cfg.ConnConfig.RuntimeParams["statement_timeout"] = strconv.Itoa(int(conf.StatementTimeout.Get().Milliseconds()))
	}

	return cfg, nil
}
