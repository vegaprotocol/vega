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

	"github.com/jackc/pgx/v4/pgxpool"
)

type PostgresConnection struct {
	Host            string `long:"host" description:"hostname of postgres server"`
	Port            int    `long:"port" description:"port postgres is running on"`
	Username        string `long:"username" description:"username to connect with"`
	Password        string `long:"password" description:"password for user"`
	Database        string `long:"database" description:"database name"`
	ApplicationName string `long:"application-name" description:"identify the application to the database using this name"`
}

func (conf PostgresConnection) ToConnectionString() string {
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
	return cfg, nil
}
