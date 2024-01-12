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

package ethcall_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"embed"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	eth_log "github.com/ethereum/go-ethereum/log"
)

//go:embed testdata/*
var testData embed.FS

type ToyChain struct {
	key          *ecdsa.PrivateKey
	client       *Client
	addr         common.Address
	contractAddr common.Address
	abiBytes     []byte
}

type Client struct {
	*backends.SimulatedBackend
}

func (c *Client) ChainID(context.Context) (*big.Int, error) {
	return big.NewInt(1337), nil
}

func NewToyChain() (*ToyChain, error) {
	// Stop go-ethereum writing loads of uninteresting logs
	eth_log.Root().SetHandler(eth_log.DiscardHandler())

	// Setup keys
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("could not generate key: %w", err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	signer, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	if err != nil {
		return nil, fmt.Errorf("couldn't create signer: %w", err)
	}

	// Setup simulated backend (works a bit like ganache) with a balance so we can deploy contracts
	client := backends.NewSimulatedBackend(
		core.GenesisAlloc{
			addr: {Balance: big.NewInt(10000000000000000)},
		}, 10000000,
	)

	// Read in contract ABI
	contractAbiBytes, err := testData.ReadFile("testdata/MyContract.abi")
	if err != nil {
		log.Fatal(err)
	}

	contractAbi, err := abi.JSON(bytes.NewReader(contractAbiBytes))
	if err != nil {
		return nil, fmt.Errorf("could not get code at test addr: %w", err)
	}

	// Read in contract bytecode
	contractBytecodeBytes, err := testData.ReadFile("testdata/MyContract.bin")
	if err != nil {
		log.Fatal(err)
	}

	// Deploy contract
	contractAddr, _, _, err := bind.DeployContract(signer, contractAbi, common.FromHex(string(contractBytecodeBytes)), client)
	if err != nil {
		return nil, fmt.Errorf("could not deploy contract")
	}

	client.Commit()

	return &ToyChain{
		key:          key,
		client:       &Client{client},
		addr:         addr,
		contractAddr: contractAddr,
		abiBytes:     contractAbiBytes,
	}, nil
}
