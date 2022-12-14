package dehistory

import (
	"context"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/paths"

	"code.vegaprotocol.io/vega/datanode/dehistory/aggregation"

	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/config"
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
	fixConfig(&cmd.Config, vegaPaths)

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

	contiguousHistory := GetHighestContiguousHistoryFromHistorySegments(segments)

	if contiguousHistory != nil {
		fmt.Printf("\nAvailable contiguous decentralized history spans block %d to %d\n", contiguousHistory[0].HeightFrom,
			contiguousHistory[len(contiguousHistory)-1].HeightTo)
	} else {
		fmt.Printf("\nNo decentralized history available.  Use the fetch command to fetch decentralised history\n")
	}

	span, err := sqlstore.GetDatanodeBlockSpan(context.Background(), cmd.Config.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to get datanode block span:%w", err)
	}

	if span.HasData {
		fmt.Printf("\nDatanode currently has data from block height %d to %d\n", span.FromHeight, span.ToHeight)
	} else {
		fmt.Printf("\nDatanode contains no data\n")
	}

	return nil
}

func GetHighestContiguousHistoryFromHistorySegments(histories []*v2.HistorySegment) []aggregation.AggregatedHistorySegment {
	aggHistory := make([]aggregation.AggregatedHistorySegment, 0, 10)
	for _, history := range histories {
		aggHistory = append(aggHistory, aggregation.AggregatedHistorySegment{
			HeightFrom: history.FromHeight,
			HeightTo:   history.ToHeight,
			ChainID:    history.ChainId,
		})
	}

	return aggregation.GetHighestContiguousHistory(aggHistory)
}
