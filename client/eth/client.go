package eth

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
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
	ErrUnexpectedContractHash   = errors.New("hash of contract bytecode not as expected")
	ErrUnexpectedSolidityFormat = errors.New("unexpected format of solidity bytecode")
)

// ContractHashes the sha3-256(bytecode)
var ContractHashes = map[string]struct{}{
	"d66948e12817f8ae6ca94d56b43ca12e66416e7e9bc23bb09056957b25afc6bd": {}, // staking
	"5278802577f4aca315b9524bfa78790f8f0fae08939ec58bc9e8f0ea40123b09": {}, // vesting

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
	// nil block number means latest block
	b, err := c.CodeAt(ctx, ethcommon.HexToAddress(address), nil)
	if err != nil {
		return err
	}

	// the bytecode of the contract is appended which is deployment specific. We only care about
	// the contract code itself and so we need to strip this meta-data before hashing it. For the version
	// of Solidity we use, the format is [contract-bytecode]a264[CBOR-encoded meta-data]
	asHex := strings.Split(hex.EncodeToString(b), "a264")
	if len(asHex) != 2 {
		return fmt.Errorf("%w: address: %s", ErrUnexpectedSolidityFormat, address)
	}

	// Back to bytes for hashing
	b, err = hex.DecodeString(asHex[0])
	if err != nil {
		return err
	}

	h := hex.EncodeToString(vgcrypto.Hash(b))
	if _, ok := ContractHashes[h]; !ok {
		return fmt.Errorf("%w: address: %s, hash: %s", ErrUnexpectedContractHash, address, h)
	}

	return nil
}
