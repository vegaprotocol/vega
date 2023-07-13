package ethcall

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"text/scanner"

	"code.vegaprotocol.io/vega/core/types"
	"github.com/PaesslerAG/gval"
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

	passesFilters, err := call.filters.Match(normalised)
	if err != nil {
		return Result{}, fmt.Errorf("error evaluating filters: %w", err)
	}

	return Result{
		Bytes:         bytes,
		Values:        values,
		Normalised:    normalised,
		PassesFilters: passesFilters,
	}, nil
}

func normaliseValues(values []any, rules map[string]string) (map[string]string, error) {
	// The data in 'values' is relatively well typed, after being unpacked by the ABI.
	// For example, a uint256 will be a big.Int, structs returned from the contract call
	// will be anonymous go structs rather than a string map etc.. In order to fish data
	// out with jsonpath it need to be simple lists, maps, strings and numbers; so we
	// serialise to json and then deserialise to an into an []interface{}.
	valuesJson, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("unable to serialse values: %v", err)
	}

	valuesSimple := []interface{}{}
	d := json.NewDecoder(bytes.NewBuffer(valuesJson))
	// Keep numbers as a json.Number, which holds the original string representation
	// otherwise all numbers get cast to float64 which is no good for e.g uint256.
	d.UseNumber()
	err = d.Decode(&valuesSimple)
	if err != nil {
		return nil, fmt.Errorf("unable to deserialse values: %v", err)
	}

	res := make(map[string]string)

	for key, path := range rules {
		value, err := myJSONPathGet(path, valuesSimple)
		if err != nil {
			return nil, fmt.Errorf("unable to normalise key %v: %v", key, err)
		}
		switch v := value.(type) {
		case json.Number:
			res[key] = v.String()
		case int64:
			// all of the numbers in the json from the ethereum call result will be
			// json.Number and handled above; this case is just for the corner case
			// where someone specifies a number as a static value in the json path itself
			res[key] = strconv.FormatInt(v, 10)
		case string:
			res[key] = v
		default:
			return nil, fmt.Errorf("unable to normalise key %v of type %T", key, value)
		}
	}

	return res, nil
}

// myJSONPathGet works exactly like jsonpath.Get(path, values), except that any numbers found
// are returned as int64 rather than float64 which is the default.
// ** in the path expression itself, not the json being queried **
// Evaluation will fail if numbers in the path are not integers.
func myJSONPathGet(path string, values []interface{}) (interface{}, error) {
	baselang := jsonpath.Language()
	mylang := gval.PrefixExtension(scanner.Int, parseNumberAsInt64)
	lang := gval.NewLanguage(baselang, mylang)

	eval, err := lang.NewEvaluable(path)
	if err != nil {
		return nil, err
	}
	value, err := eval(context.Background(), values)
	return value, err
}

func parseNumberAsInt64(c context.Context, p *gval.Parser) (gval.Evaluable, error) {
	n, err := strconv.ParseInt(p.TokenText(), 10, 64)
	if err != nil {
		return nil, err
	}
	return p.Const(n), nil
}
