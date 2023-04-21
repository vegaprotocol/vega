// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package processor

import (
	"code.vegaprotocol.io/vega/core/blockchain/abci"
)

type TxCodec struct{}

// Decode takes a raw input from a Tendermint Tx and decodes into a vega Tx,
// the decoding process involves a signature verification.
func (c *TxCodec) Decode(payload []byte, chainID string) (abci.Tx, error) {
	return DecodeTx(payload, chainID)
}

type NullBlockchainTxCodec struct{}

func (c *NullBlockchainTxCodec) Decode(payload []byte, _ string) (abci.Tx, error) {
	return DecodeTxNoValidation(payload)
}
