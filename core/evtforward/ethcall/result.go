package ethcall

import (
	"encoding/json"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"github.com/PaesslerAG/jsonpath"
)

type Result struct {
	call  Call
	bytes []byte
}

func NewResult(spec types.EthCallSpec, bytes []byte) (Result, error) {
	call, err := NewCall(spec)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create result: %w", err)
	}

	return Result{
		call:  call,
		bytes: bytes,
	}, nil
}

func (r Result) Bytes() []byte {
	return r.bytes
}

func (r Result) Values() ([]any, error) {
	values, err := r.call.abi.Unpack(r.call.method, r.bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack contract call result: %w", err)
	}
	return values, nil
}

func (r Result) Normalised() (map[string]string, error) {
	values, err := r.Values()
	if err != nil {
		return nil, err
	}

	res := make(map[string]string)
	for key, path := range r.call.spec.Normaliser {
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

func (r Result) PassesFilters() (bool, error) {
	// TODO
	return true, nil
}

func (r Result) HasRequiredConfirmations() bool {
	// TODO
	return true
}
