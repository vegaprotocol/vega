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

package abci

import (
	"context"
	"errors"
	"os"
	"time"

	tmlog "github.com/cometbft/cometbft/libs/log"
	tmquery "github.com/cometbft/cometbft/libs/pubsub/query"
	tmclihttp "github.com/cometbft/cometbft/rpc/client/http"
	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
)

var ErrEmptyClientAddr = errors.New("abci client addr is empty in config")

type Client struct {
	tmclt *tmclihttp.HTTP
}

func NewClient(addr string) (*Client, error) {
	if len(addr) <= 0 {
		return nil, ErrEmptyClientAddr
	}

	clt, err := tmclihttp.New(addr, "/websocket")
	if err != nil {
		return nil, err
	}

	// log errors only
	clt.Logger = tmlog.NewFilter(
		tmlog.NewTMLogger(os.Stdout),
		tmlog.AllowError(),
	)

	return &Client{
		tmclt: clt,
	}, nil
}

func (c *Client) SendTransactionAsync(ctx context.Context, bytes []byte) (*tmctypes.ResultBroadcastTx, error) {
	// Fire off the transaction for consensus
	return c.tmclt.BroadcastTxAsync(ctx, bytes)
}

func (c *Client) CheckTransaction(ctx context.Context, bytes []byte) (*tmctypes.ResultCheckTx, error) {
	return c.tmclt.CheckTx(ctx, bytes)
}

func (c *Client) SendTransactionSync(ctx context.Context, bytes []byte) (*tmctypes.ResultBroadcastTx, error) {
	// Fire off the transaction for consensus
	return c.tmclt.BroadcastTxSync(ctx, bytes)
}

func (c *Client) SendTransactionCommit(ctx context.Context, bytes []byte) (*tmctypes.ResultBroadcastTxCommit, error) {
	// Fire off the transaction for consensus
	return c.tmclt.BroadcastTxCommit(ctx, bytes)
}

// GetGenesisTime retrieves the genesis time from the blockchain.
func (c *Client) GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error) {
	res, err := c.tmclt.Genesis(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return res.Genesis.GenesisTime.UTC(), nil
}

// GetChainID retrieves the chainID from the blockchain.
func (c *Client) GetChainID(ctx context.Context) (chainID string, err error) {
	res, err := c.tmclt.Genesis(ctx)
	if err != nil {
		return "", err
	}
	return res.Genesis.ChainID, nil
}

// GetStatus returns the current status of the chain.
func (c *Client) GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error) {
	return c.tmclt.Status(ctx)
}

// GetNetworkInfo return information of the current network.
func (c *Client) GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error) {
	return c.tmclt.NetInfo(ctx)
}

// GetUnconfirmedTxCount return the current count of unconfirmed transactions.
func (c *Client) GetUnconfirmedTxCount(ctx context.Context) (count int, err error) {
	res, err := c.tmclt.NumUnconfirmedTxs(ctx)
	if err != nil {
		return 0, err
	}
	return res.Count, err
}

// Health returns the result of the health endpoint of the chain.
func (c *Client) Health(ctx context.Context) (*tmctypes.ResultHealth, error) {
	return c.tmclt.Health(ctx)
}

func (c *Client) Validators(ctx context.Context, height *int64) ([]*tmtypes.Validator, error) {
	res, err := c.tmclt.Validators(ctx, height, nil, nil)
	if err != nil {
		return nil, err
	}
	return res.Validators, nil
}

func (c *Client) Genesis(ctx context.Context) (*tmtypes.GenesisDoc, error) {
	res, err := c.tmclt.Genesis(ctx)
	if err != nil {
		return nil, err
	}
	return res.Genesis, nil
}

func (c *Client) GenesisValidators(ctx context.Context) ([]*tmtypes.Validator, error) {
	gen, err := c.Genesis(ctx)
	if err != nil {
		return nil, err
	}

	validators := make([]*tmtypes.Validator, 0, len(gen.Validators))
	for _, v := range gen.Validators {
		validators = append(validators, &tmtypes.Validator{
			Address:     v.Address,
			PubKey:      v.PubKey,
			VotingPower: v.Power,
		})
	}

	return validators, nil
}

// Subscribe subscribes to any event matching query (https://godoc.org/github.com/cometbft/cometbft/types#pkg-constants).
// Subscribe will call fn each time it receives an event from the node.
// The function returns nil when the context is canceled or when fn returns an error.
func (c *Client) Subscribe(ctx context.Context, fn func(tmctypes.ResultEvent) error, queries ...string) error {
	if err := c.tmclt.Start(); err != nil {
		return err
	}
	defer c.tmclt.Stop()

	errCh := make(chan error)

	for _, query := range queries {
		q, err := tmquery.New(query)
		if err != nil {
			return err
		}

		// For subscription we use "vega" as the client name but it's ignored by the implementation.
		// 10 is the channel capacity which is absolutely arbitraty.
		out, err := c.tmclt.Subscribe(ctx, "vega", q.String(), 10)
		if err != nil {
			return err
		}

		go func() {
			for res := range out {
				if err := fn(res); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	defer c.tmclt.UnsubscribeAll(context.Background(), "vega")

	return <-errCh
}

func (c *Client) Start() error {
	return nil // Nothing to do for this client type.
}
