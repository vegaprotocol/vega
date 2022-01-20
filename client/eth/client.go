package eth

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/types"
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
	mu                      sync.Mutex
	currentHeightLastUpdate time.Time
	currentHeight           uint64
}

func Dial(ctx context.Context, rawURL string) (*Client, error) {
	ethClient, err := ethclient.DialContext(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("couldn't instantiate Ethereum client: %w", err)
	}

	return &Client{ETHClient: ethClient}, nil
}

func (c *Client) OnEthereumConfigUpdate(_ context.Context, v interface{}) error {
	if c == nil {
		return nil
	}

	ethConfig, err := types.EthereumConfigFromUntypedProto(v)
	if err != nil {
		return err
	}

	return c.setEthereumConfig(ethConfig)
}

func (c *Client) setEthereumConfig(ethConfig *types.EthereumConfig) error {
	netID, err := c.NetworkID(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't retrieve the network ID form the ethereum client: %w", err)
	}

	chainID, err := c.ChainID(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't retrieve the chain ID form the ethereum client: %w", err)
	}

	if netID.String() != ethConfig.NetworkID() {
		return fmt.Errorf("updated network ID does not match the one set during start up, expected %s got %v", ethConfig.NetworkID(), netID)
	}

	if chainID.String() != ethConfig.ChainID() {
		return fmt.Errorf("updated chain ID does not matchthe one set during start up, expected %v got %v", ethConfig.ChainID(), chainID)
	}

	c.ethConfig = ethConfig

	return nil
}

func (c *Client) CollateralBridgeAddress() ethcommon.Address {
	return c.ethConfig.CollateralBridge().Address()
}

func (c *Client) CollateralBridgeAddressHex() string {
	return c.ethConfig.CollateralBridge().HexAddress()
}

func (c *Client) CurrentHeight(ctx context.Context) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If last update of the height was more than 15 seconds
	// ago, we try to update, as we assume an Ethereum block takes
	// ~15 seconds.
	if now := time.Now(); c.currentHeightLastUpdate.Add(15).Before(now) {
		lastBlockHeader, err := c.HeaderByNumber(ctx, nil)
		if err != nil {
			return c.currentHeight, err
		}
		c.currentHeightLastUpdate = now
		c.currentHeight = lastBlockHeader.Number.Uint64()
	}

	return c.currentHeight, nil
}

func (c *Client) ConfirmationsRequired() uint64 {
	return c.ethConfig.Confirmations()
}
