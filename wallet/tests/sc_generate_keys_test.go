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

func TestGenerateKey(t *testing.T) {
	// given
	home := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, home)
	walletName := vgrand.RandomStr(5)

	// when
	createWalletResp, err := WalletCreate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", passphraseFilePath,
	})

	// then
	require.NoError(t, err)
	AssertCreateWallet(t, createWalletResp).
		WithName(walletName).
		LocatedUnder(home)

	// when
	descResp, err := KeyDescribe(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", passphraseFilePath,
		"--pubkey", createWalletResp.Key.PublicKey,
	})

	// then
	require.NoError(t, err)
	AssertDescribeKey(t, descResp).
		WithMeta(map[string]string{"name": "Key 1"}).
		WithAlgorithm("vega/ed25519", 1).
		WithTainted(false)

	// when
	generateKeyResp, err := KeyGenerate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", passphraseFilePath,
		"--meta", "name:key-2,role:validation",
	})

	// then
	require.NoError(t, err)
	AssertGenerateKey(t, generateKeyResp).
		WithMetadata(map[string]string{"name": "key-2", "role": "validation"})

	// when
	descResp, err = KeyDescribe(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", passphraseFilePath,
		"--pubkey", generateKeyResp.PublicKey,
	})

	// then
	require.NoError(t, err)
	AssertDescribeKey(t, descResp).
		WithMeta(map[string]string{"name": "key-2", "role": "validation"}).
		WithAlgorithm("vega/ed25519", 1).
		WithTainted(false)
}
