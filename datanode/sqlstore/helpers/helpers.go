package helpers

import (
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
)

// GenerateID generates a 256 bit pseudo-random hash ID.
func GenerateID() string {
	return vgcrypto.RandomHash()
}
