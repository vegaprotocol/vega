package crypto

import (
	"crypto"

	"golang.org/x/crypto/ed25519"
)

type ed25519Sig struct{}

func newEd25519() *ed25519Sig {
	return &ed25519Sig{}
}

func (e *ed25519Sig) GenKey() (crypto.PublicKey, crypto.PrivateKey, error) {
	return ed25519.GenerateKey(nil)
}

func (e *ed25519Sig) Sign(priv crypto.PrivateKey, buf []byte) []byte {
	return ed25519.Sign(priv.(ed25519.PrivateKey), buf)
}

func (e *ed25519Sig) Verify(pub crypto.PublicKey, message, sig []byte) bool {
	return ed25519.Verify(pub.(ed25519.PublicKey), message, sig)
}

func (e *ed25519Sig) Name() string {
	return "ed25519"
}
