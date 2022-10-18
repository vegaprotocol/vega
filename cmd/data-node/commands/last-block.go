package commands

import (
	"context"
	"os"
	"time"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgx/v4"
	"github.com/jessevdk/go-flags"
)

type LastBlockCmd struct {
	config.VegaHomeFlag
	config.Config

	Timeout time.Duration `long:"timeout" description:"Database connection timeout" default:"10s"`
}

var lastBlockCmd LastBlockCmd

func (cmd *LastBlockCmd) Execute(args []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

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
	connectionString := cmd.Config.SQLStore.ConnectionConfig.GetConnectionString()

	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()
	var conn *pgx.Conn

	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = time.Second
	expBackoff.MaxInterval = time.Second
	expBackoff.MaxElapsedTime = cmd.Timeout

	// Retry the connect in case we have to wait for the database to start
	err = backoff.Retry(func() (opErr error) {
		conn, opErr = pgx.Connect(ctx, connectionString)
		return opErr
	}, expBackoff)
	if err != nil {
		log.Error("Failed to connect to database", logging.Error(err))
		os.Exit(1)
	}

	var lastBlock int64
	err = conn.QueryRow(ctx, "select max(height) from blocks").Scan(&lastBlock)
	if err != nil {
		log.Error("Unable to retrieve last block", logging.Error(err))
		os.Exit(1)
	}

	log.Info("Last block", logging.Int64("height", lastBlock))
	return nil
}

func LastBlock(ctx context.Context, parser *flags.Parser) error {
	lastBlockCmd = LastBlockCmd{
		Config: config.NewDefaultConfig(),
	}
	_, err := parser.AddCommand("last-block", "Get last block", "Get last block", &lastBlockCmd)
	return err
}
