package networkhistory

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"google.golang.org/grpc/status"

	"code.vegaprotocol.io/vega/datanode/networkhistory"

	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/sqlstore"

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
		return errors.New("expected <history-segment-id> <num-blocks-to-fetch>")
	}

	rootSegmentID := args[0]

	numBlocksToFetch, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse number of blocks to fetch: %w", err)
	}

	vegaPaths := paths.New(cmd.VegaHome)
	err = fixConfig(&cmd.Config, vegaPaths)
	if err != nil {
		return fmt.Errorf("failed to fix config:%w", err)
	}

	err = verifyChainID(log, cmd.SQLStore.ConnectionConfig, cmd.ChainID)
	if err != nil {
		return fmt.Errorf("failed to verify chain id:%w", err)
	}

	if !datanodeLive(cmd.Config) {
		return fmt.Errorf("datanode must be running for this command to work")
	}

	client := getDatanodeAdminClient(log, cmd.Config)
	blocksFetched, err := networkhistory.FetchHistoryBlocks(context.Background(), func(s string, args ...interface{}) {
		fmt.Printf(s+"\n", args...)
	}, rootSegmentID,
		func(ctx context.Context, historySegmentId string) (networkhistory.FetchResult, error) {
			resp, err := client.FetchNetworkHistorySegment(context.Background(), historySegmentId)
			if err != nil {
				return networkhistory.FetchResult{},
					errorFromGrpcError("failed to fetch network history segment", err)
			}

			return networkhistory.FetchResult{
				HeightFrom:               resp.HeightFrom,
				HeightTo:                 resp.HeightTo,
				PreviousHistorySegmentID: resp.PreviousHistorySegmentID,
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
		return fmt.Errorf("failed to create new transactional connection source: %w", err)
	}

	defer connSource.Close()

	store := sqlstore.NewChain(connSource)
	chainService := service.NewChain(store)

	err = networkhistory.VerifyChainID(chainID, chainService)
	if err != nil {
		return fmt.Errorf("failed to verify chain id:%w", err)
	}
	return nil
}

func errorFromGrpcError(msg string, err error) error {
	s, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("%s:%s", msg, err)
	}

	return fmt.Errorf("%s:%s", msg, s.Details())
}
