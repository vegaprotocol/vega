package processor

import (
	"code.vegaprotocol.io/vega/blockchain/abci"
)

type TxCodec struct{}

// Decode takes a raw input from a Tendermint Tx and decodes into a vega Tx,
// the decoding process involves a signature verification.
func (c *TxCodec) Decode(payload []byte) (abci.Tx, error) {
	return DecodeTx(payload)
}

type NullBlockchainTxCodec struct{}

func (c *NullBlockchainTxCodec) Decode(payload []byte) (abci.Tx, error) {
	return DecodeTxNoValidation(payload)
}
