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
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/data-node/datanode/config"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/shared/paths"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jessevdk/go-flags"
)

type PostgresCmd struct {
	Run PostgresRunCmd `command:"run"`
}

var postgresCmd PostgresCmd

func Postgres(ctx context.Context, parser *flags.Parser) error {
	postgresCmd = PostgresCmd{
		Run: PostgresRunCmd{},
	}

	_, err := parser.AddCommand("postgres", "Embedded Postgres", "Embedded Postgres", &postgresCmd)
	return err
}

type PostgresRunCmd struct {
	config.VegaHomeFlag
	config.Config
}

func (cmd *PostgresRunCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	log.Info("Launching Postgres")

	// we define this option to parse the cli args each time the config is
	// loaded. So that we can respect the cli flag precedence.
	parseFlagOpt := func(cfg *config.Config) error {
		_, err := flags.NewParser(cfg, flags.Default|flags.IgnoreUnknown).Parse()
		return err
	}

	vegaPaths := paths.New(cmd.VegaHome)

	configWatcher, err := config.NewWatcher(context.Background(), log, vegaPaths, config.Use(parseFlagOpt))
	if err != nil {
		return err
	}

	cmd.Config = configWatcher.Get()

	stateDir := vegaPaths.StatePathFor(paths.DataNodeStorageHome)

	dbConfig := embeddedpostgres.DefaultConfig().
		Username(cmd.Config.SQLStore.ConnectionConfig.Username).
		Password(cmd.Config.SQLStore.ConnectionConfig.Password).
		Database(cmd.Config.SQLStore.ConnectionConfig.Database).
		Port(uint32(cmd.Config.SQLStore.ConnectionConfig.Port)).
		RuntimePath(paths.JoinStatePath(paths.StatePath(stateDir), "sqlstore").String()).
		DataPath(paths.JoinStatePath(paths.StatePath(stateDir), "sqlstore", "node-data").String())

	db := embeddedpostgres.NewDatabase(dbConfig)
	err = db.Start()
	if err != nil {
		return err
	}

	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

	// It would be nice to watch the child process itself and return if it exits unexpectedly,
	// but embedded-postgres-go doesn't give us a way to do that right now.
	sig := <-gracefulStop
	log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
	return db.Stop()
}
