package crypto

import "golang.org/x/crypto/sha3"

func Hash(key []byte) []byte {
	hasher := sha3.New256()
	hasher.Write(key)
	return hasher.Sum(nil)
}
