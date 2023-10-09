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
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"gopkg.in/natefinch/lumberjack.v2"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

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
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	defer log.AtExit()

	log.Info("Launching Postgres")

	// we define this option to parse the cli args each time the config is
	// loaded. So that we can respect the cli flag precedence.
	parseFlagOpt := func(cfg *config.Config) error {
		_, err := flags.NewParser(cfg, flags.Default|flags.IgnoreUnknown).Parse()
		return err
	}

	vegaPaths := paths.New(cmd.VegaHome)

	configWatcher, err := config.NewWatcher(ctx, log, vegaPaths, config.Use(parseFlagOpt))
	if err != nil {
		return err
	}

	cmd.Config = configWatcher.Get()

	lumberjackLog := &lumberjack.Logger{
		Filename: paths.StatePath(filepath.Join(paths.DataNodeLogsHome.String(), "embedded-postgres.log")).String(),
		MaxSize:  cmd.Config.SQLStore.LogRotationConfig.MaxSize,
		MaxAge:   cmd.Config.SQLStore.LogRotationConfig.MaxAge,
		Compress: true,
	}

	dbConfig := embeddedpostgres.DefaultConfig().
		Username(cmd.Config.SQLStore.ConnectionConfig.Username).
		Password(cmd.Config.SQLStore.ConnectionConfig.Password).
		Database(cmd.Config.SQLStore.ConnectionConfig.Database).
		Port(uint32(cmd.Config.SQLStore.ConnectionConfig.Port)).
		Logger(lumberjackLog).
		RuntimePath(vegaPaths.StatePathFor(paths.DataNodeStorageSQLStoreHome)).
		DataPath(vegaPaths.StatePathFor(paths.DataNodeStorageSQLStoreNodeDataHome))

	db := embeddedpostgres.NewDatabase(dbConfig)
	err = db.Start()
	if err != nil {
		return err
	}

	cmd.wait(ctx, log, cfunc)
	return db.Stop()
}

func (cmd *PostgresRunCmd) wait(ctx context.Context, log *logging.Logger, cfunc func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	for {
		select {
		case sig := <-ch:
			cfunc()
			log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
			return
		case <-ctx.Done():
			return
		}
	}
}
