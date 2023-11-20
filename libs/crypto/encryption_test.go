// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
