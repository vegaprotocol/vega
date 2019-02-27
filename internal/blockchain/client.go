package blockchain

import (
	"context"
	"errors"
	"fmt"
	"time"

	types "vega/proto"

	"github.com/golang/protobuf/proto"
	uuid "github.com/satori/go.uuid"

	"vega/internal/logging"

	tmRPC "github.com/tendermint/tendermint/rpc/client"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type Client interface {
	CreateOrder(ctx context.Context, order *types.Order) (success bool, orderReference string, err error)
	CancelOrder(ctx context.Context, order *types.Order) (success bool, err error)
	AmendOrder(ctx context.Context, amendment *types.Amendment) (success bool, err error)
	GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error)
	GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error) // todo(cdm) revisit after stats refactor
	GetUnconfirmedTxCount(ctx context.Context) (count int, err error)
	GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error)
}

type client struct {
	*Config
	tmClient *tmRPC.HTTP
}

func NewClient(config *Config) (Client, error) {
	if config.ClientAddr == "" {
		return nil, errors.New("abci client addr is empty in config")
	}
	if config.ClientEndpoint == "" {
		return nil, errors.New("abci client websocket endpoint is empty in config")
	}
	cli := tmRPC.NewHTTP(config.ClientAddr, config.ClientEndpoint)
	return &client{Config: config, tmClient: cli}, nil
}

func (b *client) CancelOrder(ctx context.Context, order *types.Order) (success bool, err error) {
	return b.sendOrderCommand(ctx, order, CancelOrderCommand)
}

func (b *client) AmendOrder(ctx context.Context, amendment *types.Amendment) (success bool, err error) {
	return b.sendAmendmentCommand(ctx, amendment, AmendOrderCommand)
}

func (b *client) CreateOrder(ctx context.Context, order *types.Order) (success bool, orderReference string, err error) {
	order.Reference = fmt.Sprintf("%s", uuid.NewV4())
	success, err = b.sendOrderCommand(ctx, order, SubmitOrderCommand)
	return success, order.Reference, err
}

func (b *client) GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error) {
	res, err := b.tmClient.Genesis()
	if err != nil {
		return time.Now(), err
	}
	return res.Genesis.GenesisTime, nil
}

func (b *client) GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error) {
	return b.tmClient.Status()
}

func (b *client) GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error) {
	return b.tmClient.NetInfo()
}

func (b *client) GetUnconfirmedTxCount(ctx context.Context) (count int, err error) {
	res, err := b.tmClient.NumUnconfirmedTxs()
	if err != nil {
		return 0, err
	}
	return res.N, err
}

func (b *client) sendOrderCommand(ctx context.Context, order *types.Order, cmd Command) (success bool, err error) {

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

func (b *client) sendAmendmentCommand(ctx context.Context, amendment *types.Amendment, cmd Command) (success bool, err error) {

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

func (b *client) sendCommand(ctx context.Context, bytes []byte, cmd Command) (success bool, err error) {

	// Tendermint requires unique transactions so we pre-pend a guid + pipe to the byte array.
	// It's split on arrival out of consensus along with a byte that represents command e.g. cancel order
	bytes, err = txEncode(bytes, cmd)
	if err != nil {
		return false, err
	}

	// Fire off the transaction for consensus
	res, err := b.tmClient.BroadcastTxAsync(bytes)
	if err != nil {
		return false, err
	}

	// todo remove this once investigation into tm client issues are complete
	b.log.Debug("BroadcastTxAsync response",
		logging.String("log", res.Log),
		logging.Uint32("code", res.Code),
		logging.String("data", string(res.Data)),
		logging.String("hash", string(res.Hash)))

	return true, nil
}

func txEncode(input []byte, cmd Command) (proto []byte, err error) {
	prefix := uuid.NewV4().String()
	prefixBytes := []byte(prefix)
	commandInput := append([]byte{byte(cmd)}, input...)
	return append(prefixBytes, commandInput...), nil
}
