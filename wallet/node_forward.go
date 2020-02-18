package wallet

import (
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto/api"

	"google.golang.org/grpc"
)

type nodeForward struct {
	nodeCfg NodeConfig
	clt     api.TradingClient
	conn    *grpc.ClientConn
}

func NewNodeForward(log *logging.Logger, nodeConfig NodeConfig) (*nodeForward, error) {
	nodeAddr := fmt.Sprintf("%v:%v", nodeConfig.IP, nodeConfig.Port)
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := api.NewTradingClient(conn)
	return &nodeForward{
		nodeCfg: nodeConfig,
		clt:     client,
		conn:    conn,
	}, nil
}

func (n *nodeForward) Send(tx *SignedBundle) error {
	return nil
}
