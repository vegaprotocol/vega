package ethcall

import (
	"encoding/json"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
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

	normalised, err := normaliseValues(values, call.spec.Normaliser)
	return Result{
		Bytes:         bytes,
		Values:        values,
		Normalised:    normalised,
		PassesFilters: true,
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
