package networkhistory

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/admin"

	"code.vegaprotocol.io/vega/datanode/networkhistory"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
	"google.golang.org/grpc"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/config"
)

type Cmd struct {
	// Subcommands
	Show                          showCmd                       `command:"show"                              description:"shows network history segments currently stored by the node"`
	Load                          loadCmd                       `command:"load"                              description:"load [from height] [to height], loads the given span of network history into the datanode, if no span is specified the latest contiguous network history will be loaded"`
	Rollback                      rollbackCmd                   `command:"rollback"                          description:"rollback [to height], rolls back the datanode to the given height, the rollback height must match the to height of a network history segment"`
	Fetch                         fetchCmd                      `command:"fetch"                             description:"fetch <history segment id> <blocks to fetch>, fetches the given segment and all previous segments until <blocks to fetch> blocks have been retrieved"`
	LatestHistorySegmentFromPeers latestHistorySegmentFromPeers `command:"latest-history-segment-from-peers" description:"latest-history-segment-from-peers returns the id of the networks latest history segment"`
	LatestHistorySegment          latestHistorySegment          `command:"latest-history-segment"            description:"latest-history-segment returns the id of the node's latest history segment"`
	ListActivePeers               listActivePeers               `command:"list-active-peers"                 description:"list the active datanode peers"`
	Copy                          copyCmd                       `command:"copy"                              description:"copy a history segment from network history to a file"`
}

var networkHistoryCmd Cmd

func NetworkHistory(_ context.Context, parser *flags.Parser) error {
	cfg := config.NewDefaultConfig()
	networkHistoryCmd = Cmd{
		Show: showCmd{},
		Load: loadCmd{
			Config: cfg,
		},
		Fetch:                         fetchCmd{},
		LatestHistorySegmentFromPeers: latestHistorySegmentFromPeers{},
		LatestHistorySegment:          latestHistorySegment{},
		ListActivePeers:               listActivePeers{},
		Copy: copyCmd{
			Config: cfg,
		},
	}

	desc := "commands for managing network history"
	_, err := parser.AddCommand("network-history", desc, desc, &networkHistoryCmd)
	if err != nil {
		return err
	}
	return nil
}

func getDatanodeClient(cfg config.Config) (v2.TradingDataServiceClient, *grpc.ClientConn, error) {
	return networkhistory.GetDatanodeClientFromIPAndPort(cfg.API.IP, cfg.API.Port)
}

func getDatanodeAdminClient(log *logging.Logger, cfg config.Config) *admin.Client {
	client := admin.NewClient(log, cfg.Admin)
	return client
}

func datanodeLive(cfg config.Config) bool {
	client, conn, err := getDatanodeClient(cfg)
	if err != nil {
		return false
	}
	defer conn.Close()

	_, err = client.Ping(context.Background(), &v2.PingRequest{})
	return err == nil
}

// getConfig figures out where to read a config file from, reads it, and then applies any extra
// modifications on top of that.
//
// This is working around a bit of awkwardness in that the config supplied by go-flags is a blank
// config updated with command line flags. There is not enough information in it to apply an
// 'overlay' to a config read from a file, because it is not possible for us to tell if someone
// is trying to override a value back to its 'zero' value. (e.g. --something.enabled=false gives
// the same go-flags structure as no argument at all).
func fixConfig(config *config.Config, vegaPaths paths.Paths) error {
	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.DataNodeDefaultConfigFile)
	if err != nil {
		return fmt.Errorf("couldn't get path for %s: %w", paths.DataNodeDefaultConfigFile, err)
	}

	// Read config from file
	err = paths.ReadStructuredFile(configFilePath, config)
	if err != nil {
		return fmt.Errorf("failed to read config:%w", err)
	}

	// Apply command-line flags on top
	_, err = flags.NewParser(config, flags.Default|flags.IgnoreUnknown).Parse()
	if err != nil {
		return fmt.Errorf("failed to parse args:%w", err)
	}
	return nil
}

func handleErr(log *logging.Logger, outputJSON bool, msg string, err error) {
	if outputJSON {
		_ = vgjson.Print(struct {
			Error string `json:"error"`
		}{
			Error: err.Error(),
		})
	} else {
		log.Error(msg, logging.Error(err))
	}
}
