package ethcall

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type Call struct {
	address common.Address
	method  string
	args    []byte
	abi     abi.ABI
	abiJSON []byte
}

func NewCall(method string, args []any, address string, abiJSON []byte) (Call, error) {
	abiJSON, err := CanonicalizeJSON(abiJSON)
	if err != nil {
		return Call{}, fmt.Errorf("unable to canonicalize abi JSON: %w", err)
	}

	reader := bytes.NewReader(abiJSON)
	abi, err := abi.JSON(reader)
	if err != nil {
		return Call{}, fmt.Errorf("unable to parse abi JSON: %w", err)
	}

	packedArgs, err := abi.Pack(method, args...)
	if err != nil {
		return Call{}, fmt.Errorf("failed to pack inputs: %w", err)
	}

	return Call{
		address: common.HexToAddress(address),
		method:  method,
		args:    packedArgs,
		abi:     abi,
		abiJSON: abiJSON,
	}, nil
}

func (c Call) Args() ([]any, error) {
	inputsAbi := c.abi.Methods[c.method].Inputs

	if len(c.args) < 4 {
		return nil, fmt.Errorf("invalid packed args")
	}

	args, err := inputsAbi.Unpack(c.args[4:])
	if err != nil {
		return nil, fmt.Errorf("failed to unpack args: %w", err)
	}
	return args, nil
}

func (c Call) Call(ctx context.Context, caller ethereum.ContractCaller, blockNumber *big.Int) ([]byte, error) {
	// TODO: timeout?
	msg := ethereum.CallMsg{
		To:   &c.address,
		Data: c.args,
	}

	output, err := caller.CallContract(ctx, msg, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	return output, nil
}

func (c Call) UnpackResult(bytes []byte) ([]any, error) {
	values, err := c.abi.Unpack(c.method, bytes)
	if err != nil {
		return values, fmt.Errorf("failed to unpack contract call result: %w", err)
	}
	return values, nil
}

type Result struct {
	*Call
	Bytes []byte
}

func (r Result) Values() ([]any, error) {
	return r.UnpackResult(r.Bytes)
}
