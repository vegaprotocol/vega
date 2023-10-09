// Copyright (C) 2023  Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
