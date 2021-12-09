package eth

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	types "code.vegaprotocol.io/protos/vega"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	ErrUnexpectedContractAddress = errors.New("unexpected contract address - cannot verify contract code")
	ErrContractHashMismatch      = errors.New("mismatched contract code")
)

// ContractHashes map[contract addresses] -> sha3-256(contract bytecode).
var ContractHashes = map[string]string{
	// vega mainnet1 - eth mainnet
	"0x23d1bFE8fA50a167816fBD79D7932577c06011f4": "0d83655b7be5c60f6762723e29f20ce1ed6e0f4d4de0781c2a2332e83d65a11f", // vesting
	"0x195064D33f09e0c42cF98E665D9506e0dC17de68": "889a39f8323d9fd1fd6bcc9a549fc4cf6a41b537af01d3f0523343398e12b5d7", // staking

	// vega testnet1 - eth ropsten
	"0xfce2CC92203A266a9C8e67461ae5067c78f67235": "e91eb100c4cbecb6c404de873bf84457b5313da8e39fed231edc965812f12ae1", // staking
	"0x0614188938f5C3bD8461D4B413A39eeC2C5f42D9": "885e507d590170eae2a3b52d56894e1486eb25f6c65ca92e6e37e952d4f75e33", // vesting

}

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
	if c == nil {
		return nil
	}
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
	if now := time.Now(); c.curHeightLastUpdate.Add(15).Before(now) {
		// get the last block header
		h, err := c.HeaderByNumber(ctx, nil)
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

// VerifyContract takes the address of a contract in hex and checks the hash of the byte-code is as expected.
func (c *Client) VerifyContract(ctx context.Context, address string) error {
	expected, ok := ContractHashes[address]
	if !ok {
		return fmt.Errorf("%w: address %s", ErrUnexpectedContractAddress, address)
	}

	// nil block number means latest block
	b, err := c.CodeAt(ctx, ethcommon.HexToAddress(address), nil)
	if err != nil {
		return err
	}

	actual := hex.EncodeToString(vgcrypto.Hash(b))
	if expected != actual {
		return fmt.Errorf("%w: address: %s, expected: %s got %s", ErrContractHashMismatch, address, expected, actual)
	}
	return nil
}
