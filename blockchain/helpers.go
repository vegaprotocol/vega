package blockchain

import (
	"errors"
	"github.com/satori/go.uuid"
)

func VegaTxEncode(input []byte, cmd Command) (proto []byte, err error) {
	prefix := uuid.NewV4().String()
	prefixBytes := []byte(prefix)
	commandInput := append([]byte{byte(cmd)}, input...)
	return append(prefixBytes, commandInput...), nil
}

func VegaTxDecode(input []byte) (proto []byte, cmd Command, err error) {
	// Input is typically the bytes that arrive from Tendermint after consensus is reached.
	// Split the transaction dropping the unification bytes (uuid&pipe)
	var value []byte
	var cmdByte byte
	if len(input) > 37 {
		// obtain command from byte slice (0 indexed)
		cmdByte = input[36]
		// remaining bytes are payload
		value = input[37:]
	} else {
		return nil, 0, errors.New("payload size is incorrect, should be > 38 bytes")
	}
	return value, Command(cmdByte), nil
}
