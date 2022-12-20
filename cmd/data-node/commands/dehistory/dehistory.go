package dehistory

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/dehistory"
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
	Show                          showCmd              `command:"show" description:"shows decentralised history segments currently stored by the node"`
	Load                          loadCmd              `command:"load" description:"loads the most recent contiguous history from decentralised history into the datanode"`
	Fetch                         fetchCmd             `command:"fetch" description:"fetch <start from history segment id> <blocks to fetch>, fetches the given number of blocks into this node's decentralised history"`
	LatestHistorySegmentFromPeers latestHistorySegment `command:"latest-history-segment-from-peers" description:"latest-history-segment returns the id of the networks latest history segment"`
	ListActivePeers               listActivePeers      `command:"list-active-peers" description:"list the active datanode peers"`
	DumpSegment                   dumpSegment          `command:"dump-segment" description:"dumps the specified segment to disk"`
}

var dehistoryCmd Cmd

func DeHistory(ctx context.Context, parser *flags.Parser) error {
	dehistoryCmd = Cmd{
		Show:                          showCmd{},
		Load:                          loadCmd{},
		Fetch:                         fetchCmd{},
		LatestHistorySegmentFromPeers: latestHistorySegment{},
		ListActivePeers:               listActivePeers{},
		DumpSegment:                   dumpSegment{},
	}

	desc := "commands for managing decentralised history"
	_, err := parser.AddCommand("dehistory", desc, desc, &dehistoryCmd)
	if err != nil {
		return err
	}
	return nil
}

func getDatanodeClient(cfg config.Config) (v2.DeHistoryServiceClient, *grpc.ClientConn, error) {
	return dehistory.GetDatanodeClientFromIPAndPort(cfg.API.IP, cfg.API.Port)
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
// is trying to override a value back to it's 'zero' value. (e.g. --something.enabled=false gives
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
