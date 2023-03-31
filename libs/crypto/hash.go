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
