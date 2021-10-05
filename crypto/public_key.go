package crypto

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
