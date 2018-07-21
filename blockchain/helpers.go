package blockchain

import (
	"errors"
	"github.com/satori/go.uuid"
	"bytes"
)

func VegaTxEncode(input []byte, cmd Command) (proto []byte, err error) {
	prefix := uuid.NewV4().String() + "|"
	prefixBytes := []byte(prefix)
	//commandInput := append([]byte{cmd}, input...)
	return append(prefixBytes, input...), nil
}

func VegaTxDecode(input []byte) (proto []byte, cmd Command, err error) {

	// Input is typically the bytes that arrive from Tendermint after consensus is reached.
	// Split the transaction dropping the unification bytes (uuid&pipe)
	var _, value []byte
	parts := bytes.Split(input, []byte("|"))       // todo(cdm): use byte positions rather than split
	if len(parts) == 2 {
		_, value = parts[0], parts[1]
	} else {
		return nil, 0, errors.New("decoding error when splitting transaction")
	}
	
	return value, CreateOrderCommand, nil
}

