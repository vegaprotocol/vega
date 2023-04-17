package cmd_test

import (
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKeyFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testGenerateKeyFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testGenerateKeyFlagsMissingWalletFails)
	t.Run("Invalid metadata fails", testGenerateKeyFlagsInvalidMetadataFails)
}

func testGenerateKeyFlagsValidFlagsSucceeds(t *testing.T) {
	// given
	testDir := t.TempDir()
	walletName := vgrand.RandomStr(10)
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)

	f := &cmd.GenerateKeyFlags{
		Wallet:         walletName,
		PassphraseFile: passphraseFilePath,
		RawMetadata:    []string{"name:my-wallet", "role:validation"},
	}

	expectedReq := api.AdminGenerateKeyParams{
		Wallet: walletName,
		Metadata: []wallet.Metadata{
			{Key: "name", Value: "my-wallet"},
			{Key: "role", Value: "validation"},
		},
	}

	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
	assert.Equal(t, expectedPassphrase, passphrase)
}

func testGenerateKeyFlagsMissingWalletFails(t *testing.T) {
	// given
	f := newGenerateKeyFlags(t)
	f.Wallet = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func testGenerateKeyFlagsInvalidMetadataFails(t *testing.T) {
	// given
	f := newGenerateKeyFlags(t)
	f.RawMetadata = []string{"is=invalid"}

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.InvalidFlagFormatError("meta"))
	assert.Empty(t, req)
}

func newGenerateKeyFlags(t *testing.T) *cmd.GenerateKeyFlags {
	t.Helper()
	testDir := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	return &cmd.GenerateKeyFlags{
		Wallet:         vgrand.RandomStr(5),
		PassphraseFile: passphraseFilePath,
		RawMetadata:    []string{"name:my-wallet", "role:validation"},
	}
}
