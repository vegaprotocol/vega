package eth

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	types "code.vegaprotocol.io/protos/vega"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ETHClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_client_mock.go -package mocks code.vegaprotocol.io/vega/client/eth ETHClient
type ETHClient interface {
	bind.ContractBackend
	ChainID(context.Context) (*big.Int, error)
	NetworkID(context.Context) (*big.Int, error)
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

type Client struct {
	ETHClient
	ethConfig *types.EthereumConfig

	// this is all just to prevent spamming the infura just
	// to get the last height of the blockchain
	mu                  sync.Mutex
	curHeightLastUpdate time.Time
	curHeight           uint64
}

func Dial(ctx context.Context, rawURL string) (*Client, error) {
	ethClient, err := ethclient.DialContext(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("could not instantiate ethereum client: %w", err)
	}

	return &Client{ETHClient: ethClient}, nil
}

func (c *Client) OnEthereumConfigUpdate(ctx context.Context, v interface{}) error {
	ecfg, ok := v.(*types.EthereumConfig)
	if !ok {
		return errors.New("invalid types for Ethereum config")
	}
	return c.setEthereumConfig(ecfg)
}

func (c *Client) setEthereumConfig(ethConfig *types.EthereumConfig) error {
	nid, err := c.NetworkID(context.Background())
	if err != nil {
		return err
	}
	chid, err := c.ChainID(context.Background())
	if err != nil {
		return err
	}
	if nid.String() != ethConfig.NetworkId {
		return fmt.Errorf("ethereum network id does not match, expected %v got %v", ethConfig.NetworkId, nid)
	}
	if chid.String() != ethConfig.ChainId {
		return fmt.Errorf("ethereum chain id does not match, expected %v got %v", ethConfig.ChainId, chid)
	}
	c.ethConfig = ethConfig
	return nil
}

func (c *Client) BridgeAddress() ethcommon.Address {
	return ethcommon.HexToAddress(c.ethConfig.BridgeAddress)
}

func (c *Client) BridgeAddressHex() string {
	return c.ethConfig.BridgeAddress
}

func (c *Client) CurrentHeight(ctx context.Context) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// if last update of the heigh was more that 15 seconds
	// ago, we try to update, we assume an eth block takes
	// ~15 seconds
	now := time.Now()
	if c.curHeightLastUpdate.Add(15).Before(now) {
		// get the last block header
		h, err := c.HeaderByNumber(context.Background(), nil)
		if err != nil {
			return c.curHeight, err
		}
		c.curHeightLastUpdate = now
		c.curHeight = h.Number.Uint64()
	}

	return c.curHeight, nil
}

func (c *Client) ConfirmationsRequired() uint32 {
	return c.ethConfig.Confirmations
}
