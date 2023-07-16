// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package blockchain

import (
	"context"
	"errors"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/proto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/cometbft/cometbft/libs/bytes"
	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
)

var ErrClientNotReady = errors.New("tendermint client is not ready")

// nolint: interfacebloat
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
	clt         ChainClientImpl
	mempoolSize int64
	mu          sync.RWMutex
}

// NewClient instantiate a new blockchain client.
func NewClient() *Client {
	return &Client{
		clt: nil,
	}
}

func NewClientWithImpl(clt ChainClientImpl) *Client {
	return &Client{
		clt: clt,
	}
}

func (c *Client) Set(clt ChainClientImpl, mempoolSize int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clt = clt
	c.mempoolSize = mempoolSize
}

func (c *Client) isReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.clt != nil
}

func (c *Client) CheckRawTransaction(ctx context.Context, tx []byte) (*tmctypes.ResultCheckTx, error) {
	if !c.isReady() {
		return nil, ErrClientNotReady
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.CheckTransaction(ctx, tx)
}

func (c *Client) CheckTransaction(ctx context.Context, tx *commandspb.Transaction) (*tmctypes.ResultCheckTx, error) {
	if !c.isReady() {
		return nil, ErrClientNotReady
	}

	chainID, err := c.clt.GetChainID(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := commands.CheckTransaction(tx, chainID); err != nil {
		return &tmctypes.ResultCheckTx{
			ResponseCheckTx: *NewResponseCheckTxError(AbciTxnDecodingFailure, err),
		}, nil
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
	if !c.isReady() {
		return nil, ErrClientNotReady
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
	if !c.isReady() {
		return nil, ErrClientNotReady
	}

	chainID, err := c.clt.GetChainID(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := commands.CheckTransaction(tx, chainID); err != nil {
		return &tmctypes.ResultBroadcastTxCommit{
			CheckTx: *NewResponseCheckTxError(AbciTxnDecodingFailure, err),
		}, nil
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
	if !c.isReady() {
		return nil, ErrClientNotReady
	}

	chainID, err := c.clt.GetChainID(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := commands.CheckTransaction(tx, chainID); err != nil {
		return &tmctypes.ResultBroadcastTx{ //nolint:nilerr
			Code: AbciTxnDecodingFailure,
			Data: bytes.HexBytes(err.Error()),
		}, nil
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
	if !c.isReady() {
		return nil, ErrClientNotReady
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.SendTransactionSync(timeoutCtx, tx)
}

func (c *Client) SubmitRawTransactionAsync(ctx context.Context, tx []byte) (*tmctypes.ResultBroadcastTx, error) {
	if !c.isReady() {
		return nil, ErrClientNotReady
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.SendTransactionAsync(timeoutCtx, tx)
}

func (c *Client) SubmitRawTransactionCommit(ctx context.Context, tx []byte) (*tmctypes.ResultBroadcastTxCommit, error) {
	if !c.isReady() {
		return nil, ErrClientNotReady
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.SendTransactionCommit(timeoutCtx, tx)
}

// GetGenesisTime retrieves the genesis time from the blockchain.
func (c *Client) GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error) {
	if !c.isReady() {
		return time.Time{}, ErrClientNotReady
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetGenesisTime(ctx)
}

// GetChainID retrieves the chainID from the blockchain.
func (c *Client) GetChainID(ctx context.Context) (chainID string, err error) {
	if !c.isReady() {
		return "", ErrClientNotReady
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetChainID(ctx)
}

// GetStatus returns the current status of the chain.
func (c *Client) GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error) {
	if !c.isReady() {
		return nil, ErrClientNotReady
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetStatus(ctx)
}

// GetNetworkInfo return information of the current network.
func (c *Client) GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error) {
	if !c.isReady() {
		return nil, ErrClientNotReady
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetNetworkInfo(ctx)
}

// GetUnconfirmedTxCount return the current count of unconfirmed transactions.
func (c *Client) GetUnconfirmedTxCount(ctx context.Context) (count int, err error) {
	if !c.isReady() {
		return 0, ErrClientNotReady
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.clt.GetUnconfirmedTxCount(ctx)
}

// Health returns the result of the health endpoint of the chain.
func (c *Client) Health() (*tmctypes.ResultHealth, error) {
	if !c.isReady() {
		return nil, ErrClientNotReady
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.clt.Health(ctx)
}

func (c *Client) GenesisValidators() ([]*tmtypes.Validator, error) {
	if !c.isReady() {
		return nil, ErrClientNotReady
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.clt.GenesisValidators(ctx)
}

func (c *Client) MaxMempoolSize() int64 {
	return c.mempoolSize
}

func (c *Client) Validators(height *int64) ([]*tmtypes.Validator, error) {
	if !c.isReady() {
		return nil, ErrClientNotReady
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.clt.Validators(ctx, height)
}

func (c *Client) Subscribe(ctx context.Context, fn func(tmctypes.ResultEvent) error, queries ...string) error {
	if !c.isReady() {
		return ErrClientNotReady
	}
	return c.clt.Subscribe(ctx, fn, queries...)
}

func (c *Client) Start() error {
	return c.clt.Start()
}
