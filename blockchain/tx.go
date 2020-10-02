package blockchain

import (
	"errors"

	"code.vegaprotocol.io/vega/tx"
	uuid "github.com/satori/go.uuid"
)

// NOTE:
// These functions has been implemented in processor/{codec,tx}.go too.
// We will eventually merge both implementations

// TxEncode takes a Tx payload and a command and builds a raw Tx.
func TxEncode(input []byte, cmd tx.Command) ([]byte, error) {
	prefix := uuid.NewV4().String()
	prefixBytes := []byte(prefix)
	commandInput := append([]byte{byte(cmd)}, input...)
	return append(prefixBytes, commandInput...), nil
}

// TxDecode takes the raw payload bytes and decodes the contents using a pre-defined
// strategy, we have a simple and efficient encoding at present. A partner encode function
// can be found in the blockchain client.
func TxDecode(input []byte) ([]byte, tx.Command, error) {
	// Input is typically the bytes that arrive in raw format after consensus is reached.
	// Split the transaction dropping the unification bytes (uuid&pipe)
	if len(input) > 37 {
		// obtain command from byte slice (0 indexed)
		// remaining bytes are payload
		return input[37:], tx.Command(input[36]), nil
	}
	return nil, 0, errors.New("payload size is incorrect, should be > 38 bytes")
}
