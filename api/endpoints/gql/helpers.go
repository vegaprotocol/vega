package gql

import (
	"strconv"
	"fmt"
	"github.com/pkg/errors"
)

func SafeStringUint64(input string) (uint64, error) {
	if i, err := strconv.ParseUint(input, 10, 64); err == nil {
		fmt.Printf("i=%d, type: %T\n", i, i)
		return i, nil
	}
	// A conversion error occurred, return the error
	return 0, errors.New(fmt.Sprintf("Invalid input string for uint64 conversion %s", input))
}
