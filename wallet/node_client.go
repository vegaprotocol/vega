package wallet

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto/api"

	"google.golang.org/grpc"
)

type nodeClient struct {
	log     *logging.Logger
	nodeCfg NodeConfig
	clt     api.TradingDataServiceClient
	conn    *grpc.ClientConn
}

func NewNodeClient(
	log *logging.Logger, nodeConfig NodeConfig,
) (*nodeClient, error) {
	nodeAddr := fmt.Sprintf("%v:%v", nodeConfig.IP, nodeConfig.Port)
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := api.NewTradingDataServiceClient(conn)
	return &nodeClient{
		log:     log,
		nodeCfg: nodeConfig,
		clt:     client,
		conn:    conn,
	}, nil
}

func (n *nodeClient) Stop() error {
	n.log.Info("closing grpc client",
		logging.String("address",
			fmt.Sprintf("%v:%v", n.nodeCfg.IP, n.nodeCfg.Port)))
	return n.conn.Close()
}

func (n *nodeClient) LastBlockHeight(ctx context.Context) (uint64, error) {
	resp, err := n.clt.LastBlockHeight(ctx, &api.LastBlockHeightRequest{})
	if err != nil {
		n.log.Debug("could not get last block", logging.Error(err))
	}

	return resp.Height, nil
}
