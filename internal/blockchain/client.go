package blockchain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
	uuid "github.com/satori/go.uuid"

	tmRPC "github.com/tendermint/tendermint/rpc/client"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type Client struct {
	*Config
	tmClient *tmRPC.HTTP
}

func NewClient(config *Config) (*Client, error) {
	if config.ClientAddr == "" {
		return nil, errors.New("abci client addr is empty in config")
	}
	if config.ClientEndpoint == "" {
		return nil, errors.New("abci client websocket endpoint is empty in config")
	}
	cli := tmRPC.NewHTTP(config.ClientAddr, config.ClientEndpoint)
	return &Client{Config: config, tmClient: cli}, nil
}

func (b *Client) CancelOrder(ctx context.Context, order *types.Order) (success bool, err error) {
	return b.sendOrderCommand(ctx, order, CancelOrderCommand)
}

func (b *Client) AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (success bool, err error) {
	return b.sendAmendmentCommand(ctx, amendment, AmendOrderCommand)
}

func (b *Client) NotifyTraderAccount(
	ctx context.Context, notif *types.NotifyTraderAccount) (success bool, err error) {
	return b.sendNotifyTraderAccountCommand(ctx, notif, NotifyTraderAccountCommand)
}

func (b *Client) CreateOrder(ctx context.Context, order *types.Order) (*types.PendingOrder, error) {
	order.Reference = fmt.Sprintf("%s", uuid.NewV4())
	_, err := b.sendOrderCommand(ctx, order, SubmitOrderCommand)

	if err != nil {
		return nil, err
	}

	return &types.PendingOrder{
		Reference:   order.Reference,
		Price:       order.Price,
		TimeInForce: order.TimeInForce,
		Side:        order.Side,
		MarketID:    order.MarketID,
		Size:        order.Size,
		PartyID:     order.PartyID,
		Status:      order.Status,
	}, nil
}

func (b *Client) GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error) {
	res, err := b.tmClient.Genesis()
	if err != nil {
		return vegatime.Now(), err
	}
	return res.Genesis.GenesisTime.UTC(), nil
}

func (b *Client) GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error) {
	return b.tmClient.Status()
}

func (b *Client) GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error) {
	return b.tmClient.NetInfo()
}

func (b *Client) GetUnconfirmedTxCount(ctx context.Context) (count int, err error) {
	res, err := b.tmClient.NumUnconfirmedTxs()
	if err != nil {
		return 0, err
	}
	return res.Count, err
}

func (b *Client) Health() (*tmctypes.ResultHealth, error) {
	return b.tmClient.Health()
}

func (b *Client) sendOrderCommand(ctx context.Context, order *types.Order, cmd Command) (success bool, err error) {

	// Proto-buf marshall the incoming order to byte slice.
	bytes, err := proto.Marshal(order)
	if err != nil {
		return false, err
	}
	if len(bytes) == 0 {
		return false, errors.New("order message empty after marshal")
	}

	return b.sendCommand(ctx, bytes, cmd)
}

func (b *Client) sendAmendmentCommand(ctx context.Context, amendment *types.OrderAmendment, cmd Command) (success bool, err error) {

	// Proto-buf marshall the incoming order to byte slice.
	bytes, err := proto.Marshal(amendment)
	if err != nil {
		return false, err
	}
	if len(bytes) == 0 {
		return false, errors.New("order message empty after marshal")
	}

	return b.sendCommand(ctx, bytes, cmd)
}

func (b *Client) sendNotifyTraderAccountCommand(
	ctx context.Context, notif *types.NotifyTraderAccount, cmd Command) (success bool, err error) {

	bytes, err := proto.Marshal(notif)
	if err != nil {
		return false, err
	}
	if len(bytes) == 0 {
		return false, errors.New("notify trader account message empty after marshal")
	}

	return b.sendCommand(ctx, bytes, cmd)
}

func (b *Client) sendCommand(ctx context.Context, bytes []byte, cmd Command) (success bool, err error) {

	// Tendermint requires unique transactions so we pre-pend a guid + pipe to the byte array.
	// It's split on arrival out of consensus along with a byte that represents command e.g. cancel order
	bytes, err = txEncode(bytes, cmd)
	if err != nil {
		return false, err
	}

	// Fire off the transaction for consensus
	_, err = b.tmClient.BroadcastTxAsync(bytes)
	if err != nil {
		return false, err
	}

	//b.log.Debug("BroadcastTxAsync response",
	//	logging.String("log", res.Log),
	//	logging.Uint32("code", res.Code),
	//	logging.String("data", string(res.Data)),
	//	logging.String("hash", string(res.Hash)))

	return true, nil
}

func txEncode(input []byte, cmd Command) (proto []byte, err error) {
	prefix := uuid.NewV4().String()
	prefixBytes := []byte(prefix)
	commandInput := append([]byte{byte(cmd)}, input...)
	return append(prefixBytes, commandInput...), nil
}
