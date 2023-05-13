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

func TestRotateKeyFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testRotateKeyFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testRotateKeyFlagsMissingWalletFails)
	t.Run("Missing chain ID fails", testRotateKeyFlagsMissingChainIDFails)
	t.Run("Missing new public key fails", testRotateKeyFlagsMissingNewPublicKeyFails)
	t.Run("Missing current public key fails", testRotateKeyFlagsMissingCurrentPublicKeyFails)
	t.Run("Missing tx height fails", testRotateKeyFlagsMissingTxBlockHeightFails)
	t.Run("Missing target height fails", testRotateKeyFlagsMissingTargetBlockHeightFails)
	t.Run("Validate fails when target height is less then tx height", testRotateKeyFlagsTargetFailsWhenBlockHeightIsLessThanTXHeight)
}

func testRotateKeyFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	currentPubKey := vgrand.RandomStr(20)
	pubKey := vgrand.RandomStr(20)
	submissionBlockHeight := uint64(20)
	enactmentBlockHeight := uint64(25)
	chainID := vgrand.RandomStr(5)

	f := &cmd.RotateKeyFlags{
		Wallet:                walletName,
		PassphraseFile:        passphraseFilePath,
		FromPublicKey:         currentPubKey,
		ChainID:               chainID,
		ToPublicKey:           pubKey,
		SubmissionBlockHeight: submissionBlockHeight,
		EnactmentBlockHeight:  enactmentBlockHeight,
	}

	expectedReq := api.AdminRotateKeyParams{
		Wallet:                walletName,
		FromPublicKey:         currentPubKey,
		ChainID:               chainID,
		ToPublicKey:           pubKey,
		SubmissionBlockHeight: submissionBlockHeight,
		EnactmentBlockHeight:  enactmentBlockHeight,
	}

	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedReq, req)
	assert.Equal(t, expectedPassphrase, passphrase)
}

func testRotateKeyFlagsMissingWalletFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newRotateKeyFlags(t, testDir)
	f.Wallet = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func testRotateKeyFlagsMissingChainIDFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newRotateKeyFlags(t, testDir)
	f.ChainID = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("chain-id"))
	assert.Empty(t, req)
}

func testRotateKeyFlagsMissingTxBlockHeightFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newRotateKeyFlags(t, testDir)
	f.SubmissionBlockHeight = 0

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("tx-height"))
	assert.Empty(t, req)
}

func testRotateKeyFlagsMissingTargetBlockHeightFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newRotateKeyFlags(t, testDir)
	f.EnactmentBlockHeight = 0

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("target-height"))
	assert.Empty(t, req)
}

func testRotateKeyFlagsMissingNewPublicKeyFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newRotateKeyFlags(t, testDir)
	f.ToPublicKey = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("new-pubkey"))
	assert.Empty(t, req)
}

func testRotateKeyFlagsMissingCurrentPublicKeyFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newRotateKeyFlags(t, testDir)
	f.FromPublicKey = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("current-pubkey"))
	assert.Empty(t, req)
}

func testRotateKeyFlagsTargetFailsWhenBlockHeightIsLessThanTXHeight(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newRotateKeyFlags(t, testDir)
	f.SubmissionBlockHeight = 25
	f.EnactmentBlockHeight = 20

	// when
	req, _, err := f.Validate()

	// then
	assert.EqualError(t, err, "--target-height flag must be greater than --tx-height")
	assert.Empty(t, req)
}

func newRotateKeyFlags(t *testing.T, testDir string) *cmd.RotateKeyFlags {
	t.Helper()

	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)
	currentPubKey := vgrand.RandomStr(20)
	chainID := vgrand.RandomStr(5)
	txBlockHeight := uint64(20)
	targetBlockHeight := uint64(25)

	return &cmd.RotateKeyFlags{
		Wallet:                walletName,
		ToPublicKey:           pubKey,
		FromPublicKey:         currentPubKey,
		PassphraseFile:        passphraseFilePath,
		SubmissionBlockHeight: txBlockHeight,
		ChainID:               chainID,
		EnactmentBlockHeight:  targetBlockHeight,
	}
}
