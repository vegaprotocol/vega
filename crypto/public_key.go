package crypto

type PublicKeyOrAddress struct {
	hex   string
	bytes []byte
}

func NewPublicKeyOrAddress(hex string, bytes []byte) PublicKeyOrAddress {
	return PublicKeyOrAddress{
		hex:   hex,
		bytes: bytes,
	}
}

func (p PublicKeyOrAddress) Hex() string {
	return p.hex
}

func (p PublicKeyOrAddress) Bytes() []byte {
	return p.bytes
}
