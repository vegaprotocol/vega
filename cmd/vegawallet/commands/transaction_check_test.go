package cmd_test

import (
	"encoding/json"
	"fmt"
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCommandFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testCheckCommandFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testCheckCommandFlagsMissingWalletFails)
	t.Run("Missing log level fails", testCheckCommandFlagsMissingLogLevelFails)
	t.Run("Unsupported log level fails", testCheckCommandFlagsUnsupportedLogLevelFails)
	t.Run("Missing network and node address fails", testCheckCommandFlagsMissingNetworkAndNodeAddressFails)
	t.Run("Both network and node address specified fails", testCheckCommandFlagsBothNetworkAndNodeAddressSpecifiedFails)
	t.Run("Missing public key fails", testCheckCommandFlagsMissingPubKeyFails)
	t.Run("Missing request fails", testCheckCommandFlagsMissingRequestFails)
	t.Run("Malformed request fails", testCheckCommandFlagsMalformedRequestFails)
}

func testCheckCommandFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	network := vgrand.RandomStr(10)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	f := &cmd.CheckTransactionFlags{
		Network:        network,
		NodeAddress:    "",
		Wallet:         walletName,
		PubKey:         pubKey,
		Retries:        10,
		LogLevel:       "debug",
		PassphraseFile: passphraseFilePath,
		RawTransaction: testTransactionJSON,
	}

	expectedReq := &api.AdminCheckTransactionParams{
		Network:     network,
		NodeAddress: "",
		Wallet:      walletName,
		PublicKey:   pubKey,
		Retries:     10,
		Transaction: testTransaction(t),
	}

	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedPassphrase, passphrase)
	expectedJSON, _ := json.Marshal(expectedReq)
	actualJSON, _ := json.Marshal(req)
	assert.Equal(t, expectedJSON, actualJSON)
}

func testCheckCommandFlagsMissingWalletFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newCheckCommandFlags(t, testDir)
	f.Wallet = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func testCheckCommandFlagsMissingLogLevelFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newCheckCommandFlags(t, testDir)
	f.LogLevel = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("level"))
	assert.Empty(t, req)
}

func testCheckCommandFlagsUnsupportedLogLevelFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newCheckCommandFlags(t, testDir)
	f.LogLevel = vgrand.RandomStr(5)

	// when
	req, _, err := f.Validate()

	// then
	assert.EqualError(t, err, fmt.Sprintf("unsupported log level %q, supported levels: debug, info, warn, error", f.LogLevel))
	assert.Empty(t, req)
}

// Do we need this?
func testCheckCommandFlagsMissingNetworkAndNodeAddressFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newCheckCommandFlags(t, testDir)
	f.Network = ""
	f.NodeAddress = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.OneOfFlagsMustBeSpecifiedError("network", "node-address"))
	assert.Empty(t, req)
}

func testCheckCommandFlagsBothNetworkAndNodeAddressSpecifiedFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newCheckCommandFlags(t, testDir)
	f.Network = vgrand.RandomStr(10)
	f.NodeAddress = vgrand.RandomStr(10)

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MutuallyExclusiveError("network", "node-address"))
	assert.Empty(t, req)
}

func testCheckCommandFlagsMissingPubKeyFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newCheckCommandFlags(t, testDir)
	f.PubKey = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("pubkey"))
	assert.Empty(t, req)
}

func testCheckCommandFlagsMissingRequestFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newCheckCommandFlags(t, testDir)
	f.RawTransaction = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.ArgMustBeSpecifiedError("transaction"))
	assert.Empty(t, req)
}

func testCheckCommandFlagsMalformedRequestFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newCheckCommandFlags(t, testDir)
	f.RawTransaction = vgrand.RandomStr(5)

	// when
	req, _, err := f.Validate()

	// then
	assert.Error(t, err)
	assert.Empty(t, req)
}

func newCheckCommandFlags(t *testing.T, testDir string) *cmd.CheckTransactionFlags {
	t.Helper()

	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	networkName := vgrand.RandomStr(10)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	return &cmd.CheckTransactionFlags{
		Network:        networkName,
		NodeAddress:    "",
		Retries:        10,
		LogLevel:       "debug",
		RawTransaction: `{"voteSubmission": {"proposalId": "some-id", "value": "VALUE_YES"}}`,
		Wallet:         walletName,
		PubKey:         pubKey,
		PassphraseFile: passphraseFilePath,
	}
}
