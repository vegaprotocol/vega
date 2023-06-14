package ethcall

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"code.vegaprotocol.io/vega/core/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type Call struct {
	spec    types.EthCallSpec
	address common.Address
	method  string
	args    []byte
	abi     abi.ABI
	abiJSON []byte
}

func NewCall(spec types.EthCallSpec) (Call, error) {
	abiJSON, err := CanonicalizeJSON(spec.AbiJson)
	if err != nil {
		return Call{}, fmt.Errorf("unable to canonicalize abi JSON: %w", err)
	}

	reader := bytes.NewReader(abiJSON)
	abi, err := abi.JSON(reader)
	if err != nil {
		return Call{}, fmt.Errorf("unable to parse abi JSON: %w", err)
	}

	args, err := JsonArgsToAny(spec.Method, spec.ArgsJson, spec.AbiJson)
	if err != nil {
		return Call{}, fmt.Errorf("unable to deserialize args: %w", err)
	}

	packedArgs, err := abi.Pack(spec.Method, args...)
	if err != nil {
		return Call{}, fmt.Errorf("failed to pack inputs: %w", err)
	}

	return Call{
		address: common.HexToAddress(spec.Address),
		method:  spec.Method,
		args:    packedArgs,
		abi:     abi,
		abiJSON: abiJSON,
		spec:    spec,
	}, nil
}

func (c Call) Call(ctx context.Context, ethClient EthReaderCaller, blockNumber uint64) (Result, error) {
	// TODO: timeout?
	msg := ethereum.CallMsg{
		To:   &c.address,
		Data: c.args,
	}

	n := big.NewInt(0).SetUint64(blockNumber)
	bytes, err := ethClient.CallContract(ctx, msg, n)
	if err != nil {
		return Result{}, fmt.Errorf("failed to call contract: %w", err)
	}

	return Result{
		bytes: bytes,
		call:  c,
	}, nil
}

func (c Call) triggered(prevEthBlock blockish, currentEthBlock blockish) bool {
	switch trigger := c.spec.Trigger.(type) {
	case *types.EthTimeTrigger:
		// Before initial?
		if currentEthBlock.Time() < trigger.Initial {
			return false
		}

		// Crossing initial boundary?
		if prevEthBlock.Time() < trigger.Initial && currentEthBlock.Time() >= trigger.Initial {
			return true
		}

		// After until?
		if trigger.Until != 0 && currentEthBlock.Time() > trigger.Until {
			return false
		}

		if trigger.Every == 0 {
			return false
		}
		// Somewhere in the middle..
		prevTriggerCount := (prevEthBlock.Time() - trigger.Initial) / trigger.Every
		currentTriggerCount := (currentEthBlock.Time() - trigger.Initial) / trigger.Every
		return currentTriggerCount > prevTriggerCount
	}
	return false
}
