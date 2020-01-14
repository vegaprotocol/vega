package blockchain

import (
	"context"
	"errors"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
	uuid "github.com/satori/go.uuid"

	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type chainClientImpl interface {
	GetGenesisTime(context.Context) (time.Time, error)
	GetStatus(context.Context) (*tmctypes.ResultStatus, error)
	GetNetworkInfo(context.Context) (*tmctypes.ResultNetInfo, error)
	GetUnconfirmedTxCount(context.Context) (int, error)
	Health() (*tmctypes.ResultHealth, error)
	SendTransaction(context.Context, []byte) (bool, error)
}

// Client abstract all communication to the blockchain
type Client struct {
	*Config
	clt chainClientImpl
}

// NewClient instantiate a new blockchain client
func newClient(clt chainClientImpl) *Client {
	return &Client{
		clt: clt,
	}
}

// CancelOrder will send a cancel order transaction to the blockchain
func (c *Client) CancelOrder(ctx context.Context, order *types.OrderCancellation) (success bool, err error) {
	return c.sendCancellationCommand(ctx, order, CancelOrderCommand)
}

// AmendOrder will send an amend order transaction to the blockchain
func (c *Client) AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (success bool, err error) {
	return c.sendAmendmentCommand(ctx, amendment, AmendOrderCommand)
}

// NotifyTraderAccount will send a Notifytraderaccount transaction to the blockchain
func (c *Client) NotifyTraderAccount(
	ctx context.Context, notif *types.NotifyTraderAccount) (success bool, err error) {
	return c.sendNotifyTraderAccountCommand(ctx, notif, NotifyTraderAccountCommand)
}

// Withdraw will send a Withdraw transaction to the blockchain
func (c *Client) Withdraw(ctx context.Context, w *types.Withdraw) (bool, error) {
	return c.sendWithdrawCommand(ctx, w, WithdrawCommand)
}

// CreateOrder will send a submit order transaction to the blockchain
func (c *Client) CreateOrder(ctx context.Context, order *types.Order) (*types.PendingOrder, error) {
	order.Reference = uuid.NewV4().String()
	_, err := c.sendOrderCommand(ctx, order, SubmitOrderCommand)

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
		Type:        order.Type,
	}, nil
}

// GetGenesisTime will retrieve the genesis time from the blockchain
func (c *Client) GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error) {
	return c.clt.GetGenesisTime(ctx)
}

// GetStatus returns the current status of the chain
func (c *Client) GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error) {
	return c.clt.GetStatus(ctx)
}

// GetNetworkInfo return information of the current network
func (c *Client) GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error) {
	return c.clt.GetNetworkInfo(ctx)
}

// GetUnconfirmedTxCount return the current count of unconfirmed transactions
func (c *Client) GetUnconfirmedTxCount(ctx context.Context) (count int, err error) {
	return c.clt.GetUnconfirmedTxCount(ctx)
}

// Health returns the result of the health endpoint of the chain
func (c *Client) Health() (*tmctypes.ResultHealth, error) {
	return c.clt.Health()
}

func (c *Client) sendOrderCommand(ctx context.Context, order *types.Order, cmd Command) (success bool, err error) {

	// Proto-buf marshall the incoming order to byte slice.
	bytes, err := proto.Marshal(order)
	if err != nil {
		return false, err
	}
	if len(bytes) == 0 {
		return false, errors.New("order message empty after marshal")
	}

	return c.sendCommand(ctx, bytes, cmd)
}

func (c *Client) sendAmendmentCommand(ctx context.Context, amendment *types.OrderAmendment, cmd Command) (success bool, err error) {

	// Proto-buf marshall the incoming order to byte slice.
	bytes, err := proto.Marshal(amendment)
	if err != nil {
		return false, err
	}
	if len(bytes) == 0 {
		return false, errors.New("order message empty after marshal")
	}

	return c.sendCommand(ctx, bytes, cmd)
}

func (c *Client) sendCancellationCommand(ctx context.Context, cancel *types.OrderCancellation, cmd Command) (success bool, err error) {

	// Proto-buf marshall the incoming order to byte slice.
	bytes, err := proto.Marshal(cancel)
	if err != nil {
		return false, err
	}
	if len(bytes) == 0 {
		return false, errors.New("order message empty after marshal")
	}

	return c.sendCommand(ctx, bytes, cmd)
}

func (c *Client) sendNotifyTraderAccountCommand(
	ctx context.Context, notif *types.NotifyTraderAccount, cmd Command) (success bool, err error) {

	bytes, err := proto.Marshal(notif)
	if err != nil {
		return false, err
	}
	if len(bytes) == 0 {
		return false, errors.New("notify trader account message empty after marshal")
	}

	return c.sendCommand(ctx, bytes, cmd)
}

func (c *Client) sendWithdrawCommand(
	ctx context.Context, w *types.Withdraw, cmd Command) (success bool, err error) {

	bytes, err := proto.Marshal(w)
	if err != nil {
		return false, err
	}
	if len(bytes) == 0 {
		return false, errors.New("withdraw message empty after marshal")
	}

	return c.sendCommand(ctx, bytes, cmd)
}

func (c *Client) sendCommand(ctx context.Context, bytes []byte, cmd Command) (success bool, err error) {
	// Tendermint requires unique transactions so we pre-pend a guid + pipe to the byte array.
	// It's split on arrival out of consensus along with a byte that represents command e.g. cancel order
	bytes, err = txEncode(bytes, cmd)
	if err != nil {
		return false, err
	}

	// Fire off the transaction for consensus
	return c.clt.SendTransaction(ctx, bytes)
}

func txEncode(input []byte, cmd Command) (proto []byte, err error) {
	prefix := uuid.NewV4().String()
	prefixBytes := []byte(prefix)
	commandInput := append([]byte{byte(cmd)}, input...)
	return append(prefixBytes, commandInput...), nil
}
