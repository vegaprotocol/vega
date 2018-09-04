package blockchain

import (
	"context"
	"errors"
	"sync"

	"vega/msg"
	"vega/tendermint/rpc"

	"github.com/golang/protobuf/proto"
	"github.com/satori/go.uuid"
	"fmt"
)

type Client interface {
	CreateOrder(ctx context.Context, order *msg.Order) (success bool, orderReference string, err error)
	CancelOrder(ctx context.Context, order *msg.Order) (success bool, err error)
	EditOrder(ctx context.Context, order *msg.Order) (success bool, err error)
	GetGenesisTime(ctx context.Context) (genesis *rpc.Genesis, err error)
	GetStatus(ctx context.Context) (status *rpc.Status, err error)
	GetUnconfirmedTxCount(ctx context.Context) (count int, err error)
	GetNetworkInfo(ctx context.Context) (netInfo *rpc.NetInfo, err error)
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

func (b *client) EditOrder(ctx context.Context, order *msg.Order) (success bool, err error) {
	return b.sendOrderCommand(ctx, order, EditOrderCommand)
}

func (b *client) CreateOrder(ctx context.Context, order *msg.Order) (success bool, orderReference string, err error) {
	order.Reference = fmt.Sprintf("%s", uuid.NewV4())
	success, err = b.sendOrderCommand(ctx, order, CreateOrderCommand)
	return success, order.Reference, err
}

func (b *client) GetGenesisTime(ctx context.Context) (genesis *rpc.Genesis, err error) {
	client, err := b.getRpcClient()
	if err != nil {
		return nil, err
	}
	genesis, err = client.Genesis(ctx)
	if genesis == nil && err != nil {
		if !client.HasError() {
			b.releaseRpcClient(client)
		}
		return nil, err
	}
	if client != nil {
		b.releaseRpcClient(client)
	}
	return genesis, nil
}

func (b *client) GetStatus(ctx context.Context) (status *rpc.Status, err error) {
	client, err := b.getRpcClient()
	if err != nil {
		return nil, err
	}
	status, err = client.Status(ctx)
	if status == nil && err != nil {
		if !client.HasError() {
			b.releaseRpcClient(client)
		}
		return nil, err
	}
	if client != nil {
		b.releaseRpcClient(client)
	}
	return status, nil
}

func (b *client) GetNetworkInfo(ctx context.Context) (netInfo *rpc.NetInfo, err error) {
	client, err := b.getRpcClient()
	if err != nil {
		return nil, err
	}
	netInfo, err = client.NetInfo(ctx)
	if err != nil {
		if !client.HasError() {
			b.releaseRpcClient(client)
		}
		return nil, err
	}
	if client != nil {
		b.releaseRpcClient(client)
	}
	return netInfo, nil

}

func (b *client) GetUnconfirmedTxCount(ctx context.Context) (count int, err error) {
	client, err := b.getRpcClient()
	if err != nil {
		return 0, err
	}
	count, err = client.UnconfirmedTransactionsCount(ctx)
	if err != nil {
		if !client.HasError() {
			b.releaseRpcClient(client)
		}
		return 0, err
	}
	if client != nil {
		b.releaseRpcClient(client)
	}
	return count, nil
}

func (b *client) getRpcClient() (*rpc.Client, error) {
	b.rpcClientMux.Lock()
	if len(b.rpcClients) == 0 {
		b.rpcClientMux.Unlock()
		client := rpc.Client{}
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

	// assigne reference number, add it to the order, send to consensus and also return to client
	return true, nil
}
