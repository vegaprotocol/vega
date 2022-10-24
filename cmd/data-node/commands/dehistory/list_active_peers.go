package dehistory

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type listActivePeers struct {
	config.VegaHomeFlag
	config.Config
}

func (cmd *listActivePeers) Execute(_ []string) error {
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
		return fmt.Errorf("failed to active peer addresses:%w", err)
	}
	peerAddresses := resp.IpAddresses

	if len(peerAddresses) == 0 {
		fmt.Printf("No active peers found:\n\n")
	}

	fmt.Printf("Active Peers:\n\n")

	for _, peer := range peerAddresses {
		fmt.Printf("Active Peer:  %s\n", peer)
	}

	return nil
}
