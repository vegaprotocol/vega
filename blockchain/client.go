package blockchain

import (
	"sync"
	"context"
	"errors"

	"vega/msg"
	"vega/tendermint/rpc"

	"github.com/golang/protobuf/proto"
)

type Client interface
{
	CreateOrder(ctx context.Context, order *msg.Order) (success bool, err error)
	CancelOrder(ctx context.Context, order *msg.Order) (success bool, err error)
}

type client struct {
	rpcClients   []*rpc.Client
	rpcClientMux sync.Mutex
}

func NewClient() Client {
	return &client{}
}

func (b *client) CancelOrder(ctx context.Context, order *msg.Order) (success bool, err error) {
	return b.sendOrderCommand(ctx, order, CancelOrderCommand)
}

func (b *client) CreateOrder(ctx context.Context, order *msg.Order) (success bool, err error) {
	return b.sendOrderCommand(ctx, order, CreateOrderCommand)
}

func (b *client) getRpcClient() (*rpc.Client, error) {
	b.rpcClientMux.Lock()
	if len(b.rpcClients) == 0 {
		b.rpcClientMux.Unlock()
		client := rpc.Client{
		}
		if err := client.Connect(); err != nil {
			return nil, err
		}
		return &client, nil
	}
	client := b.rpcClients[0]
	b.rpcClients = b.rpcClients[1:]
	b.rpcClientMux.Unlock()
	return client, nil
}

func (b *client) releaseRpcClient(c *rpc.Client) {
	b.rpcClientMux.Lock()
	b.rpcClients = append(b.rpcClients, c)
	b.rpcClientMux.Unlock()
}

func (b *client) sendOrderCommand(ctx context.Context, order *msg.Order, cmd Command) (success bool, err error) {
	// Protobuf marshall the incoming order to byte slice.
	bytes, err := proto.Marshal(order)
	if err != nil {
		return false, err
	}
	if len(bytes) == 0 {
		return false, errors.New("order message empty after marshal")
	}

	// Tendermint requires unique transactions so we pre-pend a guid + pipe to the byte array.
	// It's split on arrival out of consensus along with a byte that represents command e.g. cancel order
	bytes, err = VegaTxEncode(bytes, cmd)
	if err != nil {
		return false, err
	}

	// Get a lightweight RPC client (our custom Tendermint client) from a pool (create one if n/a).
	client, err := b.getRpcClient()
	if err != nil {
		return false, err
	}

	// Fire off the transaction for consensus
	err = client.AsyncTransaction(ctx, bytes)
	if err != nil {
		if !client.HasError() {
			b.releaseRpcClient(client)
		}
		return false, err
	}

	// If all went well we return the client to the pool for another caller.
	if client != nil {
		b.releaseRpcClient(client)
	}
	return true, nil
}

