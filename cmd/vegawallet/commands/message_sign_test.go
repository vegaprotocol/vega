package cmd_test

import (
	"encoding/base64"
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignMessageFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testSignMessageFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testSignMessageFlagsMissingWalletFails)
	t.Run("Missing public key fails", testSignMessageFlagsMissingPubKeyFails)
	t.Run("Missing message fails", testSignMessageFlagsMissingMessageFails)
	t.Run("Malformed message fails", testSignMessageFlagsMalformedMessageFails)
}

func testSignMessageFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)
	encodedMessage := vgrand.RandomStr(20)

	f := &cmd.SignMessageFlags{
		Wallet:         walletName,
		PubKey:         pubKey,
		Message:        encodedMessage,
		PassphraseFile: passphraseFilePath,
	}

	expectedReq := api.AdminSignMessageParams{
		Wallet:         walletName,
		PubKey:         pubKey,
		EncodedMessage: encodedMessage,
	}

	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
	assert.Equal(t, expectedPassphrase, passphrase)
}

func testSignMessageFlagsMissingWalletFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignMessageFlags(t, testDir)
	f.Wallet = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func testSignMessageFlagsMissingPubKeyFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignMessageFlags(t, testDir)
	f.PubKey = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("pubkey"))
	assert.Empty(t, req)
}

func testSignMessageFlagsMissingMessageFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignMessageFlags(t, testDir)
	f.Message = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("message"))
	assert.Empty(t, req)
}

func testSignMessageFlagsMalformedMessageFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignMessageFlags(t, testDir)
	f.Message = vgrand.RandomStr(5)

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBase64EncodedError("message"))
	assert.Empty(t, req)
}

func newSignMessageFlags(t *testing.T, testDir string) *cmd.SignMessageFlags {
	t.Helper()

	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)
	decodedMessage := []byte(vgrand.RandomStr(20))

	return &cmd.SignMessageFlags{
		Wallet:         walletName,
		PubKey:         pubKey,
		Message:        base64.StdEncoding.EncodeToString(decodedMessage),
		PassphraseFile: passphraseFilePath,
	}
}
