package networkhistory

import (
	"context"
	"fmt"
	"os"

	coreConfig "code.vegaprotocol.io/vega/core/config"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
)

type latestHistorySegment struct {
	config.VegaHomeFlag
	coreConfig.OutputFlag
	config.Config
}

type latestHistoryOutput struct {
	LatestSegment *v2.HistorySegment
}

func (o *latestHistoryOutput) printHuman() {
	fmt.Printf("Latest segment to use data {%s}\n\n", o.LatestSegment)
}

func (cmd *latestHistorySegment) Execute(_ []string) error {
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.ErrorLevel
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

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	networkHistoryStore, err := store.New(ctx, log, cmd.Config.ChainID, cmd.Config.NetworkHistory.Store, vegaPaths.StatePathFor(paths.DataNodeNetworkHistoryHome), false)
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "failed to create network history store", err)
		os.Exit(1)
	}

	segments, err := networkHistoryStore.ListAllIndexEntriesOldestFirst()
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "failed to list network history segments", err)
		os.Exit(1)
	}

	if len(segments) < 1 {
		err := fmt.Errorf("no history segments found")
		handleErr(log, cmd.Output.IsJSON(), err.Error(), err)
		os.Exit(1)
	}

	latest := segments[len(segments)-1]

	output := latestHistoryOutput{
		LatestSegment: &v2.HistorySegment{
			FromHeight:               latest.GetFromHeight(),
			ToHeight:                 latest.GetToHeight(),
			HistorySegmentId:         latest.GetHistorySegmentId(),
			PreviousHistorySegmentId: latest.GetPreviousHistorySegmentId(),
		},
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
