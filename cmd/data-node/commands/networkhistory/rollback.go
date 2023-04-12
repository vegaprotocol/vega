package networkhistory

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	vgterm "code.vegaprotocol.io/vega/libs/term"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/datanode/networkhistory"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"code.vegaprotocol.io/vega/datanode/config"
)

type rollbackCmd struct {
	config.VegaHomeFlag
	config.Config

	Force bool `short:"f" long:"force" description:"do not prompt for confirmation"`
}

func (cmd *rollbackCmd) Execute(args []string) error {
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.WarnLevel
	cfg.Environment = "custom"
	log := logging.NewLoggerFromConfig(
		cfg,
	)
	defer log.AtExit()

	if len(args) != 1 {
		return errors.New("expected <rollback-to-height>")
	}

	rollbackToHeight, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse rollback to height: %w", err)
	}

	vegaPaths := paths.New(cmd.VegaHome)
	err = fixConfig(&cmd.Config, vegaPaths)
	if err != nil {
		return fmt.Errorf("failed to fix config:%w", err)
	}

	if datanodeLive(cmd.Config) {
		return fmt.Errorf("datanode must be shutdown before it can be rolled back")
	}

	if !cmd.Force && vgterm.HasTTY() {
		if !flags.YesOrNo("Running this command will kill all existing database connections, do you want to continue?") {
			return nil
		}
	}

	if err := networkhistory.KillAllConnectionsToDatabase(context.Background(), cmd.SQLStore.ConnectionConfig); err != nil {
		return fmt.Errorf("failed to kill all connections to database: %w", err)
	}

	connPool, err := getCommandConnPool(cmd.Config.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to get command connection pool: %w", err)
	}
	defer connPool.Close()

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	networkHistoryService, err := createNetworkHistoryService(ctx, log, cmd.Config, connPool, vegaPaths)
	if err != nil {
		return fmt.Errorf("failed to created network history service: %w", err)
	}
	defer networkHistoryService.Stop()

	loadLog := newLoadLog()
	defer loadLog.AtExit()
	err = networkHistoryService.RollbackToHeight(context.Background(), loadLog, rollbackToHeight)
	if err != nil {
		return fmt.Errorf("failed to rollback datanode: %w", err)
	}

	fmt.Printf("Rolled back datanode to height %d\n", rollbackToHeight)

	return nil
}
