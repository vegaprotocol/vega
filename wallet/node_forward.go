package wallet

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto/gen/golang/api"

	"github.com/cenkalti/backoff/v4"
	"google.golang.org/grpc"
)

type nodeForward struct {
	log     *logging.Logger
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
		log:     log,
		nodeCfg: nodeConfig,
		clt:     client,
		conn:    conn,
	}, nil
}

func (n *nodeForward) Stop() error {
	n.log.Info("closing grpc client", logging.String("address", fmt.Sprintf("%v:%v", n.nodeCfg.IP, n.nodeCfg.Port)))
	return n.conn.Close()
}

func (n *nodeForward) Send(ctx context.Context, tx *SignedBundle, ty api.SubmitTransactionRequest_Type) error {
	req := api.SubmitTransactionRequest{
		Tx:   tx.IntoProto(),
		Type: ty,
	}
	return backoff.Retry(
		func() error {
			resp, err := n.clt.SubmitTransaction(ctx, &req)
			if err != nil {
				return err
			}
			n.log.Debug("response from SubmitTransaction", logging.Bool("success", resp.Success))
			return nil
		},
		backoff.WithMaxRetries(backoff.NewExponentialBackOff(), n.nodeCfg.Retries),
	)
}
