package crypto_test

import (
	"testing"

	"code.vegaprotocol.io/vega/wallet/crypto"
	"github.com/stretchr/testify/assert"
)

func TestSignature(t *testing.T) {
	t.Run("create signature ed25519 success", testCreateEd25519SignatureOK)
	t.Run("create signature ed25519 fail", testCreateSignatureFailureNotAnAlgo)
	t.Run("generate key success", testGenerateKey)
	t.Run("verify success", testVerifyOK)
	t.Run("verify fail wrong message", testVerifyFailWrongMessage)
	t.Run("verify fail wrong pubkey", testVerifyFailWrongPubKey)
}

func testCreateEd25519SignatureOK(t *testing.T) {
	_, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
	assert.NoError(t, err)
}

func testCreateSignatureFailureNotAnAlgo(t *testing.T) {
	_, err := crypto.NewSignatureAlgorithm("not an algo")
	assert.EqualError(t, err, crypto.ErrUnsupportedSignatureAlgorithm.Error())
}

func testGenerateKey(t *testing.T) {
	s, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
	assert.NoError(t, err)
	_, _, err = s.GenKey()
	assert.NoError(t, err)
}

func testVerifyOK(t *testing.T) {
	s, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
	assert.NoError(t, err)
	pub, priv, err := s.GenKey()
	assert.NoError(t, err)

	message := []byte("hello world")

	sig := s.Sign(priv, message)
	assert.NotEmpty(t, sig)

	ok := s.Verify(pub, message, sig)
	assert.True(t, ok)
}

func testVerifyFailWrongMessage(t *testing.T) {
	s, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
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
	s, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
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
