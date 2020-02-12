package crypto

import (
	"crypto"
	"encoding/json"
	"errors"

	"golang.org/x/crypto/ed25519"
)

const (
	Ed25519 string = "ed25519"
)

var (
	ErrUnsupportedSignatureAlgorithm = errors.New("unsupported signature algorithm")
)

type SignatureAlgorithm struct {
	impl signatureAlgorithmImpl
}

type signatureAlgorithmImpl interface {
	GenKey() (crypto.PublicKey, crypto.PrivateKey, error)
	Sign(priv crypto.PrivateKey, buf []byte) []byte
	Verify(pub crypto.PublicKey, message, sig []byte) bool
	Name() string
}

func NewEd25519() SignatureAlgorithm {
	return SignatureAlgorithm{
		impl: newEd25519(),
	}
}

func NewSignatureAlgorithm(algo string) (SignatureAlgorithm, error) {
	switch algo {
	case Ed25519:
		return NewEd25519(), nil
	default:
		return SignatureAlgorithm{}, ErrUnsupportedSignatureAlgorithm
	}

}

func (s *SignatureAlgorithm) GenKey() (crypto.PublicKey, crypto.PrivateKey, error) {
	return s.impl.GenKey()
}

func (s *SignatureAlgorithm) Sign(priv crypto.PrivateKey, buf []byte) []byte {
	return s.impl.Sign(priv.(ed25519.PrivateKey), buf)
}

func (s *SignatureAlgorithm) Verify(pub crypto.PublicKey, message, sig []byte) bool {
	return s.impl.Verify(pub.(ed25519.PublicKey), message, sig)
}

func (s *SignatureAlgorithm) Name() string {
	return s.impl.Name()
}

func (s *SignatureAlgorithm) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Name())
}

func (s *SignatureAlgorithm) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}

	switch name {
	case Ed25519:
		s.impl = newEd25519()
		return nil
	default:
		return ErrUnsupportedSignatureAlgorithm
	}
}
