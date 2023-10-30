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

func TestListWallets(t *testing.T) {
	// given
	home := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, home)
	walletName1 := "a" + vgrand.RandomStr(5)

	// when
	createWalletResp1, err := WalletCreate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName1,
		"--passphrase-file", passphraseFilePath,
	})

	// then
	require.NoError(t, err)
	AssertCreateWallet(t, createWalletResp1).
		WithName(walletName1).
		LocatedUnder(home)

	// when
	listWalletsResp1, err := WalletList(t, []string{
		"--home", home,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, listWalletsResp1)
	require.Len(t, listWalletsResp1.Wallets, 1)
	assert.Equal(t, listWalletsResp1.Wallets[0], createWalletResp1.Wallet.Name)

	// given
	walletName2 := "b" + vgrand.RandomStr(5)

	// when
	createWalletResp2, err := WalletCreate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName2,
		"--passphrase-file", passphraseFilePath,
	})

	// then
	require.NoError(t, err)
	AssertCreateWallet(t, createWalletResp2).
		WithName(walletName2).
		LocatedUnder(home)

	// when
	listWalletsResp2, err := WalletList(t, []string{
		"--home", home,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, listWalletsResp2)
	require.Len(t, listWalletsResp2.Wallets, 2)
	assert.Equal(t, listWalletsResp2.Wallets[0], createWalletResp1.Wallet.Name)
	assert.Equal(t, listWalletsResp2.Wallets[1], createWalletResp2.Wallet.Name)
}
