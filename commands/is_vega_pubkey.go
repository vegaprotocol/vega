package commands

import (
	"encoding/hex"
	"errors"
)

const (
	vegaPublicKeyLen = 64
	vegaIDLen        = 64
)

var (
	ErrShouldBeAValidVegaPublicKey = errors.New("should be a valid vega public key")
	ErrShouldBeAValidVegaID        = errors.New("should be a valid Vega ID")
)

// IsVegaPublicKey check if a string is a valid Vega public key.
// A public key is a string of 64 characters containing only hexadecimal characters.
// Despite being similar to the function IsVegaID, the Vega ID and public
// key are different concept that generated, and used differently.
func IsVegaPublicKey(key string) bool {
	_, err := hex.DecodeString(key)
	return len(key) == vegaPublicKeyLen && err == nil
}

// IsVegaID check if a string is a valid Vega public key.
// An ID is a string of 64 characters containing only hexadecimal characters.
// Despite being similar to the function IsVegaPublicKey, the Vega ID and public
// key are different concept that generated, and used differently.
func IsVegaID(id string) bool {
	_, err := hex.DecodeString(id)
	return len(id) == vegaIDLen && err == nil
}
