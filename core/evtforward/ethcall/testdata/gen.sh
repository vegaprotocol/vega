#!/bin/bash

# shanghai EVM is a pain to set up using go-ethereum's simulated backend
solc --abi mycontract.sol --evm-version paris --bin --overwrite -o .
# go run github.com/ethereum/go-ethereum/cmd/abigen@latest --abi Store.abi --bin Store.bin --pkg ethcall_test --out ../testcontract_bindings_test.go
