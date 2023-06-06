package ethcall_test

import (
	"bytes"
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
	client       *backends.SimulatedBackend
	addr         common.Address
	contractAddr common.Address
	abiBytes     []byte
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
		client:       client,
		addr:         addr,
		contractAddr: contractAddr,
		abiBytes:     contractAbiBytes,
	}, nil
}
