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

	"golang.org/x/crypto/sha3"
)

func Hash(key []byte) []byte {
	hasher := sha3.New256()
	hasher.Write(key)
	return hasher.Sum(nil)
}

func RandomHash() string {
	data := make([]byte, 10)
	rand.Read(data)
	return hex.EncodeToString(Hash(data))
}
