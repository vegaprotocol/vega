// Copyright (C) 2023 Gobalsky Labs Limited
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

package test

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"

	"golang.org/x/crypto/sha3"
)

func RandomVegaID() string {
	data := make([]byte, 10)
	if _, err := crand.Read(data); err != nil {
		panic(fmt.Errorf("couldn't generate random string: %w", err))
	}

	hashFunc := sha3.New256()
	hashFunc.Write(data)
	hashedData := hashFunc.Sum(nil)

	return hex.EncodeToString(hashedData)
}

func RandomNegativeI64() int64 {
	return (rand.Int63n(1000) + 1) * -1
}

func RandomNegativeI64AsString() string {
	return strconv.FormatInt(RandomNegativeI64(), 10)
}

func RandomI64() int64 {
	return rand.Int63()
}

func RandomPositiveI64() int64 {
	return rand.Int63()
}

func RandomPositiveI64Before(n int64) int64 {
	return rand.Int63n(n)
}

func RandomPositiveU32() uint32 {
	return rand.Uint32() + 1
}

func RandomPositiveU64() uint64 {
	return rand.Uint64() + 1
}

func RandomPositiveU64AsString() string {
	return strconv.FormatUint(RandomPositiveU64(), 10)
}

func RandomPositiveU64Before(n int64) uint64 {
	return uint64(rand.Int63n(n))
}
