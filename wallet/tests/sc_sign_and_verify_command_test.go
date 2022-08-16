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
		`{"voteSubmission": {"proposalId": "1da3c57bfc2ff8fac2bd2160e5bed5f88f49d1d54d655918cf0758585f248ef7", "value": "VALUE_YES"}}`,
	})

	// then
	require.EqualError(t, err, "couldn't sign transaction: public key is tainted")
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
		`{"voteSubmission": {"proposalId": "1da3c57bfc2ff8fac2bd2160e5bed5f88f49d1d54d655918cf0758585f248ef7", "value": "VALUE_YES"}}`,
	})

	// then
	require.NoError(t, err)
	AssertSignCommand(t, signResp)
}
