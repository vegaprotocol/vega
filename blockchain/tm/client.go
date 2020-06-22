package tm

import (
	"context"
	"errors"
	"time"

	"code.vegaprotocol.io/vega/vegatime"

	tmclihttp "github.com/tendermint/tendermint/rpc/client/http"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

var (
	ErrEmptyClientAddr     = errors.New("abci client addr is empty in config")
	ErrEmptyClientEndpoint = errors.New("abci client websocket endpoint is empty in config")
)

type Client struct {
	tmclt *tmclihttp.HTTP
}

func NewClient(cfg Config) (*Client, error) {
	if len(cfg.ClientAddr) <= 0 {
		return nil, ErrEmptyClientAddr
	}
	if len(cfg.ClientEndpoint) <= 0 {
		return nil, ErrEmptyClientEndpoint
	}
	clt, err := tmclihttp.New(cfg.ClientAddr, cfg.ClientEndpoint)
	if err != nil {
		return nil, err
	}
	return &Client{
		tmclt: clt,
	}, nil
}

func (c *Client) SendTransaction(ctx context.Context, bytes []byte) (bool, error) {
	// Fire off the transaction for consensus
	_, err := c.tmclt.BroadcastTxAsync(bytes)
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetGenesisTime retrieves the genesis time from the blockchain
func (c *Client) GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error) {
	res, err := c.tmclt.Genesis()
	if err != nil {
		return vegatime.Now(), err
	}
	return res.Genesis.GenesisTime.UTC(), nil
}

// GetChainID retrieves the chainID from the blockchain
func (c *Client) GetChainID(ctx context.Context) (chainID string, err error) {
	res, err := c.tmclt.Genesis()
	if err != nil {
		return "", err
	}
	return res.Genesis.ChainID, nil
}

// GetStatus returns the current status of the chain
func (c *Client) GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error) {
	return c.tmclt.Status()
}

// GetNetworkInfo return information of the current network
func (c *Client) GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error) {
	return c.tmclt.NetInfo()
}

// GetUnconfirmedTxCount return the current count of unconfirmed transactions
func (c *Client) GetUnconfirmedTxCount(ctx context.Context) (count int, err error) {
	res, err := c.tmclt.NumUnconfirmedTxs()
	if err != nil {
		return 0, err
	}
	return res.Count, err
}

// Health returns the result of the health endpoint of the chain
func (c *Client) Health() (*tmctypes.ResultHealth, error) {
	return c.tmclt.Health()
}

func (c *Client) Validators() ([]*tmtypes.Validator, error) {
	res, err := c.tmclt.Validators(nil, 0, 100)
	if err != nil {
		return nil, err
	}
	return res.Validators, nil
}

func (c *Client) Genesis() (*tmtypes.GenesisDoc, error) {
	res, err := c.tmclt.Genesis()
	if err != nil {
		return nil, err
	}
	return res.Genesis, nil
}

func (c *Client) GenesisValidators() ([]*tmtypes.Validator, error) {
	gen, err := c.Genesis()
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
