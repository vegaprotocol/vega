package dehistory

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/dehistory"

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
}

var dehistoryCmd Cmd

func DeHistory(ctx context.Context, parser *flags.Parser) error {
	dehistoryCmd = Cmd{
		Show:                          showCmd{},
		Load:                          loadCmd{},
		Fetch:                         fetchCmd{},
		LatestHistorySegmentFromPeers: latestHistorySegment{},
		ListActivePeers:               listActivePeers{},
	}

	desc := "commands for managing decentralised history"
	_, err := parser.AddCommand("dehistory", desc, desc, &dehistoryCmd)
	if err != nil {
		return err
	}
	return nil
}

func getDatanodeClient(cfg config.Config) (v2.TradingDataServiceClient, *grpc.ClientConn, error) {
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
