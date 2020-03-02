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
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}

	return []byte(pub), []byte(priv), nil
}

func (e *ed25519Sig) Sign(priv crypto.PrivateKey, buf []byte) []byte {
	return ed25519.Sign(priv.([]byte), buf)
}

func (e *ed25519Sig) Verify(pub crypto.PublicKey, message, sig []byte) bool {
	return ed25519.Verify(pub.([]byte), message, sig)
}

func (e *ed25519Sig) Name() string {
	return "ed25519"
}
