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

package networkhistory

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/networkhistory"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"google.golang.org/grpc/status"
)

type fetchCmd struct {
	config.VegaHomeFlag
	config.Config
}

func (cmd *fetchCmd) Execute(args []string) error {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
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

	err = verifyChainID(cmd.SQLStore.ConnectionConfig, cmd.ChainID)
	if err != nil {
		return fmt.Errorf("failed to verify chain id:%w", err)
	}

	if !datanodeLive(cmd.Config) {
		return fmt.Errorf("datanode must be running for this command to work")
	}

	client := getDatanodeAdminClient(log, cmd.Config)
	blocksFetched, err := networkhistory.FetchHistoryBlocks(ctx, func(s string, args ...interface{}) {
		fmt.Printf(s+"\n", args...)
	}, rootSegmentID,
		func(ctx context.Context, historySegmentId string) (networkhistory.FetchResult, error) {
			resp, err := client.FetchNetworkHistorySegment(ctx, historySegmentId)
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

func verifyChainID(connConfig sqlstore.ConnectionConfig, chainID string) error {
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
