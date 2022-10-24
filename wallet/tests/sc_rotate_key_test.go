package tests_test

import (
	"testing"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"github.com/stretchr/testify/require"
)

func TestRotateKeySucceeds(t *testing.T) {
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
	resp, err := KeyRotate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--chain-id", "testnet",
		"--passphrase-file", passphraseFilePath,
		"--current-pubkey", createWalletResp.Key.PublicKey,
		"--new-pubkey", generateKeyResp.PublicKey,
		"--tx-height", "20",
		"--target-height", "25",
	})

	// then
	require.NoError(t, err)
	AssertKeyRotate(t, resp)
}

func TestRotateKeyFailsOnTaintedPublicKey(t *testing.T) {
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
	err = KeyTaint(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", passphraseFilePath,
		"--pubkey", generateKeyResp.PublicKey,
	})

	// then
	require.NoError(t, err)

	// when
	resp, err := KeyRotate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--chain-id", "testnet",
		"--passphrase-file", passphraseFilePath,
		"--current-pubkey", createWalletResp.Key.PublicKey,
		"--new-pubkey", generateKeyResp.PublicKey,
		"--tx-height", "20",
		"--target-height", "25",
	})

	// then
	require.Nil(t, resp)
	require.EqualError(t, err, api.ErrNextPublicKeyIsTainted.Error())
}

func TestRotateKeyFailsInIsolatedWallet(t *testing.T) {
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
	resp, err := KeyRotate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", isolateKeyResp.Wallet,
		"--chain-id", "testnet",
		"--passphrase-file", isolatedWalletPassphraseFilePath,
		"--new-pubkey", createWalletResp.Key.PublicKey,
		"--current-pubkey", "current-public-key",
		"--tx-height", "20",
		"--target-height", "25",
	})

	// then
	require.Nil(t, resp)
	require.EqualError(t, err, api.ErrCannotRotateKeysOnIsolatedWallet.Error())
}

func TestRotateKeyFailsOnNonExitingNewPublicKey(t *testing.T) {
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
	KeyRotateResp, err := KeyRotate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--chain-id", "testnet",
		"--passphrase-file", passphraseFilePath,
		"--current-pubkey", createWalletResp.Key.PublicKey,
		"--new-pubkey", "nonexisting",
		"--tx-height", "20",
		"--target-height", "25",
	})

	// then
	require.Nil(t, KeyRotateResp)
	require.EqualError(t, err, api.ErrNextPublicKeyDoesNotExist.Error())
}

func TestRotateKeyFailsOnNonExitingCurrentPublicKey(t *testing.T) {
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
	keyRotateResp, err := KeyRotate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--chain-id", "testnet",
		"--passphrase-file", passphraseFilePath,
		"--current-pubkey", "nonexisting",
		"--new-pubkey", createWalletResp.Key.PublicKey,
		"--tx-height", "20",
		"--target-height", "25",
	})

	// then
	require.Nil(t, keyRotateResp)
	require.EqualError(t, err, api.ErrCurrentPublicKeyDoesNotExist.Error())
}

func TestRotateKeyFailsOnNonExitingWallet(t *testing.T) {
	// given
	home := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, home)
	walletName := vgrand.RandomStr(5)

	// when
	keyRotateResp, err := KeyRotate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--chain-id", "testnet",
		"--passphrase-file", passphraseFilePath,
		"--new-pubkey", "nonexisting1",
		"--current-pubkey", "nonexisting2",
		"--tx-height", "20",
		"--target-height", "25",
	})

	// then
	require.Nil(t, keyRotateResp)
	require.EqualError(t, err, api.ErrWalletDoesNotExist.Error())
}

func TestRotateKeyFailsWhenTargetHeightIsLessThanTxHeight(t *testing.T) {
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
	keyRotateResp, err := KeyRotate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--chain-id", "testnet",
		"--passphrase-file", passphraseFilePath,
		"--new-pubkey", "nonexisting",
		"--current-pubkey", "nonexisting",
		"--tx-height", "20",
		"--target-height", "19",
	})

	// then
	require.Nil(t, keyRotateResp)
	require.ErrorIs(t, err, flags.RequireLessThanFlagError("tx-height", "target-height"))
}
