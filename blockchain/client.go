package blockchain

import (
	"context"
	"errors"
	"time"

	"code.vegaprotocol.io/protos/commands"
	"code.vegaprotocol.io/protos/vega/api"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"github.com/golang/protobuf/proto"

	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type chainClientImpl interface {
	GetGenesisTime(context.Context) (time.Time, error)
	GetChainID(context.Context) (string, error)
	GetStatus(context.Context) (*tmctypes.ResultStatus, error)
	GetNetworkInfo(context.Context) (*tmctypes.ResultNetInfo, error)
	GetUnconfirmedTxCount(context.Context) (int, error)
	Health(context.Context) (*tmctypes.ResultHealth, error)
	SendTransactionAsync(context.Context, []byte) error
	SendTransactionSync(context.Context, []byte) error
	SendTransactionCommit(context.Context, []byte) error
	GenesisValidators(context.Context) ([]*tmtypes.Validator, error)
	Validators(context.Context) ([]*tmtypes.Validator, error)
	Subscribe(context.Context, func(tmctypes.ResultEvent) error, ...string) error
}

// Client abstract all communication to the blockchain
type Client struct {
	*Config
	clt chainClientImpl
}

// NewClient instantiate a new blockchain client
func NewClient(clt chainClientImpl) *Client {
	return &Client{
		clt: clt,
	}
}

func (c *Client) SubmitTransactionV2(ctx context.Context, tx *commandspb.Transaction, ty api.SubmitTransactionRequest_Type) error {
	_, err := commands.CheckTransaction(tx)
	if err != nil {
		return err
	}

	marshalledTx, err := proto.Marshal(tx)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.sendTxV2(ctx, marshalledTx, ty)
}

func (c *Client) sendTxV2(ctx context.Context, msg []byte, ty api.SubmitTransactionRequest_Type) error {
	switch ty {
	case api.SubmitTransactionRequest_TYPE_ASYNC:
		return c.clt.SendTransactionAsync(ctx, msg)
	case api.SubmitTransactionRequest_TYPE_SYNC:
		return c.clt.SendTransactionSync(ctx, msg)
	case api.SubmitTransactionRequest_TYPE_COMMIT:
		return c.clt.SendTransactionCommit(ctx, msg)
	default:
		return errors.New("invalid submit transaction request type")
	}
}

// GetGenesisTime retrieves the genesis time from the blockchain
func (c *Client) GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetGenesisTime(ctx)
}

// GetChainID retrieves the chainID from the blockchain
func (c *Client) GetChainID(ctx context.Context) (chainID string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetChainID(ctx)
}

// GetStatus returns the current status of the chain
func (c *Client) GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetStatus(ctx)
}

// GetNetworkInfo return information of the current network
func (c *Client) GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetNetworkInfo(ctx)
}

// GetUnconfirmedTxCount return the current count of unconfirmed transactions
func (c *Client) GetUnconfirmedTxCount(ctx context.Context) (count int, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetUnconfirmedTxCount(ctx)
}

// Health returns the result of the health endpoint of the chain
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

func (c *Client) Validators() ([]*tmtypes.Validator, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.clt.Validators(ctx)
}

func (c *Client) Subscribe(ctx context.Context, fn func(tmctypes.ResultEvent) error, queries ...string) error {
	return c.clt.Subscribe(ctx, fn, queries...)
}
