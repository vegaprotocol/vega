package blockchain

import (
	"context"
	"time"

	"code.vegaprotocol.io/protos/commands"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"github.com/golang/protobuf/proto"

	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type ChainClientImpl interface {
	GetGenesisTime(context.Context) (time.Time, error)
	GetChainID(context.Context) (string, error)
	GetStatus(context.Context) (*tmctypes.ResultStatus, error)
	GetNetworkInfo(context.Context) (*tmctypes.ResultNetInfo, error)
	GetUnconfirmedTxCount(context.Context) (int, error)
	Health(context.Context) (*tmctypes.ResultHealth, error)
	SendTransactionAsync(context.Context, []byte) (*tmctypes.ResultBroadcastTx, error)
	SendTransactionSync(context.Context, []byte) (*tmctypes.ResultBroadcastTx, error)
	CheckTransaction(context.Context, []byte) (*tmctypes.ResultCheckTx, error)
	SendTransactionCommit(context.Context, []byte) (*tmctypes.ResultBroadcastTxCommit, error)
	GenesisValidators(context.Context) ([]*tmtypes.Validator, error)
	Validators(context.Context, *int64) ([]*tmtypes.Validator, error)
	Subscribe(context.Context, func(tmctypes.ResultEvent) error, ...string) error
	Start() error
}

// Client abstract all communication to the blockchain.
type Client struct {
	*Config
	clt ChainClientImpl
}

// NewClient instantiate a new blockchain client.
func NewClient(clt ChainClientImpl) *Client {
	return &Client{
		clt: clt,
	}
}

func (c *Client) CheckRawTransaction(ctx context.Context, tx []byte) (*tmctypes.ResultCheckTx, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.CheckTransaction(ctx, tx)
}

func (c *Client) CheckTransaction(ctx context.Context, tx *commandspb.Transaction) (*tmctypes.ResultCheckTx, error) {
	_, err := commands.CheckTransaction(tx)
	if err != nil {
		return nil, err
	}

	marshalledTx, err := proto.Marshal(tx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.CheckTransaction(ctx, marshalledTx)
}

func (c *Client) SubmitTransactionSync(ctx context.Context, tx *commandspb.Transaction) (*tmctypes.ResultBroadcastTx, error) {
	_, err := commands.CheckTransaction(tx)
	if err != nil {
		return nil, err
	}

	marshalledTx, err := proto.Marshal(tx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	t, err := c.clt.SendTransactionSync(ctx, marshalledTx)

	return t, err
}

func (c *Client) SubmitTransactionCommit(ctx context.Context, tx *commandspb.Transaction) (*tmctypes.ResultBroadcastTxCommit, error) {
	_, err := commands.CheckTransaction(tx)
	if err != nil {
		return nil, err
	}

	marshalledTx, err := proto.Marshal(tx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	t, err := c.clt.SendTransactionCommit(ctx, marshalledTx)

	return t, err
}

func (c *Client) SubmitTransactionAsync(ctx context.Context, tx *commandspb.Transaction) (*tmctypes.ResultBroadcastTx, error) {
	_, err := commands.CheckTransaction(tx)
	if err != nil {
		return nil, err
	}

	marshalledTx, err := proto.Marshal(tx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	t, err := c.clt.SendTransactionAsync(ctx, marshalledTx)

	return t, err
}

func (c *Client) SubmitRawTransactionSync(ctx context.Context, tx []byte) (*tmctypes.ResultBroadcastTx, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.SendTransactionSync(timeoutCtx, tx)
}

func (c *Client) SubmitRawTransactionAsync(ctx context.Context, tx []byte) (*tmctypes.ResultBroadcastTx, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.SendTransactionAsync(timeoutCtx, tx)
}

func (c *Client) SubmitRawTransactionCommit(ctx context.Context, tx []byte) (*tmctypes.ResultBroadcastTxCommit, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.SendTransactionCommit(timeoutCtx, tx)
}

// GetGenesisTime retrieves the genesis time from the blockchain.
func (c *Client) GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetGenesisTime(ctx)
}

// GetChainID retrieves the chainID from the blockchain.
func (c *Client) GetChainID(ctx context.Context) (chainID string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetChainID(ctx)
}

// GetStatus returns the current status of the chain.
func (c *Client) GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetStatus(ctx)
}

// GetNetworkInfo return information of the current network.
func (c *Client) GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetNetworkInfo(ctx)
}

// GetUnconfirmedTxCount return the current count of unconfirmed transactions.
func (c *Client) GetUnconfirmedTxCount(ctx context.Context) (count int, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetUnconfirmedTxCount(ctx)
}

// Health returns the result of the health endpoint of the chain.
func (c *Client) Health() (*tmctypes.ResultHealth, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.clt.Health(ctx)
}

func (c *Client) GenesisValidators() ([]*tmtypes.Validator, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.clt.GenesisValidators(ctx)
}

func (c *Client) Validators(height *int64) ([]*tmtypes.Validator, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.clt.Validators(ctx, height)
}

func (c *Client) Subscribe(ctx context.Context, fn func(tmctypes.ResultEvent) error, queries ...string) error {
	return c.clt.Subscribe(ctx, fn, queries...)
}

func (c *Client) Start() error {
	return c.clt.Start()
}
