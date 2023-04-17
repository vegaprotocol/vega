package cmd_test

import (
	"encoding/json"
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignCommandFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testSignCommandFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testSignCommandFlagsMissingWalletFails)
	t.Run("Missing chain ID fails", testSignCommandFlagsMissingChainIDFails)
	t.Run("Missing public key fails", testSignCommandFlagsMissingPubKeyFails)
	t.Run("Missing tx height fails", testSignCommandFlagsMissingTxBlockHeightFails)
	t.Run("Missing tx height fails", testSignCommandFlagsMissingTxBlockHashFails)
	t.Run("Missing pow difficulty fails", testSignCommandFlagsMissingPoWDifficultyFails)
	t.Run("Missing request fails", testSignCommandFlagsMissingRequestFails)
	t.Run("Network and PoW mutually exclusive", testSignCommandFlagsNetworkPoWMutuallyExclusive)
}

func testSignCommandFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	f := &cmd.SignTransactionFlags{
		Wallet:         walletName,
		PubKey:         pubKey,
		PassphraseFile: passphraseFilePath,
		Network:        "fairground",
		RawTransaction: `{"voteSubmission": {"proposalId": "ec066610abbd1736b69cadcb059b9efdfdd9e3e33560fc46b2b8b62764edf33f", "value": "VALUE_YES"}}`,
	}

	expectedTx := make(map[string]any)
	assert.NoError(t, json.Unmarshal([]byte(f.RawTransaction), &expectedTx))

	expectedReq := api.AdminSignTransactionParams{
		Wallet:      walletName,
		Network:     "fairground",
		PublicKey:   pubKey,
		Transaction: expectedTx,
	}

	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, expectedReq, req)
	assert.Equal(t, expectedPassphrase, passphrase)
}

func testSignCommandFlagsMissingWalletFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.Wallet = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func testSignCommandFlagsMissingChainIDFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.ChainID = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("chain-id"))
	assert.Empty(t, req)
}

func testSignCommandFlagsMissingPubKeyFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.PubKey = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("pubkey"))
	assert.Empty(t, req)
}

func testSignCommandFlagsMissingTxBlockHeightFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.TxBlockHeight = 0

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("tx-height"))
	assert.Empty(t, req)
}

func testSignCommandFlagsMissingTxBlockHashFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.TxBlockHash = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("tx-block-hash"))
	assert.Empty(t, req)
}

func testSignCommandFlagsMissingPoWDifficultyFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.PowDifficulty = 0

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("pow-difficulty"))
	assert.Empty(t, req)
}

func testSignCommandFlagsNetworkPoWMutuallyExclusive(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.Network = "fairground"

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MutuallyExclusiveError("network", "tx-height"))
	assert.Empty(t, req)
}

func testSignCommandFlagsMissingRequestFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.RawTransaction = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.ArgMustBeSpecifiedError("transaction"))
	assert.Empty(t, req)
}

func newSignCommandFlags(t *testing.T, testDir string) *cmd.SignTransactionFlags {
	t.Helper()

	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	return &cmd.SignTransactionFlags{
		RawTransaction: `{"voteSubmission": {"proposalId": "some-id", "value": "VALUE_YES"}}`,
		Wallet:         walletName,
		PubKey:         pubKey,
		TxBlockHeight:  150,
		ChainID:        vgrand.RandomStr(5),
		TxBlockHash:    "hashhash",
		PowDifficulty:  12,
		PassphraseFile: passphraseFilePath,
	}
}
