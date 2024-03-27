// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type SecondaryClient struct {
	ETHClient
	ethConfig *types.EVMChainConfig

	// this is all just to prevent spamming the infura just
	// to get the last height of the blockchain
	mu                      sync.Mutex
	currentHeightLastUpdate time.Time
	currentHeight           uint64

	retryDelay time.Duration
}

func SecondaryDial(ctx context.Context, cfg Config) (*SecondaryClient, error) {
	if len(cfg.SecondaryRPCEndpoint) <= 0 {
		return nil, errors.New("no secondary ethereum rpc endpoint configured")
	}

	ethClient, err := ethclient.DialContext(ctx, cfg.SecondaryRPCEndpoint)
	if err != nil {
		return nil, fmt.Errorf("couldn't instantiate secondary Ethereum client: %w", err)
	}

	return &SecondaryClient{
		ETHClient:  newEthClientWrapper(ethClient),
		retryDelay: cfg.RetryDelay.Get(),
	}, nil
}

func (c *SecondaryClient) UpdateEthereumConfig(ctx context.Context, ethConfig *types.EVMChainConfig) error {
	if c == nil {
		return nil
	}

	netID, err := c.NetworkID(ctx)
	if err != nil {
		return fmt.Errorf("couldn't retrieve the network ID from the ethereum client: %w", err)
	}

	chainID, err := c.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("couldn't retrieve the chain ID form the ethereum client: %w", err)
	}

	if netID.String() != ethConfig.NetworkID() {
		return fmt.Errorf("updated network ID does not match the one set during start up, expected %s got %v", ethConfig.NetworkID(), netID)
	}

	if chainID.String() != ethConfig.ChainID() {
		return fmt.Errorf("updated chain ID does not match the one set during start up, expected %v got %v", ethConfig.ChainID(), chainID)
	}

	c.ethConfig = ethConfig

	return nil
}

func (c *SecondaryClient) CollateralBridgeAddress() ethcommon.Address {
	return c.ethConfig.CollateralBridge().Address()
}

func (c *SecondaryClient) CollateralBridgeAddressHex() string {
	return c.ethConfig.CollateralBridge().HexAddress()
}

func (c *SecondaryClient) CurrentHeight(ctx context.Context) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if now := time.Now(); c.currentHeightLastUpdate.Add(c.retryDelay).Before(now) {
		lastBlockHeader, err := c.HeaderByNumber(ctx, nil)
		if err != nil {
			return c.currentHeight, err
		}
		c.currentHeightLastUpdate = now
		c.currentHeight = lastBlockHeader.Number.Uint64()
	}

	return c.currentHeight, nil
}

func (c *SecondaryClient) ConfirmationsRequired() uint64 {
	return c.ethConfig.Confirmations()
}

// VerifyContract takes the address of a contract in hex and checks the hash of the byte-code is as expected.
func (c *SecondaryClient) VerifyContract(ctx context.Context, address ethcommon.Address, expectedHash string) error {
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
