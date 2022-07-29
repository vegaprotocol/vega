package cmd_test

import (
	"encoding/json"
	"testing"

	"code.vegaprotocol.io/protos/vega"
	v1 "code.vegaprotocol.io/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/protos/vega/wallet/v1"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	wcommands "code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignCommandFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testSignCommandFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testSignCommandFlagsMissingWalletFails)
	t.Run("Missing public key fails", testSignCommandFlagsMissingPubKeyFails)
	t.Run("Missing tx height fails", testSignCommandFlagsMissingTxBlockHeightFails)
	t.Run("Missing request fails", testSignCommandFlagsMissingRequestFails)
	t.Run("Malformed request fails", testSignCommandFlagsMalformedRequestFails)
	t.Run("Invalid request fails", testSignCommandFlagsInvalidRequestFails)
	t.Run("Request with public key set in it fails", testSignCommandFlagsRequestWithPubKeyFails)
}

func testSignCommandFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	passphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	f := &cmd.SignCommandFlags{
		Wallet:         walletName,
		PubKey:         pubKey,
		PassphraseFile: passphraseFilePath,
		TxBlockHeight:  150,
		RawCommand:     `{"voteSubmission": {"proposalId": "ec066610abbd1736b69cadcb059b9efdfdd9e3e33560fc46b2b8b62764edf33f", "value": "VALUE_YES"}}`,
	}

	expectedReq := &wcommands.SignCommandRequest{
		Wallet:        walletName,
		Passphrase:    passphrase,
		TxBlockHeight: 150,
		Request: &walletpb.SubmitTransactionRequest{
			PubKey:    pubKey,
			Propagate: true,
			Command: &walletpb.SubmitTransactionRequest_VoteSubmission{
				VoteSubmission: &v1.VoteSubmission{
					ProposalId: "ec066610abbd1736b69cadcb059b9efdfdd9e3e33560fc46b2b8b62764edf33f",
					Value:      vega.Vote_VALUE_YES,
				},
			},
		},
	}

	// when
	req, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)

	expectedJSON, _ := json.Marshal(expectedReq)
	actualJSON, _ := json.Marshal(req)
	assert.Equal(t, expectedJSON, actualJSON)
}

func testSignCommandFlagsMissingWalletFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.Wallet = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.FlagMustBeSpecifiedError("wallet"))
	assert.Nil(t, req)
}

func testSignCommandFlagsMissingPubKeyFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.PubKey = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.FlagMustBeSpecifiedError("pubkey"))
	assert.Nil(t, req)
}

func testSignCommandFlagsMissingTxBlockHeightFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newRotateKeyFlags(t, testDir)
	f.TxBlockHeight = 0

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.FlagMustBeSpecifiedError("tx-height"))
	assert.Nil(t, req)
}

func testSignCommandFlagsMissingRequestFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.RawCommand = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.ArgMustBeSpecifiedError("command"))
	assert.Nil(t, req)
}

func testSignCommandFlagsMalformedRequestFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.RawCommand = vgrand.RandomStr(5)

	// when
	req, err := f.Validate()

	// then
	assert.Error(t, err)
	assert.Nil(t, req)
}

func testSignCommandFlagsInvalidRequestFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.RawCommand = `{"voteSubmission": {}}`

	// when
	req, err := f.Validate()

	// then
	assert.Error(t, err)
	assert.Nil(t, req)
}

func testSignCommandFlagsRequestWithPubKeyFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignCommandFlags(t, testDir)
	f.RawCommand = `{"pubKey": "qwerty123456", "voteSubmission": {"proposalId": "some-id", "value": "VALUE_YES"}}`

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, cmd.ErrDoNotSetPubKeyInCommand)
	assert.Nil(t, req)
}

func newSignCommandFlags(t *testing.T, testDir string) *cmd.SignCommandFlags {
	t.Helper()

	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	return &cmd.SignCommandFlags{
		RawCommand:     `{"voteSubmission": {"proposalId": "some-id", "value": "VALUE_YES"}}`,
		Wallet:         walletName,
		PubKey:         pubKey,
		TxBlockHeight:  150,
		PassphraseFile: passphraseFilePath,
	}
}
