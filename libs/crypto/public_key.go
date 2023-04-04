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

package crypto

import "encoding/hex"

type PublicKey struct {
	hex   string
	bytes []byte
}

func NewPublicKey(hex string, bytes []byte) PublicKey {
	return PublicKey{
		hex:   hex,
		bytes: bytes,
	}
}

func (p PublicKey) Hex() string {
	return p.hex
}

func (p PublicKey) Bytes() []byte {
	return p.bytes
}

func IsValidVegaPubKey(pkey string) bool {
	return IsValidVegaID(pkey)
}

func IsValidVegaID(id string) bool {
	// should be exactly 64 chars
	if len(id) != 64 {
		return false
	}

	// should be strictly hex encoded
	if _, err := hex.DecodeString(id); err != nil {
		return false
	}

	return true

}
