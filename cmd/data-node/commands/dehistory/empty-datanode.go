package dehistory

import (
	"fmt"

	"code.vegaprotocol.io/vega/datanode/sqlstore"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"code.vegaprotocol.io/vega/datanode/config"
)

type emptyDatanodeCmd struct {
	config.VegaHomeFlag
	config.Config
}

func (cmd *emptyDatanodeCmd) Execute(_ []string) error {
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.WarnLevel
	cfg.Environment = "custom"
	log := logging.NewLoggerFromConfig(
		cfg,
	)
	defer log.AtExit()

	vegaPaths := paths.New(cmd.VegaHome)
	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.DataNodeDefaultConfigFile)
	if err != nil {
		return fmt.Errorf("couldn't get path for %s: %w", paths.DataNodeDefaultConfigFile, err)
	}

	err = paths.ReadStructuredFile(configFilePath, &cmd.Config)
	if err != nil {
		return fmt.Errorf("failed to read configuration:%w", err)
	}

	if datanodeLive(cmd.Config) {
		return fmt.Errorf("datanode must be shutdown before datanode can be emptied")
	}

	yes := flags.YesOrNo("This will remove all data from datanode, are you sure?")

	if yes {
		err = sqlstore.WipeDatabaseAndMigrateSchemaToVersion(log, cmd.Config.SQLStore.ConnectionConfig, 0, sqlstore.EmbedMigrations)
		if err != nil {
			return fmt.Errorf("failed to wipe database and migrate to schema version 0: %w", err)
		}
		fmt.Println("Datanode is now empty")
	}

	return nil
}
