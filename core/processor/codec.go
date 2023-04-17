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

// Codec is the abci codec, with the extra methods not needed by the abci package, but used
// to update values like max TTL, that are in turn used to when decoding transactions.
type Codec interface {
	abci.Codec
	UpdateMaxTTL(ttl uint64)
}

// BaseCodec is the base codec, shared by the TxCodec and null blockchain codec. Just to make life easier adding
// things like the max TTL netparam value to both.
type BaseCodec struct {
	maxTTL uint64
}

type TxCodec struct {
	BaseCodec
}

// Decode takes a raw input from a Tendermint Tx and decodes into a vega Tx,
// the decoding process involves a signature verification.
func (c *TxCodec) Decode(payload []byte, chainID string) (abci.Tx, error) {
	return DecodeTx(payload, chainID, c.maxTTL)
}

type NullBlockchainTxCodec struct {
	BaseCodec
}

func (c *NullBlockchainTxCodec) Decode(payload []byte, _ string) (abci.Tx, error) {
	return DecodeTxNoValidation(payload, c.maxTTL)
}

func (c *BaseCodec) UpdateMaxTTL(ttl uint64) {
	c.maxTTL = ttl
}
