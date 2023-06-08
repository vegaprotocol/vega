package ethcall

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"

	"code.vegaprotocol.io/vega/core/types"
)

type dataSource struct {
	call Call
	spec types.EthCallSpec
}

func newDataSource(spec types.EthCallSpec) (dataSource, error) {
	// Convert JSON args to go types using ABI
	args, err := JsonArgsToAny(spec.Method, spec.ArgsJson, spec.AbiJson)
	if err != nil {
		return dataSource{}, fmt.Errorf("unable to deserialize args: %w", err)
	}

	call, err := NewCall(spec.Method, args, spec.Address, spec.AbiJson)
	if err != nil {
		return dataSource{}, fmt.Errorf("unable to create call: %w", err)
	}

	return dataSource{
		call: call,
		spec: spec,
	}, nil
}

func (d dataSource) CallContract(ctx context.Context, caller ethereum.ContractCaller, blockNumber *big.Int) ([]byte, error) {
	return d.call.Call(ctx, caller, blockNumber)
}

func (d dataSource) RequiredConfirmations() uint64 {
	return d.spec.RequiredConfirmations
}

func (d dataSource) PassesFilters(result []byte, blockHeight uint64, blockTime uint64) bool {
	return true
}

func (d dataSource) Normalise(callResult []byte) (map[string]string, error) {
	// TODO
	result, err := d.call.UnpackResult(callResult)
	if err != nil {
		return nil, fmt.Errorf("unable to unpack result: %w", err)
	}

	return map[string]string{"price": fmt.Sprintf("%s", result)}, nil
}
