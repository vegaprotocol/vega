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
	// should be exactly 64 chars
	if len(pkey) != 64 {
		return false
	}

	// should be strictly hex encoded
	if _, err := hex.DecodeString(pkey); err != nil {
		return false
	}

	return true
}
