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
