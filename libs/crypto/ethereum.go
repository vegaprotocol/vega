package crypto

import (
	"github.com/ethereum/go-ethereum/common"
)

// EthereumChecksumAddress is a simple utility function
// to ensure all ethereum addresses used in vega are checksumed
// this expects a hex encoded string.
func EthereumChecksumAddress(s string) string {
	// as per docs the Hex method return EIP-55 compliant hex strings
	return common.HexToAddress(s).Hex()
}

// EthereumIsValidAddress returns whether the given string is a valid ethereum address.
func EthereumIsValidAddress(s string) bool {
	return common.IsHexAddress(s)
}
