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
