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

func TestDescribeKeyFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testKeyDescribeValidFlagsSucceeds)
	t.Run("Missing wallet fails", testKeyMissingWalletFails)
	t.Run("Missing public key fails", testKeyMissingPublicKeyFails)
}

func testKeyDescribeValidFlagsSucceeds(t *testing.T) {
	// given
	testDir := t.TempDir()
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(10)

	f := &cmd.DescribeKeyFlags{
		Wallet:         walletName,
		PassphraseFile: passphraseFilePath,
		PublicKey:      pubKey,
	}

	expectedReq := api.AdminDescribeKeyParams{
		Wallet:    walletName,
		PublicKey: pubKey,
	}

	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedReq, req)
	assert.Equal(t, expectedPassphrase, passphrase)
}

func testKeyMissingWalletFails(t *testing.T) {
	// given
	testDir := t.TempDir()

	// given
	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	pubKey := vgrand.RandomStr(10)

	f := &cmd.DescribeKeyFlags{
		PassphraseFile: passphraseFilePath,
		PublicKey:      pubKey,
	}

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	require.Empty(t, req)
}

func testKeyMissingPublicKeyFails(t *testing.T) {
	// given
	testDir := t.TempDir()

	// given
	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)

	f := &cmd.DescribeKeyFlags{
		PassphraseFile: passphraseFilePath,
		Wallet:         walletName,
	}

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("pubkey"))
	require.Empty(t, req)
}
