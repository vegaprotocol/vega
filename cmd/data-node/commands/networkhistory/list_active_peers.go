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

	coreConfig "code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/datanode/config"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type listActivePeers struct {
	config.VegaHomeFlag
	config.Config
	coreConfig.OutputFlag
}

type listActivePeersOutput struct {
	ActivePeers []string
}

func (o *listActivePeersOutput) printHuman() {
	if len(o.ActivePeers) == 0 {
		fmt.Printf("No active peers found\n")
		return
	}
	fmt.Printf("Active Peers:\n\n")

	for _, peer := range o.ActivePeers {
		fmt.Printf("Active Peer:  %s\n", peer)
	}
}

func (cmd *listActivePeers) Execute(_ []string) error {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.InfoLevel
	cfg.Environment = "custom"
	log := logging.NewLoggerFromConfig(
		cfg,
	)
	defer log.AtExit()

	vegaPaths := paths.New(cmd.VegaHome)
	err := fixConfig(&cmd.Config, vegaPaths)
	if err != nil {
		handleErr(log,
			cmd.Output.IsJSON(),
			"failed to fix config",
			err)
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
		handleErr(log,
			cmd.Output.IsJSON(),
			"failed to get datanode client",
			err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	resp, err := client.GetActiveNetworkHistoryPeerAddresses(ctx, &v2.GetActiveNetworkHistoryPeerAddressesRequest{})
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "failed to get active peer addresses", errorFromGrpcError("", err))
		os.Exit(1)
	}

	output := listActivePeersOutput{ActivePeers: resp.IpAddresses}

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
