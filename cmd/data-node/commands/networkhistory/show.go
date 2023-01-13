package networkhistory

import (
	"context"
	"fmt"
	"os"
	"sort"

	coreConfig "code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/datanode/networkhistory/aggregation"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/paths"

	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/config"
)

type showCmd struct {
	config.VegaHomeFlag
	config.Config
	coreConfig.OutputFlag
}

type showOutput struct {
	Segments                   []*v2.HistorySegment
	AvailableHistoryBlockStart int64
	AvailableHistoryBlockEnd   int64
	LocalHistoryBlockStart     int64
	LocalHistoryBlockEnd       int64
}

func (o *showOutput) printHuman() {
	fmt.Printf("All Network History Segments:\n\n")
	for _, segment := range o.Segments {
		fmt.Printf("%s\n", segment)
	}

	if o.AvailableHistoryBlockEnd > 0 {
		fmt.Printf("\nAvailable contiguous network history spans block %d to %d\n",
			o.AvailableHistoryBlockStart,
			o.AvailableHistoryBlockEnd)
	} else {
		fmt.Printf("\nNo network history is available.  Use the fetch command to fetch network history\n")
	}

	if o.LocalHistoryBlockEnd > 0 {
		fmt.Printf("\nDatanode currently has data from block height %d to %d\n", o.LocalHistoryBlockStart, o.LocalHistoryBlockEnd)
	} else {
		fmt.Printf("\nDatanode contains no data\n")
	}
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
	err := fixConfig(&cmd.Config, vegaPaths)
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "failed to fix config", err)
		os.Exit(1)
	}

	if !datanodeLive(cmd.Config) {
		handleErr(log,
			cmd.Output.IsJSON(),
			"datanode must be running for this command to work",
			fmt.Errorf("couldn't connect to datanode on %v:%v", cmd.Config.API.IP, cmd.Config.API.Port))
		os.Exit(1)
	}

	client, conn, err := getDatanodeClient(cmd.Config)
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "failed to get datanode client", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	response, err := client.ListAllNetworkHistorySegments(context.Background(), &v2.ListAllNetworkHistorySegmentsRequest{})
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "failed to list all network history segments", err)
		os.Exit(1)
	}

	output := showOutput{}
	output.Segments = response.Segments

	sort.Slice(output.Segments, func(i int, j int) bool {
		return output.Segments[i].ToHeight < output.Segments[j].ToHeight
	})

	contiguousHistory := GetHighestContiguousHistoryFromHistorySegments(output.Segments)

	if contiguousHistory != nil {
		output.AvailableHistoryBlockStart = contiguousHistory[0].HeightFrom
		output.AvailableHistoryBlockEnd = contiguousHistory[len(contiguousHistory)-1].HeightTo
	}

	span, err := sqlstore.GetDatanodeBlockSpan(context.Background(), cmd.Config.SQLStore.ConnectionConfig)
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "failed to get datanode block span", err)
		os.Exit(1)
	}

	if span.HasData {
		output.LocalHistoryBlockStart = span.FromHeight
		output.LocalHistoryBlockEnd = span.ToHeight
	}

	if cmd.Output.IsJSON() {
		if err := vgjson.Print(&output); err != nil {
			handleErr(log, cmd.Output.IsJSON(), "failed to marshal output", err)
			os.Exit(1)
		}
	} else {
		output.printHuman()
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
