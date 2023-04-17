package networkhistory

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/jackc/pgx/v4/pgxpool"

	coreConfig "code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/config"
)

type showCmd struct {
	config.VegaHomeFlag
	config.Config
	coreConfig.OutputFlag

	AllSegments bool `short:"s" long:"segments" description:"show all segments for each contiguous history"`
}

type showOutput struct {
	Segments            []*v2.HistorySegment
	ContiguousHistories []segment.ContiguousHistory[*v2.HistorySegment]
	DataNodeBlockStart  int64
	DataNodeBlockEnd    int64
}

func (o *showOutput) printHuman(allSegments bool) {
	if len(o.ContiguousHistories) > 0 {
		fmt.Printf("Available contiguous history spans:")
		for _, contiguousHistory := range o.ContiguousHistories {
			fmt.Printf("\n\nContiguous history from block height %d to %d, from segment id: %s to %s\n",
				contiguousHistory.HeightFrom,
				contiguousHistory.HeightTo,
				contiguousHistory.Segments[0].GetHistorySegmentId(),
				contiguousHistory.Segments[len(contiguousHistory.Segments)-1].GetHistorySegmentId(),
			)

			if allSegments {
				for _, segment := range contiguousHistory.Segments {
					fmt.Printf("\n%d to %d, id: %s, previous segment id: %s",
						segment.GetFromHeight(),
						segment.GetToHeight(),
						segment.GetHistorySegmentId(),
						segment.GetPreviousHistorySegmentId())
				}
			}
		}
	} else {
		fmt.Printf("\nNo network history is available.  Use the fetch command to fetch network history\n")
	}

	if o.DataNodeBlockEnd > 0 {
		fmt.Printf("\n\nDatanode currently has data from block height %d to %d\n", o.DataNodeBlockStart, o.DataNodeBlockEnd)
	} else {
		fmt.Printf("\n\nDatanode contains no data\n")
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

	segments := segment.Segments[*v2.HistorySegment](response.Segments)
	output.ContiguousHistories = segments.AllContigousHistories()

	pool, err := getCommandConnPool(cmd.Config.SQLStore.ConnectionConfig)
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "failed to get command conn pool", err)
	}
	defer pool.Close()

	span, err := sqlstore.GetDatanodeBlockSpan(context.Background(), pool)
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "failed to get datanode block span", err)
		os.Exit(1)
	}

	if span.HasData {
		output.DataNodeBlockStart = span.FromHeight
		output.DataNodeBlockEnd = span.ToHeight
	}

	if cmd.Output.IsJSON() {
		if err := vgjson.Print(&output); err != nil {
			handleErr(log, cmd.Output.IsJSON(), "failed to marshal output", err)
			os.Exit(1)
		}
	} else {
		output.printHuman(cmd.AllSegments)
	}

	return nil
}

func getCommandConnPool(conf sqlstore.ConnectionConfig) (*pgxpool.Pool, error) {
	conf.MaxConnPoolSize = 3

	connPool, err := sqlstore.CreateConnectionPool(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return connPool, nil
}
