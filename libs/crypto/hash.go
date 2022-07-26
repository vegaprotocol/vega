// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/sha3"
)

func Hash(key []byte) []byte {
	hasher := sha3.New256()
	hasher.Write(key)
	return hasher.Sum(nil)
}

// HashHexStr hash a hex encoded string with sha3 256
// returns an hex encoded string of the result.
func HashHexStr(s string) string {
	x, err := hex.DecodeString(s)
	if err != nil {
		panic(fmt.Sprintf("hex string required: %v", err))
	}
	hasher := sha3.New256()
	hasher.Write(x)
	return hex.EncodeToString(hasher.Sum(nil))
}

// HashStr hash a string (converts to bytes first)
// returns an hex encoded string of the result.
func HashStr(s string) string {
	hasher := sha3.New256()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}

func RandomHash() string {
	data := make([]byte, 10)
	rand.Read(data)
	return hex.EncodeToString(Hash(data))
}
