package tests_test

import (
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatePassphrase(t *testing.T) {
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
	listKeysResp, err := KeyList(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", passphraseFilePath,
	})

	//
	require.NoError(t, err)
	require.NotNil(t, listKeysResp)
	require.Len(t, listKeysResp.Keys, 1)
	assert.Equal(t, listKeysResp.Keys[0].PublicKey, createWalletResp.Key.PublicKey)

	// given
	_, newPassphraseFilePath := NewPassphraseFile(t, home)

	// when
	err = PassphraseUpdate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", passphraseFilePath,
		"--new-passphrase-file", newPassphraseFilePath,
	})

	// then
	require.NoError(t, err)

	// when unlocking wallet with previous passphrase
	listKeysResp, err = KeyList(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", passphraseFilePath,
	})

	// then
	require.Error(t, err)
	require.Nil(t, listKeysResp)

	// when unlocking wallet with new passphrase
	listKeysResp, err = KeyList(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", newPassphraseFilePath,
	})

	//
	require.NoError(t, err)
	require.NotNil(t, listKeysResp)
	require.Len(t, listKeysResp.Keys, 1)
	assert.Equal(t, listKeysResp.Keys[0].PublicKey, createWalletResp.Key.PublicKey)
}
