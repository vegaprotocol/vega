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

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/sha3"
)

func Hash(key []byte) []byte {
	hashFunc := sha3.New256()
	hashFunc.Write(key)
	return hashFunc.Sum(nil)
}

func HashBytesBuffer(key bytes.Buffer) []byte {
	hashFunc := sha3.New256()
	key.WriteTo(hashFunc)
	return hashFunc.Sum(nil)
}

// HashToHex hash the input bytes and returns a hex encoded string of the result.
func HashToHex(data []byte) string {
	return hex.EncodeToString(Hash(data))
}

// HashStrToHex hash a string returns a hex encoded string of the result.
func HashStrToHex(s string) string {
	return hex.EncodeToString(Hash([]byte(s)))
}

func RandomHash() string {
	data := make([]byte, 10)
	if _, err := rand.Read(data); err != nil {
		panic(fmt.Errorf("couldn't generate random string: %w", err))
	}
	return hex.EncodeToString(Hash(data))
}
