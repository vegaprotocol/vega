package wallet

import (
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto/api"

	"google.golang.org/grpc"
)

type nodeForward struct {
	nodeAddr string
	clt      api.TradingClient
	conn     *grpc.ClientConn
}

func NewNodeForward(log *logging.Logger, nodeAddr string) (*nodeForward, error) {
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := api.NewTradingClient(conn)
	return &nodeForward{
		nodeAddr: nodeAddr,
		clt:      client,
		conn:     conn,
	}, nil
}

func (n *nodeForward) SendTx(tx *SignedBundle) error {
	return nil
}
