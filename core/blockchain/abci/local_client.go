package abci

import (
	"context"
	"time"

	tmquery "github.com/cometbft/cometbft/libs/pubsub/query"
	"github.com/cometbft/cometbft/libs/service"
	nm "github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/rpc/client/local"
	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
)

type LocalClient struct {
	node *local.Local
}

func newLocalClient(node service.Service) (*LocalClient, error) {
	localNode := local.New(node.(*nm.Node))
	return &LocalClient{
		node: localNode,
	}, nil
}

func (c *LocalClient) SendTransactionAsync(ctx context.Context, bytes []byte) (*tmctypes.ResultBroadcastTx, error) {
	// Fire off the transaction for consensus
	res, err := c.node.BroadcastTxAsync(ctx, bytes)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *LocalClient) SendTransactionSync(ctx context.Context, bytes []byte) (*tmctypes.ResultBroadcastTx, error) {
	// Fire off the transaction for consensus
	return c.node.BroadcastTxSync(ctx, bytes)
}

func (c *LocalClient) SendTransactionCommit(ctx context.Context, bytes []byte) (*tmctypes.ResultBroadcastTxCommit, error) {
	// Fire off the transaction for consensus
	return c.node.BroadcastTxCommit(ctx, bytes)
}

func (c *LocalClient) CheckTransaction(ctx context.Context, bytes []byte) (*tmctypes.ResultCheckTx, error) {
	return c.node.CheckTx(ctx, bytes)
}

// GetGenesisTime retrieves the genesis time from the blockchain.
func (c *LocalClient) GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error) {
	res, err := c.node.Genesis(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return res.Genesis.GenesisTime.UTC(), nil
}

// GetChainID retrieves the chainID from the blockchain.
func (c *LocalClient) GetChainID(ctx context.Context) (chainID string, err error) {
	res, err := c.node.Genesis(ctx)
	if err != nil {
		return "", err
	}
	return res.Genesis.ChainID, nil
}

// GetStatus returns the current status of the chain.
func (c *LocalClient) GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error) {
	return c.node.Status(ctx)
}

// GetNetworkInfo return information of the current network.
func (c *LocalClient) GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error) {
	return c.node.NetInfo(ctx)
}

// GetUnconfirmedTxCount return the current count of unconfirmed transactions.
func (c *LocalClient) GetUnconfirmedTxCount(ctx context.Context) (count int, err error) {
	res, err := c.node.NumUnconfirmedTxs(ctx)
	if err != nil {
		return 0, err
	}
	return res.Count, err
}

// Health returns the result of the health endpoint of the chain.
func (c *LocalClient) Health(ctx context.Context) (*tmctypes.ResultHealth, error) {
	return c.node.Health(ctx)
}

func (c *LocalClient) Validators(ctx context.Context, height *int64) ([]*tmtypes.Validator, error) {
	res, err := c.node.Validators(ctx, height, nil, nil)
	if err != nil {
		return nil, err
	}
	return res.Validators, nil
}

func (c *LocalClient) Genesis(ctx context.Context) (*tmtypes.GenesisDoc, error) {
	res, err := c.node.Genesis(ctx)
	if err != nil {
		return nil, err
	}
	return res.Genesis, nil
}

func (c *LocalClient) GenesisValidators(ctx context.Context) ([]*tmtypes.Validator, error) {
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
func (c *LocalClient) Subscribe(ctx context.Context, fn func(tmctypes.ResultEvent) error, queries ...string) error {
	if err := c.node.Start(); err != nil {
		return err
	}
	defer c.node.Stop()

	errCh := make(chan error)

	for _, query := range queries {
		q, err := tmquery.New(query)
		if err != nil {
			return err
		}

		// For subscription we use "vega" as the client name but it's ignored by the implementation.
		// 10 is the channel capacity which is absolutely arbitraty.
		out, err := c.node.Subscribe(ctx, "vega", q.String(), 10)
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
	defer c.node.UnsubscribeAll(context.Background(), "vega")

	return <-errCh
}

func (c *LocalClient) Start() error {
	return nil // Nothing to do for this client type.
}
