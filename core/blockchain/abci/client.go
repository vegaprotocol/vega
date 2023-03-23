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
	"encoding/base64"
	"errors"
	"os"
	"sync"
	"time"

	cmtjson "github.com/tendermint/tendermint/libs/json"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmquery "github.com/tendermint/tendermint/libs/pubsub/query"
	tmclihttp "github.com/tendermint/tendermint/rpc/client/http"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

var ErrEmptyClientAddr = errors.New("abci client addr is empty in config")

type Client struct {
	tmclt      *tmclihttp.HTTP
	genesisDoc *cachedGenesisDoc
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
		tmclt:      clt,
		genesisDoc: newCachedGenesisDoc(),
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
	genDoc, err := c.genesisDoc.Get(ctx, c.tmclt)
	if err != nil {
		return time.Time{}, err
	}
	return genDoc.GenesisTime.UTC(), nil
}

// GetChainID retrieves the chainID from the blockchain.
func (c *Client) GetChainID(ctx context.Context) (chainID string, err error) {
	genDoc, err := c.genesisDoc.Get(ctx, c.tmclt)
	if err != nil {
		return "", err
	}
	return genDoc.ChainID, nil
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
	genDoc, err := c.genesisDoc.Get(ctx, c.tmclt)
	if err != nil {
		return nil, err
	}
	return genDoc, nil
}

func (c *Client) GenesisValidators(ctx context.Context) ([]*tmtypes.Validator, error) {
	gen, err := c.genesisDoc.Get(ctx, c.tmclt)
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

// Subscribe subscribes to any event matching query (https://godoc.org/github.com/tendermint/tendermint/types#pkg-constants).
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

type cachedGenesisDoc struct {
	mu           sync.Mutex
	genesisCache *tmtypes.GenesisDoc
}

func newCachedGenesisDoc() *cachedGenesisDoc {
	return &cachedGenesisDoc{}
}

func (c *cachedGenesisDoc) Get(
	ctx context.Context,
	clt interface {
		GenesisChunked(context.Context, uint) (*tmctypes.ResultGenesisChunk, error)
	},
) (*tmtypes.GenesisDoc, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.genesisCache == nil {
		var err error
		if c.genesisCache, err = c.cacheGenesis(ctx, clt); err != nil {
			return nil, err
		}
	}

	return c.genesisCache, nil
}

func (c *cachedGenesisDoc) cacheGenesis(
	ctx context.Context,
	clt interface {
		GenesisChunked(context.Context, uint) (*tmctypes.ResultGenesisChunk, error)
	},
) (*tmtypes.GenesisDoc, error) {
	var (
		res = &tmctypes.ResultGenesisChunk{
			TotalChunks: 1, // just default to startup our for loop
		}
		buf []byte
		err error
	)

	for i := 0; i < res.TotalChunks; i++ {
		res, err = clt.GenesisChunked(ctx, uint(i))
		if err != nil {
			return nil, err
		}

		decoded, err := base64.StdEncoding.DecodeString(res.Data)
		if err != nil {
			return nil, err
		}

		buf = append(buf, decoded...)
	}

	genDoc := types.GenesisDoc{}
	err = cmtjson.Unmarshal(buf, &genDoc)
	if err != nil {
		return nil, err
	}

	return &genDoc, nil
}
