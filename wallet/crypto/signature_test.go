package crypto_test

import (
	"crypto"
	"testing"

	wcrypto "code.vegaprotocol.io/vega/wallet/crypto"
	"github.com/stretchr/testify/assert"
)

func TestSignature(t *testing.T) {
	t.Run("create signature ed25519 success", testCreateEd25519SignatureOK)
	t.Run("create signature ed25519 fail", testCreateSignatureFailureNotAnAlgo)
	t.Run("generate key success", testGenerateKey)
	t.Run("verify success", testVerifyOK)
	t.Run("verify fail wrong message", testVerifyFailWrongMessage)
	t.Run("verify fail wrong pubkey", testVerifyFailWrongPubKey)
	t.Run("sign fail bad key length", testSignBadKeyLength)
	t.Run("verify fail bad key length", testVerifyBadKeyLength)
}

func testCreateEd25519SignatureOK(t *testing.T) {
	_, err := wcrypto.NewSignatureAlgorithm(wcrypto.Ed25519)
	assert.NoError(t, err)
}

func testCreateSignatureFailureNotAnAlgo(t *testing.T) {
	_, err := wcrypto.NewSignatureAlgorithm("not an algo")
	assert.EqualError(t, err, wcrypto.ErrUnsupportedSignatureAlgorithm.Error())
}

func testGenerateKey(t *testing.T) {
	s, err := wcrypto.NewSignatureAlgorithm(wcrypto.Ed25519)
	assert.NoError(t, err)
	_, _, err = s.GenKey()
	assert.NoError(t, err)
}

func testVerifyOK(t *testing.T) {
	s, err := wcrypto.NewSignatureAlgorithm(wcrypto.Ed25519)
	assert.NoError(t, err)
	pub, priv, err := s.GenKey()
	assert.NoError(t, err)

	message := []byte("hello world")

	sig := s.Sign(priv, message)
	assert.NotEmpty(t, sig)

	ok := s.Verify(pub, message, sig)
	assert.True(t, ok)
}

func testSignBadKeyLength(t *testing.T) {
	s, err := wcrypto.NewSignatureAlgorithm(wcrypto.Ed25519)
	assert.NoError(t, err)
	_, priv, err := s.GenKey()

	assert.NoError(t, err)

	message := []byte("hello world")

	// Chop one byte off the key
	priv2 := priv.([]byte)
	priv3 := priv2[0 : len(priv2)-1]
	sig := s.Sign(crypto.PrivateKey(priv3), message)
	// No error, just nil
	assert.Nil(t, sig)
}

func testVerifyBadKeyLength(t *testing.T) {
	s, err := wcrypto.NewSignatureAlgorithm(wcrypto.Ed25519)
	assert.NoError(t, err)
	pub, priv, err := s.GenKey()

	assert.NoError(t, err)

	message := []byte("hello world")

	sig := s.Sign(priv, message)
	assert.NotEmpty(t, sig)

	// Chop one byte off the key
	pub2 := pub.([]byte)
	pub3 := pub2[0 : len(pub2)-1]
	ok := s.Verify(crypto.PublicKey(pub3), message, sig)
	// No error, just false
	assert.False(t, ok)
}

func testVerifyFailWrongMessage(t *testing.T) {
	s, err := wcrypto.NewSignatureAlgorithm(wcrypto.Ed25519)
	assert.NoError(t, err)
	pub, priv, err := s.GenKey()
	assert.NoError(t, err)

	message := []byte("hello world")
	wrongmessage := []byte("yolo")

	sig := s.Sign(priv, message)
	assert.NotEmpty(t, sig)

	ok := s.Verify(pub, wrongmessage, sig)
	assert.False(t, ok)
}

func testVerifyFailWrongPubKey(t *testing.T) {
	s, err := wcrypto.NewSignatureAlgorithm(wcrypto.Ed25519)
	assert.NoError(t, err)
	// gen 2 sets of  keys
	_, priv, err := s.GenKey()
	assert.NoError(t, err)
	pub, _, err := s.GenKey()
	assert.NoError(t, err)

	message := []byte("hello world")

	sig := s.Sign(priv, message)
	assert.NotEmpty(t, sig)

	ok := s.Verify(pub, message, sig)
	assert.False(t, ok)
}
