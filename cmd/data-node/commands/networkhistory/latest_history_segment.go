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
	"fmt"
	"os"
	"path/filepath"

	coreConfig "code.vegaprotocol.io/vega/core/config"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
)

var errNoHistorySegmentFound = fmt.Errorf("no history segments found")

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
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
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

	ctx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	var latestSegment *v2.HistorySegment
	if datanodeLive(cmd.Config) {
		client, conn, err := getDatanodeClient(cmd.Config)
		if err != nil {
			handleErr(log, cmd.Output.IsJSON(), "failed to get datanode client", err)
			os.Exit(1)
		}
		defer func() { _ = conn.Close() }()

		response, err := client.ListAllNetworkHistorySegments(ctx, &v2.ListAllNetworkHistorySegmentsRequest{})
		if err != nil {
			handleErr(log, cmd.Output.IsJSON(), "failed to list all network history segments", errorFromGrpcError("", err))
			os.Exit(1)
		}

		if len(response.Segments) < 1 {
			handleErr(log, cmd.Output.IsJSON(), errNoHistorySegmentFound.Error(), errNoHistorySegmentFound)
			os.Exit(1)
		}

		latestSegment = response.Segments[0]
	} else {
		// we don't need to fire up a whole IPFS node, lets just dip our fingers into the DB
		idx, err := store.NewIndex(filepath.Join(vegaPaths.StatePathFor(paths.DataNodeNetworkHistoryHome), "store", "index"), log)
		if err != nil {
			handleErr(log, cmd.Output.IsJSON(), "failed to create new index", err)
			os.Exit(1)
		}
		defer idx.Close()

		segments, err := idx.ListAllEntriesOldestFirst()
		if err != nil {
			handleErr(log, cmd.Output.IsJSON(), "failed to list all network history segments", err)
			os.Exit(1)
		}

		if len(segments) < 1 {
			handleErr(log, cmd.Output.IsJSON(), errNoHistorySegmentFound.Error(), errNoHistorySegmentFound)
			os.Exit(1)
		}

		latestSegmentIndex := segments[len(segments)-1]

		latestSegment = &v2.HistorySegment{
			FromHeight:               latestSegmentIndex.GetFromHeight(),
			ToHeight:                 latestSegmentIndex.GetToHeight(),
			HistorySegmentId:         latestSegmentIndex.GetHistorySegmentId(),
			PreviousHistorySegmentId: latestSegmentIndex.GetPreviousHistorySegmentId(),
		}
	}

	output := latestHistoryOutput{
		LatestSegment: latestSegment,
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
