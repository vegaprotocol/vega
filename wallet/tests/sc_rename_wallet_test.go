package tests_test

import (
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenameWallet(t *testing.T) {
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

	// given
	newWalletName := vgrand.RandomStr(5)

	// when
	err = WalletRename(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--new-name", newWalletName,
	})

	// then
	require.NoError(t, err)

	// when
	listKeysResp, err := KeyList(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", newWalletName,
		"--passphrase-file", passphraseFilePath,
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, listKeysResp)
	require.Len(t, listKeysResp.Keys, 1)
	assert.Equal(t, listKeysResp.Keys[0].PublicKey, createWalletResp.Key.PublicKey)
}
