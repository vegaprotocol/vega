package dehistory

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/dehistory"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type latestHistorySegment struct {
	config.VegaHomeFlag
	config.Config
}

func (cmd *latestHistorySegment) Execute(_ []string) error {
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.InfoLevel
	cfg.Environment = "custom"
	log := logging.NewLoggerFromConfig(
		cfg,
	)
	defer log.AtExit()

	var err error

	vegaPaths := paths.New(cmd.VegaHome)

	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.DataNodeDefaultConfigFile)
	if err != nil {
		return fmt.Errorf("couldn't get path for %s: %w", paths.DataNodeDefaultConfigFile, err)
	}

	err = paths.ReadStructuredFile(configFilePath, &cmd.Config)
	if err != nil {
		return fmt.Errorf("failed to read config:%w", err)
	}

	if !datanodeLive(cmd.Config) {
		return fmt.Errorf("datanode must be running for this command to work")
	}

	client, conn, err := getDatanodeClient(cmd.Config)
	if err != nil {
		return fmt.Errorf("failed to get datanode client:%w", err)
	}
	defer func() { _ = conn.Close() }()

	resp, err := client.GetActiveDeHistoryPeerAddresses(context.Background(), &v2.GetActiveDeHistoryPeerAddressesRequest{})
	if err != nil {
		return errorFromGrpcError("failed to get active peer addresses", err)
	}
	peerAddresses := resp.IpAddresses

	grpcAPIPorts := []int{cmd.Config.API.Port}
	grpcAPIPorts = append(grpcAPIPorts, cmd.Config.DeHistory.Initialise.GrpcAPIPorts...)
	selectedResponse, peerToResponse, err := dehistory.GetMostRecentHistorySegmentFromPeersAddresses(context.Background(), peerAddresses,
		cmd.Config.DeHistory.Store.GetSwarmKey(log, cmd.Config.ChainID), grpcAPIPorts)

	segmentsInfo := "Most Recent History Segments:\n\n"
	for peer, segment := range peerToResponse {
		segmentsInfo += fmt.Sprintf("Peer:%-39s,  Swarm Key:%s, Segment{%s}\n\n", peer, segment.SwarmKey, segment.Segment)
	}

	fmt.Println(segmentsInfo)

	if err != nil {
		return fmt.Errorf("failed to get most recent history segment from peers:%w", err)
	}

	fmt.Printf("Suggested segment to use to fetch decentralised history data {%s}\n\n", selectedResponse.Response.Segment)

	return nil
}
