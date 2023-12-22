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

package ethcall

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	dscommon "code.vegaprotocol.io/vega/core/datasource/common"
	ethcallcommon "code.vegaprotocol.io/vega/core/datasource/external/ethcall/common"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type Call struct {
	spec    ethcallcommon.Spec
	address common.Address
	method  string
	args    []byte
	abi     abi.ABI
	abiJSON []byte
	filters dscommon.Filters
	chainID uint64
}

func NewCall(spec ethcallcommon.Spec) (Call, error) {
	abiJSON, err := CanonicalizeJSON(spec.AbiJson)
	if err != nil {
		return Call{}, errors.Join(
			ethcallcommon.ErrInvalidEthereumAbi,
			fmt.Errorf("unable to canonicalize abi JSON: %w", err))
	}

	reader := bytes.NewReader(abiJSON)
	abi, err := abi.JSON(reader)
	if err != nil {
		return Call{}, errors.Join(
			ethcallcommon.ErrInvalidEthereumAbi,
			fmt.Errorf("unable to parse abi JSON: %w", err))
	}

	args, err := JsonArgsToAny(spec.Method, spec.ArgsJson, spec.AbiJson)
	if err != nil {
		return Call{}, errors.Join(
			ethcallcommon.ErrInvalidCallArgs,
			fmt.Errorf("unable to deserialize args: %w", err))
	}

	packedArgs, err := abi.Pack(spec.Method, args...)
	if err != nil {
		return Call{}, errors.Join(
			ethcallcommon.ErrInvalidCallArgs,
			fmt.Errorf("failed to pack inputs: %w", err))
	}

	filters, err := dscommon.NewFilters(spec.Filters, true)
	if err != nil {
		return Call{}, errors.Join(
			ethcallcommon.ErrInvalidFilters,
			fmt.Errorf("failed to create filters: %w", err))
	}

	return Call{
		address: common.HexToAddress(spec.Address),
		method:  spec.Method,
		args:    packedArgs,
		abi:     abi,
		abiJSON: abiJSON,
		spec:    spec,
		filters: filters,
		chainID: spec.L2ChainID,
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

	return newResult(c, bytes)
}

func (c Call) Spec() ethcallcommon.Spec {
	return c.spec
}

func (c Call) triggered(prevEthBlock blockish, currentEthBlock blockish) bool {
	switch trigger := c.spec.Trigger.(type) {
	case ethcallcommon.TimeTrigger:
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

func (c Call) initialTime() uint64 {
	switch trigger := c.spec.Trigger.(type) {
	case ethcallcommon.TimeTrigger:
		return trigger.Initial
	}
	return 0
}
