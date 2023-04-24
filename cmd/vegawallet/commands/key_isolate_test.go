package cmd_test

import (
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsolateKeyFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testIsolateKeyFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testIsolateKeyFlagsMissingWalletFails)
	t.Run("Missing public key fails", testIsolateKeyFlagsMissingPubKeyFails)
}

func testIsolateKeyFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	isolatedPassphrase, isolatedPassphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	f := &cmd.IsolateKeyFlags{
		Wallet:                       walletName,
		PubKey:                       pubKey,
		PassphraseFile:               passphraseFilePath,
		IsolatedWalletPassphraseFile: isolatedPassphraseFilePath,
	}

	expectedReq := api.AdminIsolateKeyParams{
		Wallet:                   walletName,
		PublicKey:                pubKey,
		IsolatedWalletPassphrase: isolatedPassphrase,
	}

	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedReq, req)
	assert.Equal(t, expectedPassphrase, passphrase)
}

func testIsolateKeyFlagsMissingWalletFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newIsolateKeyFlags(t, testDir)
	f.Wallet = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func testIsolateKeyFlagsMissingPubKeyFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newIsolateKeyFlags(t, testDir)
	f.PubKey = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("pubkey"))
	assert.Empty(t, req)
}

func newIsolateKeyFlags(t *testing.T, testDir string) *cmd.IsolateKeyFlags {
	t.Helper()

	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	_, isolatedWalletPassphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	return &cmd.IsolateKeyFlags{
		Wallet:                       walletName,
		PubKey:                       pubKey,
		PassphraseFile:               passphraseFilePath,
		IsolatedWalletPassphraseFile: isolatedWalletPassphraseFilePath,
	}
}
