package tests_test

import (
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/require"
)

func TestIsolateKey(t *testing.T) {
	// given
	home := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, home)
	_, isolatedWalletPassphraseFilePath := NewPassphraseFile(t, home)
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
	isolateKeyResp, err := KeyIsolate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--pubkey", createWalletResp.Key.PublicKey,
		"--passphrase-file", passphraseFilePath,
		"--isolated-wallet-passphrase-file", isolatedWalletPassphraseFilePath,
	})

	// then
	require.NoError(t, err)
	AssertIsolateKey(t, isolateKeyResp).
		WithSpecialName(walletName, createWalletResp.Key.PublicKey).
		LocatedUnder(home)

	// when
	generateKeyRespOnIsolatedWallet, err := KeyGenerate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", isolateKeyResp.Wallet,
		"--passphrase-file", isolatedWalletPassphraseFilePath,
	})

	// then
	require.EqualError(t, err, "could not generate a new key: an isolated wallet can't generate keys")
	require.Nil(t, generateKeyRespOnIsolatedWallet)
}
