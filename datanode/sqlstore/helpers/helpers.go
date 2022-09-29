package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"strconv"
)

// GenerateID generates a 256 bit pseudo-random hash ID.
func GenerateID() string {
	randomString := strconv.FormatInt(rand.Int63(), 10)
	hash := sha256.Sum256([]byte(randomString))
	return hex.EncodeToString(hash[:])
}
