package ethcall

import (
	"code.vegaprotocol.io/vega/core/types"
	"encoding/json"
	"fmt"
	"github.com/PaesslerAG/jsonpath"
)

type Result struct {
	Bytes         []byte
	Values        []any
	Normalised    map[string]string
	PassesFilters bool
}

func NewResult(spec types.EthCallSpec, bytes []byte) (Result, error) {
	call, err := NewCall(spec)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create result: %w", err)
	}

	return newResult(call, bytes)
}

func newResult(call Call, bytes []byte) (Result, error) {
	values, err := call.abi.Unpack(call.method, bytes)
	if err != nil {
		return Result{}, fmt.Errorf("failed to unpack contract call result: %w", err)
	}

	normalised, err := normaliseValues(values, call.spec.Normalisers)
	if err != nil {
		return Result{}, fmt.Errorf("failed to normalise contract call result: %w", err)
	}

	passesFilters := true
	// TO BE BE REPLACED BY PHILTER CHANGES - COMMENT IN TO RUN AGAINST SYSTEM TESTS
	/*
		passesFilters = false
		for _, val := range normalised {
			ival, err := strconv.Atoi(val)
			if err != nil {
				return Result{}, fmt.Errorf("unable to convert value to int: %v", err)
			}

			if ival > 25 {
				passesFilters = true
			}
		} */

	return Result{
		Bytes:         bytes,
		Values:        values,
		Normalised:    normalised,
		PassesFilters: passesFilters,
	}, nil
}

func normaliseValues(values []any, rules map[string]string) (map[string]string, error) {
	res := make(map[string]string)
	for key, path := range rules {
		value, err := jsonpath.Get(path, values)
		if err != nil {
			return nil, fmt.Errorf("unable to normalise key %v: %v", key, err)
		}
		valueJson, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("unable to serialse normalised value: %v", err)
		}

		res[key] = string(valueJson)
	}

	return res, nil
}

func (r Result) HasRequiredConfirmations() bool {
	// TODO
	return true
}
