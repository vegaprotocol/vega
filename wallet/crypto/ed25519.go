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
	privBytes := priv.([]byte)
	// Avoid panic by checking key length
	if len(privBytes) != ed25519.PrivateKeySize {
		return nil
	}
	return ed25519.Sign(privBytes, buf)
}

func (e *ed25519Sig) Verify(pub crypto.PublicKey, message, sig []byte) bool {
	pubBytes := pub.([]byte)
	// Avoid panic by checking key length
	if len(pubBytes) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(pubBytes, message, sig)
}

func (e *ed25519Sig) Name() string {
	return "ed25519"
}
