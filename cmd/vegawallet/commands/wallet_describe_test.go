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

func TestGetWalletInfoFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testGetWalletInfoFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testGetWalletInfoFlagsMissingWalletFails)
}

func testGetWalletInfoFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)

	f := &cmd.DescribeWalletFlags{
		Wallet:         walletName,
		PassphraseFile: passphraseFilePath,
	}

	expectedReq := api.AdminDescribeWalletParams{
		Wallet: walletName,
	}

	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
	assert.Equal(t, expectedPassphrase, passphrase)
}

func testGetWalletInfoFlagsMissingWalletFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newGetWalletInfoFlags(t, testDir)
	f.Wallet = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func newGetWalletInfoFlags(t *testing.T, testDir string) *cmd.DescribeWalletFlags {
	t.Helper()

	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)

	return &cmd.DescribeWalletFlags{
		Wallet:         walletName,
		PassphraseFile: passphraseFilePath,
	}
}
