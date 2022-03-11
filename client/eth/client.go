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

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	ErrUnexpectedContractHash   = errors.New("hash of contract bytecode not as expected")
	ErrUnexpectedSolidityFormat = errors.New("unexpected format of solidity bytecode")
)

// ContractHashes the sha3-256(contract-bytecode stripped of metadata).
var ContractHashes = map[string]string{
	"staking":    "d66948e12817f8ae6ca94d56b43ca12e66416e7e9bc23bb09056957b25afc6bd",
	"vesting":    "5278802577f4aca315b9524bfa78790f8f0fae08939ec58bc9e8f0ea40123b09",
	"collateral": "6d201a69218822cac1990d960674d857a0e52aa7401694e86070e778365bc6c0",
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

func (c *Client) UpdateEthereumConfig(ethConfig *types.EthereumConfig) error {
	if c == nil {
		return nil
	}

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

	// if err := c.verifyStakingContract(context.Background(), ethConfig); err != nil {
	// 	return fmt.Errorf("failed to verify staking bridge contract: %w", err)
	// }

	// if err := c.verifyVestingContract(context.Background(), ethConfig); err != nil {
	// 	return fmt.Errorf("failed to verify vesting bridge contract: %w", err)
	// }

	// if err := c.verifyCollateralContract(context.Background(), ethConfig); err != nil {
	// 	return fmt.Errorf("failed to verify collateral bridge contract: %w", err)
	// }

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

// VerifyContract takes the address of a contract in hex and checks the hash of the byte-code is as expected.
func (c *Client) VerifyContract(ctx context.Context, address ethcommon.Address, expectedHash string) error {
	// nil block number means latest block
	b, err := c.CodeAt(ctx, address, nil)
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
	if h != expectedHash {
		return fmt.Errorf("%w: address: %s, hash: %s, expected: %s", ErrUnexpectedContractHash, address, h, expectedHash)
	}

	return nil
}

func (c *Client) verifyStakingContract(ctx context.Context, ethConfig *types.EthereumConfig) error {
	if address := ethConfig.StakingBridge(); address.HasAddress() {
		return c.VerifyContract(ctx, address.Address(), ContractHashes["staking"])
	}
	return nil
}

func (c *Client) verifyVestingContract(ctx context.Context, ethConfig *types.EthereumConfig) error {
	if address := ethConfig.VestingBridge(); address.HasAddress() {
		return c.VerifyContract(ctx, address.Address(), ContractHashes["vesting"])
	}
	return nil
}

func (c *Client) verifyCollateralContract(ctx context.Context, ethConfig *types.EthereumConfig) error {
	if address := ethConfig.CollateralBridge(); address.HasAddress() {
		return c.VerifyContract(ctx, address.Address(), ContractHashes["collateral"])
	}
	return nil
}
