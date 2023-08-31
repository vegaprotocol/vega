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
