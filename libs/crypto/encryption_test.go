package crypto_test

import (
	"testing"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	"github.com/stretchr/testify/assert"
)

func TestEncryption(t *testing.T) {
	t.Run("Encrypting and decrypting data succeeds", testEncryptingAndDecryptingDataSucceeds)
	t.Run("Decrypting with wrong passphrase fails", testDecryptingWithWrongPassphraseFails)
}

func testEncryptingAndDecryptingDataSucceeds(t *testing.T) {
	data := []byte("hello world")
	passphrase := "oh yea?"

	encryptedBuf, err := vgcrypto.Encrypt(data, passphrase)
	assert.NoError(t, err)
	assert.NotEmpty(t, encryptedBuf)

	decryptedBuf, err := vgcrypto.Decrypt(encryptedBuf, passphrase)
	assert.NoError(t, err)
	assert.Equal(t, data, decryptedBuf)
}

func testDecryptingWithWrongPassphraseFails(t *testing.T) {
	data := []byte("hello world")
	passphrase := "oh yea?"
	wrongPassphrase := "oh really!"

	encryptedBuf, err := vgcrypto.Encrypt(data, passphrase)
	assert.NoError(t, err)
	assert.NotEmpty(t, encryptedBuf)

	decryptedBuf, err := vgcrypto.Decrypt(encryptedBuf, wrongPassphrase)
	assert.Error(t, err)
	assert.NotEqual(t, data, decryptedBuf)
}
