package blockchain

import (
	"context"
	"errors"
	"fmt"
	"time"

	types "code.vegaprotocol.io/vega/proto"

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
	Health() (*tmctypes.ResultHealth, error)
	SendTransaction(context.Context, []byte) (bool, error)
	GenesisValidators() ([]*tmtypes.Validator, error)
	Validators() ([]*tmtypes.Validator, error)
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

func (c *Client) SubmitTransaction(ctx context.Context, bundle *types.SignedBundle) (bool, error) {
	// unmarshal the transaction
	tx := &types.Transaction{}
	err := proto.Unmarshal(bundle.Tx, tx)
	if err != nil {
		return false, err
	}

	// first verify the transaction in the bundle is valid + signature is OK
	_, command, err := txDecode(tx.InputData)
	if err != nil {
		return false, err
	}

	// if the command is invalid, the String() func will return an empty string
	if command.String() == "" {
		// @TODO create err variable
		return false, fmt.Errorf("invalid command: %v", int(command))
	}

	// check sig
	if err := verifyBundle(nil, tx, bundle); err != nil {
		return false, err
	}

	// marshal the bundle then
	bundleBytes, err := proto.Marshal(bundle)
	if err != nil {
		return false, err
	}
	if len(bundleBytes) == 0 {
		return false, errors.New("order message empty after marshal")
	}

	return c.sendTx(ctx, bundleBytes)
}

func (c *Client) sendTx(ctx context.Context, bytes []byte) (bool, error) {
	return c.clt.SendTransaction(ctx, bytes)
}

// GetGenesisTime retrieves the genesis time from the blockchain
func (c *Client) GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error) {
	return c.clt.GetGenesisTime(ctx)
}

// GetChainID retrieves the chainID from the blockchain
func (c *Client) GetChainID(ctx context.Context) (chainID string, err error) {
	return c.clt.GetChainID(ctx)
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

func (c *Client) GenesisValidators() ([]*tmtypes.Validator, error) {
	return c.clt.GenesisValidators()
}
func (c *Client) Validators() ([]*tmtypes.Validator, error) {
	return c.clt.Validators()
}
