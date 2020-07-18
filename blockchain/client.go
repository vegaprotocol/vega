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

// FIXME(): remove once we have only signed transaction going through the system
// NotifyTraderAccount will send a Notifytraderaccount transaction to the blockchain
func (c *Client) NotifyTraderAccount(
	ctx context.Context, notif *types.NotifyTraderAccount) (success bool, err error) {
	bytes, err := proto.Marshal(notif)
	if err != nil {
		return false, err
	}
	if len(bytes) == 0 {
		return false, errors.New("notify trader account message empty after marshal")
	}

	return c.sendCommand(ctx, bytes, NotifyTraderAccountCommand)

}

// FIXME(): remove once we have only signed transaction going through the system
// Withdraw will send a Withdraw transaction to the blockchain
func (c *Client) Withdraw(ctx context.Context, w *types.Withdraw) (bool, error) {
	bytes, err := proto.Marshal(w)
	if err != nil {
		return false, err
	}
	if len(bytes) == 0 {
		return false, errors.New("withdraw message empty after marshal")
	}

	return c.sendCommand(ctx, bytes, WithdrawCommand)
}

// FIXME(): remove once we have only signed transaction going through the system
func (c *Client) sendCommand(ctx context.Context, bytes []byte, cmd Command) (success bool, err error) {
	// Tendermint requires unique transactions so we pre-pend a guid + pipe to the byte array.
	// It's split on arrival out of consensus along with a byte that represents command e.g. cancel order
	bytes, err = txEncode(bytes, cmd)
	if err != nil {
		return false, err
	}

	// make it a empty transaction
	// no nonce or pubkey here
	tx := &types.Transaction{InputData: bytes}
	rawTx, err := proto.Marshal(tx)
	if err != nil {
		return false, err
	}

	bundle := &types.SignedBundle{
		Tx:  rawTx,
		Sig: &types.Signature{}, // end an empty sig
	}
	rawBundle, err := proto.Marshal(bundle)
	if err != nil {
		return false, err
	}

	// Fire off the transaction for consensus
	return c.sendTx(ctx, rawBundle)
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
