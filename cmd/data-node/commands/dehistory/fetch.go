package dehistory

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/sqlstore"

	"code.vegaprotocol.io/vega/datanode/dehistory/initialise"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
)

type fetchCmd struct {
	config.VegaHomeFlag
	config.Config
}

func (cmd *fetchCmd) Execute(args []string) error {
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.InfoLevel
	cfg.Environment = "custom"
	log := logging.NewLoggerFromConfig(
		cfg,
	)
	defer log.AtExit()

	if len(args) != 2 {
		return errors.New("expected <start-history-segment-id> <num-blocks-to-fetch>")
	}

	rootSegmentID := args[0]

	numBlocksToFetch, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse number of blocks to fetch: %w", err)
	}

	vegaPaths := paths.New(cmd.VegaHome)

	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.DataNodeDefaultConfigFile)
	if err != nil {
		return fmt.Errorf("couldn't get path for %s: %w", paths.DataNodeDefaultConfigFile, err)
	}

	err = paths.ReadStructuredFile(configFilePath, &cmd.Config)
	if err != nil {
		return fmt.Errorf("failed to read config:%w", err)
	}

	err = verifyChainID(log, cmd.SQLStore.ConnectionConfig, cmd.ChainID)
	if err != nil {
		return fmt.Errorf("failed to verify chain id:%w", err)
	}

	if !datanodeLive(cmd.Config) {
		return fmt.Errorf("datanode must be running for this command to work")
	}

	client, conn, err := getDatanodeClient(cmd.Config)
	if err != nil {
		return fmt.Errorf("failed to get datanode client:%w", err)
	}
	defer func() { _ = conn.Close() }()

	blocksFetched, err := initialise.FetchHistoryBlocks(context.Background(), func(s string, args ...interface{}) {
		fmt.Printf(s+"\n", args...)
	}, rootSegmentID,
		func(ctx context.Context, historySegmentId string) (initialise.FetchResult, error) {
			resp, err := client.FetchDeHistorySegment(context.Background(), &v2.FetchDeHistorySegmentRequest{
				HistorySegmentId: historySegmentId,
			})
			if err != nil {
				return initialise.FetchResult{}, fmt.Errorf("failed to fetch decentralized history segments:%w", err)
			}

			return initialise.FetchResult{
				HeightFrom:               resp.Segment.FromHeight,
				HeightTo:                 resp.Segment.ToHeight,
				PreviousHistorySegmentID: resp.Segment.PreviousHistorySegmentId,
			}, nil
		}, numBlocksToFetch)
	if err != nil {
		return fmt.Errorf("failed to fetch history blocks:%w", err)
	}

	fmt.Printf("\nfinished fetching history, %d blocks retrieved \n", blocksFetched)

	return nil
}

func verifyChainID(log *logging.Logger, connConfig sqlstore.ConnectionConfig, chainID string) error {
	connSource, err := sqlstore.NewTransactionalConnectionSource(logging.NewTestLogger(), connConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database:%w", err)
	}
	defer connSource.Close()

	store := sqlstore.NewChain(connSource)
	chainService := service.NewChain(store, log)

	err = initialise.VerifyChainID(chainID, chainService)
	if err != nil {
		return fmt.Errorf("failed to verify chain id:%w", err)
	}
	return nil
}
