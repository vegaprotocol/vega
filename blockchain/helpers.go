package blockchain

import (
	"errors"
	"github.com/satori/go.uuid"
	"bytes"
)

func VegaTxEncode(input []byte, cmd Command) (proto []byte, err error) {
	prefix := uuid.NewV4().String() + "|"
	prefixBytes := []byte(prefix)
	commandInput := append([]byte{byte(cmd)}, input...)
	return append(prefixBytes, commandInput...), nil
}

func VegaTxDecode(input []byte) (proto []byte, cmd Command, err error) {

	// Input is typically the bytes that arrive from Tendermint after consensus is reached.
	// Split the transaction dropping the unification bytes (uuid&pipe)
	var _, value []byte
	var cmdByte byte
	parts := bytes.Split(input, []byte("|"))       // todo(cdm): use byte positions for uuid rather than split (36 bytes)
	if len(parts) == 2 && len(parts[1]) > 2 {
		_, value = parts[0], parts[1]
		// obtain command from byte slice
		cmdByte = value[0]
		// remaining bytes are payload
		value = value[1:]
	} else {
		return nil, 0, errors.New("decoding error when splitting transaction")
	}
	
	return value, Command(cmdByte), nil
}

