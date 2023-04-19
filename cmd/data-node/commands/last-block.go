package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	coreConfig "code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/datanode/config"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgx/v4"
	"github.com/jessevdk/go-flags"
)

type LastBlockCmd struct {
	config.VegaHomeFlag
	coreConfig.OutputFlag
	*config.Config
	ctx context.Context

	Timeout time.Duration `long:"timeout" description:"Database connection timeout" default:"10s"`
}

var lastBlockCmd LastBlockCmd

func (cmd *LastBlockCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	vegaPaths := paths.New(cmd.VegaHome)

	cfgLoader, err := config.InitialiseLoader(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise configuration loader: %w", err)
	}

	cmd.Config, err = cfgLoader.Get()
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "couldn't load configuration", err)
		os.Exit(1)
	}

	connectionString := cmd.Config.SQLStore.ConnectionConfig.GetConnectionString()

	ctx, cancel := context.WithTimeout(cmd.ctx, cmd.Timeout)
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
		handleErr(log, cmd.Output.IsJSON(), "Failed to connect to database", err)
		os.Exit(1)
	}

	var lastBlock int64
	err = conn.QueryRow(ctx, "select max(height) from blocks").Scan(&lastBlock)
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "Failed to get last block", err)
		os.Exit(1)
	}

	if cmd.Output.IsJSON() {
		return vgjson.Print(struct {
			LastBlock int64 `json:"last_block"`
		}{
			LastBlock: lastBlock,
		})
	}

	log.Info("Last block", logging.Int64("height", lastBlock))
	return nil
}

func LastBlock(ctx context.Context, parser *flags.Parser) error {
	cfg := config.NewDefaultConfig()
	lastBlockCmd = LastBlockCmd{
		Config: &cfg,
		ctx:    ctx,
	}
	_, err := parser.AddCommand("last-block", "Get last block", "Get last block", &lastBlockCmd)
	return err
}

func handleErr(log *logging.Logger, outputJSON bool, msg string, err error) {
	if outputJSON {
		_ = vgjson.Print(struct {
			Error string `json:"error"`
		}{
			Error: err.Error(),
		})
	} else {
		log.Error(msg, logging.Error(err))
	}
}
