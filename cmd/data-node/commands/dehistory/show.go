package dehistory

import (
	"context"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/datanode/dehistory/initialise"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/paths"
)

type showCmd struct {
	config.VegaHomeFlag
	config.Config
}

func (cmd *showCmd) Execute(_ []string) error {
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
		return fmt.Errorf("failed to read config:%w", err)
	}

	if !datanodeLive(cmd.Config) {
		return fmt.Errorf("datanode must be running for this command to work")
	}

	client, conn, err := getDatanodeClient(cmd.Config)
	if err != nil {
		return fmt.Errorf("failed to get datanode client:%w", err)
	}
	defer func() { _ = conn.Close() }()

	response, err := client.ListAllDeHistorySegments(context.Background(), &v2.ListAllDeHistorySegmentsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list all dehistory segments:%w", err)
	}

	segments := response.Segments

	sort.Slice(segments, func(i int, j int) bool {
		return segments[i].ToHeight < segments[j].ToHeight
	})

	fmt.Printf("All Decentralized History Segments:\n\n")
	for _, segment := range segments {
		fmt.Printf("%s\n", segment)
	}

	datanodeFromHeight, datanodeToHeight, err := initialise.GetDatanodeBlockSpan(context.Background(), cmd.Config.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to get datanode block span:%w", err)
	}

	if datanodeFromHeight == datanodeToHeight {
		fmt.Printf("\nDatanode contains no data\n")
	} else {
		fmt.Printf("\nDatanode has data from block height %d to %d\n", datanodeFromHeight, datanodeToHeight)
	}

	return nil
}
