package processor

import (
	"code.vegaprotocol.io/vega/blockchain/abci"
)

type codec struct{}

// Decode takes a raw input from a Tendermint Tx and decodes into a vega Tx,
// the decoding process involves a signature verification.
func (c *codec) Decode(payload []byte) (abci.Tx, error) {
	return DecodeTxV2(payload)
}
