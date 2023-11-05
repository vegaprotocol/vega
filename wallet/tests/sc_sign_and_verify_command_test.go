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
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"

	"github.com/stretchr/testify/require"
)

func TestSignCommand(t *testing.T) {
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

	// when
	signResp, err := SignCommand(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--chain-id", vgrand.RandomStr(5),
		"--pubkey", importWalletResp.Key.PublicKey,
		"--passphrase-file", passphraseFilePath,
		"--tx-height", "150",
		"--tx-block-hash", "1da3c57bfc2ff8fac2bd2160e5bed5f88f49d1d54d655918cf0758585f248ef7",
		"--pow-difficulty", "2",
		"--pow-hash-function", "sha3_24_rounds",
		`{"voteSubmission": {"proposalId": "1da3c57bfc2ff8fac2bd2160e5bed5f88f49d1d54d655918cf0758585f248ef7", "value": "VALUE_YES"}}`,
	})

	// then
	require.NoError(t, err)
	AssertSignCommand(t, signResp)
}

func TestSignCommandWithTaintedKey(t *testing.T) {
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

	// when
	signResp, err := SignCommand(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--chain-id", vgrand.RandomStr(5),
		"--pubkey", importWalletResp.Key.PublicKey,
		"--passphrase-file", passphraseFilePath,
		"--tx-height", "150",
		"--tx-block-hash", "1da3c57bfc2ff8fac2bd2160e5bed5f88f49d1d54d655918cf0758585f248ef7",
		"--pow-difficulty", "2",
		"--pow-hash-function", "sha3_24_rounds",
		`{"voteSubmission": {"proposalId": "1da3c57bfc2ff8fac2bd2160e5bed5f88f49d1d54d655918cf0758585f248ef7", "value": "VALUE_YES"}}`,
	})

	// then
	require.EqualError(t, err, "could not sign the transaction: the public key is tainted")
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
	signResp, err = SignCommand(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--chain-id", vgrand.RandomStr(5),
		"--pubkey", importWalletResp.Key.PublicKey,
		"--passphrase-file", passphraseFilePath,
		"--tx-height", "150",
		"--tx-block-hash", "1da3c57bfc2ff8fac2bd2160e5bed5f88f49d1d54d655918cf0758585f248ef7",
		"--pow-difficulty", "2",
		"--pow-hash-function", "sha3_24_rounds",
		`{"voteSubmission": {"proposalId": "1da3c57bfc2ff8fac2bd2160e5bed5f88f49d1d54d655918cf0758585f248ef7", "value": "VALUE_YES"}}`,
	})

	// then
	require.NoError(t, err)
	AssertSignCommand(t, signResp)
}
