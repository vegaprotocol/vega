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

func TestAnnotateKeyFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testAnnotateKeyFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testAnnotateKeyFlagsMissingWalletFails)
	t.Run("Missing public key fails", testAnnotateKeyFlagsMissingPubKeyFails)
	t.Run("Missing metadata fails", testAnnotateKeyFlagsMissingMetadataAndClearFails)
	t.Run("Clearing with metadata fails", testAnnotateKeyFlagsClearingWithMetadataFails)
	t.Run("Invalid metadata fails", testAnnotateKeyFlagsInvalidMetadataFails)
}

func testAnnotateKeyFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	f := &cmd.AnnotateKeyFlags{
		Wallet:         walletName,
		PubKey:         pubKey,
		PassphraseFile: passphraseFilePath,
		RawMetadata:    []string{"name:my-wallet", "role:validation"},
		Clear:          false,
	}

	expectedReq := api.AdminAnnotateKeyParams{
		Wallet:    walletName,
		PublicKey: pubKey,
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

func testAnnotateKeyFlagsMissingWalletFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newAnnotateKeyFlags(t, testDir)
	f.Wallet = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func testAnnotateKeyFlagsMissingPubKeyFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newAnnotateKeyFlags(t, testDir)
	f.PubKey = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("pubkey"))
	assert.Empty(t, req)
}

func testAnnotateKeyFlagsMissingMetadataAndClearFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newAnnotateKeyFlags(t, testDir)
	f.RawMetadata = []string{}

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.OneOfFlagsMustBeSpecifiedError("meta", "clear"))
	assert.Empty(t, req)
}

func testAnnotateKeyFlagsClearingWithMetadataFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newAnnotateKeyFlags(t, testDir)
	f.Clear = true

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MutuallyExclusiveError("meta", "clear"))
	assert.Empty(t, req)
}

func testAnnotateKeyFlagsInvalidMetadataFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newAnnotateKeyFlags(t, testDir)
	f.RawMetadata = []string{"is=invalid"}

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.InvalidFlagFormatError("meta"))
	assert.Empty(t, req)
}

func newAnnotateKeyFlags(t *testing.T, testDir string) *cmd.AnnotateKeyFlags {
	t.Helper()

	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	return &cmd.AnnotateKeyFlags{
		Wallet:         walletName,
		PubKey:         pubKey,
		PassphraseFile: passphraseFilePath,
		RawMetadata:    []string{"name:my-wallet", "role:validation"},
		Clear:          false,
	}
}
