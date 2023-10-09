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

func TestListAPITokens(t *testing.T) {
	// given
	home := t.TempDir()
	_, tokensPassphraseFilePath := NewPassphraseFile(t, home)
	_, walletPassphraseFilePath := NewPassphraseFile(t, home)
	walletName := vgrand.RandomStr(5)

	// when
	err := InitAPIToken(t, []string{
		"--home", home,
		"--passphrase-file", tokensPassphraseFilePath,
	})

	// then
	require.NoError(t, err)

	// when
	listTokensResp1, err := APITokensList(t, []string{
		"--home", home,
		"--passphrase-file", tokensPassphraseFilePath,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, listTokensResp1)
	require.Len(t, listTokensResp1.Tokens, 0)

	// when
	createWalletResp, err := WalletCreate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", walletPassphraseFilePath,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	AssertCreateWallet(t, createWalletResp).
		WithName(walletName).
		LocatedUnder(home)

	// when
	generateAPITokenResp1, err := APITokenGenerate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet-name", walletName,
		"--wallet-passphrase-file", walletPassphraseFilePath,
		"--tokens-passphrase-file", tokensPassphraseFilePath,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	AssertGenerateAPIToken(t, generateAPITokenResp1)

	// when
	generateAPITokenResp2, err := APITokenGenerate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet-name", walletName,
		"--wallet-passphrase-file", walletPassphraseFilePath,
		"--tokens-passphrase-file", tokensPassphraseFilePath,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	AssertGenerateAPIToken(t, generateAPITokenResp2)

	// when
	listTokensResp2, err := APITokensList(t, []string{
		"--home", home,
		"--passphrase-file", tokensPassphraseFilePath,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, listTokensResp2)
	require.Len(t, listTokensResp2.Tokens, 2)
	assert.Equal(t, generateAPITokenResp1.Token, listTokensResp2.Tokens[0].Token)
	assert.Equal(t, generateAPITokenResp2.Token, listTokensResp2.Tokens[1].Token)

	// when
	err = APITokenDelete(t, []string{
		"--home", home,
		"--token", generateAPITokenResp1.Token,
		"--passphrase-file", tokensPassphraseFilePath,
		"--output", "json",
		"--force",
	})

	// then
	require.NoError(t, err)

	// when
	listTokensResp3, err := APITokensList(t, []string{
		"--home", home,
		"--passphrase-file", tokensPassphraseFilePath,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, listTokensResp3)
	require.Len(t, listTokensResp3.Tokens, 1)
	assert.Equal(t, generateAPITokenResp2.Token, listTokensResp3.Tokens[0].Token)
}
