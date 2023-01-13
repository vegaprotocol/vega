package networkhistory

import (
	"context"
	"fmt"
	"os"

	coreConfig "code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/datanode/networkhistory"
	vgjson "code.vegaprotocol.io/vega/libs/json"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type latestHistorySegment struct {
	config.VegaHomeFlag
	coreConfig.OutputFlag
	config.Config
}

type segmentInfo struct {
	Peer     string
	SwarmKey string
	Segment  *v2.HistorySegment
}
type latestHistoryOutput struct {
	Segments              []segmentInfo
	SuggestedFetchSegment *v2.HistorySegment
}

func (o *latestHistoryOutput) printHuman() {
	segmentsInfo := "Most Recent History Segments:\n\n"
	for _, segment := range o.Segments {
		segmentsInfo += fmt.Sprintf("Peer:%-39s,  Swarm Key:%s, Segment{%s}\n\n", segment.Peer, segment.SwarmKey, segment.Segment)
	}
	fmt.Println(segmentsInfo)
	fmt.Printf("Suggested segment to use to fetch network history data {%s}\n\n", o.SuggestedFetchSegment)
}

func (cmd *latestHistorySegment) Execute(_ []string) error {
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
		handleErr(log, cmd.Output.IsJSON(), "failed to fix config", err)
		os.Exit(1)
	}

	if !datanodeLive(cmd.Config) {
		return fmt.Errorf("datanode must be running for this command to work")
	}

	client, conn, err := getDatanodeClient(cmd.Config)
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "failed to get datanode client", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	resp, err := client.GetActiveNetworkHistoryPeerAddresses(context.Background(), &v2.GetActiveNetworkHistoryPeerAddressesRequest{})
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "failed to get active peer addresses", errorFromGrpcError("", err))
		os.Exit(1)
	}

	peerAddresses := resp.IpAddresses

	grpcAPIPorts := []int{cmd.Config.API.Port}
	grpcAPIPorts = append(grpcAPIPorts, cmd.Config.NetworkHistory.Initialise.GrpcAPIPorts...)
	selectedResponse, peerToResponse, err := networkhistory.GetMostRecentHistorySegmentFromPeersAddresses(context.Background(), peerAddresses,
		cmd.Config.NetworkHistory.Store.GetSwarmKey(log, cmd.Config.ChainID), grpcAPIPorts)
	if err != nil {
		handleErr(log, cmd.Output.IsJSON(), "failed to get most recent history segment from peers", err)
		os.Exit(1)
	}

	output := latestHistoryOutput{}
	output.Segments = []segmentInfo{}

	for peer, segment := range peerToResponse {
		output.Segments = append(output.Segments, segmentInfo{
			Peer:     peer,
			SwarmKey: segment.SwarmKey,
			Segment:  segment.Segment,
		})
	}

	if selectedResponse != nil {
		output.SuggestedFetchSegment = selectedResponse.Response.Segment
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
