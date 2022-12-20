package dehistory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type dumpSegment struct {
	config.VegaHomeFlag
	config.Config
}

func (cmd *dumpSegment) Execute(args []string) error {
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.InfoLevel
	cfg.Environment = "custom"
	log := logging.NewLoggerFromConfig(
		cfg,
	)
	defer log.AtExit()

	var err error

	if len(args) != 1 {
		return errors.New("expected <history-segment-id>")
	}

	segmentID := args[0]

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

	path, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory:%w", err)
	}

	targetFilePath := filepath.Join(path, segmentID+".tar")
	_, err = client.CopyHistorySegmentToFile(context.Background(), &v2.CopyHistorySegmentToFileRequest{
		HistorySegmentId: segmentID,
		TargetFile:       targetFilePath,
	})

	if err != nil {
		return errorFromGrpcError("failed to copy segment to target file", err)
	}

	fmt.Printf("segment %s dumped to target file %s\n", segmentID, targetFilePath)

	return nil
}
