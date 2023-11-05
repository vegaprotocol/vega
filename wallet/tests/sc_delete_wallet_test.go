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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteWallet(t *testing.T) {
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
	err = WalletDelete(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--force",
	})

	// then
	require.NoError(t, err)
	assert.NoFileExists(t, createWalletResp.Wallet.FilePath)
}

func TestDeleteNonExistingWallet(t *testing.T) {
	home := t.TempDir()

	// when
	err := WalletDelete(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", vgrand.RandomStr(5),
		"--force",
	})

	// then
	require.Error(t, err)
	assert.Equal(t, "the wallet does not exist", err.Error())
}
