package ethcall

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// JsonArgsToAny takes a list of arguments marshalled as JSON strings.
// It then uses the ethereum ABI to convert each JSON argument into the go type
// which corresponds to the ethereum type defined in the ABI for that argument.
func JsonArgsToAny(methodName string, jsonArgs []string, abiJSON string) ([]any, error) {
	abi, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("unable to parse abi json: %w", err)
	}

	methodAbi, ok := abi.Methods[methodName]
	if !ok {
		return nil, fmt.Errorf("method %s not found in abi", methodName)
	}

	inputsAbi := methodAbi.Inputs
	if len(inputsAbi) != len(jsonArgs) {
		return nil, fmt.Errorf("expected %v arguments for method %s, got %v", len(inputsAbi), methodName, len(jsonArgs))
	}

	args := []any{}
	for i, jsonArg := range jsonArgs {
		argType := inputsAbi[i].Type.GetType()
		argIsPointer := argType.Kind() == reflect.Pointer

		if argIsPointer {
			argType = argType.Elem()
		}

		newArgValue := reflect.New(argType) // A reflect.Value of kind 'Pointer' to new instance of argType

		err := json.Unmarshal([]byte(jsonArg), newArgValue.Interface())
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal json argument %s: %w", jsonArg, err)
		}

		if argIsPointer {
			args = append(args, newArgValue.Interface())
		} else {
			args = append(args, newArgValue.Elem().Interface())
		}
	}
	return args, nil
}

// AnyArgsToJson does the inverse of the JsonArgsToAny; takes a list of arguments in go types
// and marshals them to a list of JSON strings.
func AnyArgsToJson(args []any) ([]string, error) {
	result := make([]string, 0, len(args))
	for _, arg := range args {
		argJSON, err := json.Marshal(arg)
		if err != nil {
			return []string{}, fmt.Errorf("failed to json marshall args to JSON: %w", err)
		}
		result = append(result, string(argJSON))
	}

	return result, nil
}

func CanonicalizeJSON(in []byte) ([]byte, error) {
	var data interface{}
	if err := json.Unmarshal(in, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	out, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json: %w", err)
	}
	return out, nil
}
