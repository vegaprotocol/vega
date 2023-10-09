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

package tests_test

import (
	"encoding/base64"
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/require"
)

func TestSignMessage(t *testing.T) {
	// given
	home := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, home)
	recoveryPhraseFilePath := NewFile(t, home, "recovery-phrase.txt", testRecoveryPhrase)
	walletName := vgrand.RandomStr(5)

	// when
	importWalletResp, err := WalletImport(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", passphraseFilePath,
		"--recovery-phrase-file", recoveryPhraseFilePath,
		"--version", "2",
	})

	// then
	require.NoError(t, err)
	AssertImportWallet(t, importWalletResp).
		WithName(walletName).
		LocatedUnder(home)

	// given
	message := []byte("Je ne connaîtrai pas la peur car la peur tue l'esprit.")
	encodedMessage := base64.StdEncoding.EncodeToString(message)

	// when
	signResp, err := SignMessage(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--pubkey", importWalletResp.Key.PublicKey,
		"--message", encodedMessage,
		"--passphrase-file", passphraseFilePath,
	})

	// then
	require.NoError(t, err)
	AssertSignMessage(t, signResp).
		WithSignature("StH82RHxjQ3yTeaSN25b6sJwAyLiq1CDvPWf0X4KIf/WTIjkunkWKn1Gq9ntCoGBfBZIyNfpPtGx0TSZsSrbCA==")

	// when
	verifyResp, err := VerifyMessage(t, []string{
		"--home", home,
		"--output", "json",
		"--pubkey", importWalletResp.Key.PublicKey,
		"--message", encodedMessage,
		"--signature", signResp.Signature,
	})

	// then
	require.NoError(t, err)
	AssertVerifyMessage(t, verifyResp).IsValid()
}

func TestSignMessageWithTaintedKey(t *testing.T) {
	// given
	home := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, home)
	recoveryPhraseFilePath := NewFile(t, home, "recovery-phrase.txt", testRecoveryPhrase)
	walletName := vgrand.RandomStr(5)

	// when
	importWalletResp, err := WalletImport(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", passphraseFilePath,
		"--recovery-phrase-file", recoveryPhraseFilePath,
		"--version", "2",
	})

	// then
	require.NoError(t, err)
	AssertImportWallet(t, importWalletResp).
		WithName(walletName).
		WithPublicKey("b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0").
		LocatedUnder(home)

	// when
	err = KeyTaint(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--pubkey", importWalletResp.Key.PublicKey,
		"--passphrase-file", passphraseFilePath,
	})

	// then
	require.NoError(t, err)

	// given
	message := []byte("Je ne connaîtrai pas la peur car la peur tue l'esprit.")
	encodedMessage := base64.StdEncoding.EncodeToString(message)

	// when
	signResp, err := SignMessage(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--pubkey", importWalletResp.Key.PublicKey,
		"--message", encodedMessage,
		"--passphrase-file", passphraseFilePath,
	})

	// then
	require.EqualError(t, err, "could not sign the message: the public key is tainted")
	require.Nil(t, signResp)

	// when
	err = KeyUntaint(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--pubkey", importWalletResp.Key.PublicKey,
		"--passphrase-file", passphraseFilePath,
	})

	// then
	require.NoError(t, err)

	// when
	signResp, err = SignMessage(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--pubkey", importWalletResp.Key.PublicKey,
		"--message", encodedMessage,
		"--passphrase-file", passphraseFilePath,
	})

	// then
	require.NoError(t, err)
	AssertSignMessage(t, signResp).
		WithSignature("StH82RHxjQ3yTeaSN25b6sJwAyLiq1CDvPWf0X4KIf/WTIjkunkWKn1Gq9ntCoGBfBZIyNfpPtGx0TSZsSrbCA==")
}
