package ethcall

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
)

type DataSource struct {
	Call
	spec types.EthCallSpec
}

func NewDataSource(spec types.EthCallSpec) (DataSource, error) {
	// Convert JSON args to go types using ABI
	args, err := JsonArgsToAny(spec.Method, spec.ArgsJson, spec.AbiJson)
	if err != nil {
		return DataSource{}, fmt.Errorf("unable to deserialize args: %w", err)
	}

	call, err := NewCall(spec.Method, args, spec.Address, spec.AbiJson)
	if err != nil {
		return DataSource{}, fmt.Errorf("unable to create call: %w", err)
	}

	return DataSource{
		Call: call,
		spec: spec,
	}, nil
}

func (c DataSource) RequiredConfirmations() uint64 {
	return c.spec.RequiredConfirmations
}

func (c DataSource) Pass(result []byte, blockHeight uint64, blockTime uint64) bool {
	return true
}

func (c DataSource) Normalise(callResult []byte) (map[string]string, error) {
	// TODO
	result, err := c.Call.UnpackResult(callResult)
	if err != nil {
		return nil, fmt.Errorf("unable to unpack result: %w", err)
	}

	return map[string]string{"price": fmt.Sprintf("%s", result)}, nil
}

func (c DataSource) Trigger(prev blockish, current blockish) bool {
	// Before initial?
	switch trigger := c.spec.Trigger.(type) {
	case *types.EthTimeTrigger:
		if current.Time() < trigger.Initial {
			return false
		}

		// Crossing initial boundary?
		if prev.Time() < trigger.Initial && current.Time() >= trigger.Initial {
			return true
		}

		// After until?
		if trigger.Until != 0 && current.Time() > trigger.Until {
			return false
		}

		if trigger.Every == 0 {
			return false
		}
		// Somewhere in the middle..
		prevTriggerCount := (prev.Time() - trigger.Initial) / trigger.Every
		currentTriggerCount := (current.Time() - trigger.Initial) / trigger.Every
		return currentTriggerCount > prevTriggerCount
	}

	return false
}
