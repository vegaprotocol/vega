package cmd_test

import (
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescribePermissionsFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testDescribePermissionsValidFlagsSucceeds)
	t.Run("Missing flags fails", testDescribePermissionsWithMissingFlagsFails)
}

func testDescribePermissionsValidFlagsSucceeds(t *testing.T) {
	// given
	testDir := t.TempDir()
	walletName := vgrand.RandomStr(10)
	hostname := vgrand.RandomStr(10)
	passphrase, passphraseFilePath := NewPassphraseFile(t, testDir)

	f := &cmd.DescribePermissionsFlags{
		Wallet:         walletName,
		Hostname:       hostname,
		PassphraseFile: passphraseFilePath,
	}

	expectedReq := &wallet.DescribePermissionsRequest{
		Wallet:     walletName,
		Hostname:   hostname,
		Passphrase: passphrase,
	}
	// when
	req, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
}

func testDescribePermissionsWithMissingFlagsFails(t *testing.T) {
	testDir := t.TempDir()
	walletName := vgrand.RandomStr(10)
	hostname := vgrand.RandomStr(10)
	_, passphraseFilePath := NewPassphraseFile(t, testDir)

	tcs := []struct {
		name        string
		flags       *cmd.DescribePermissionsFlags
		missingFlag string
	}{
		{
			name: "without hostname",
			flags: &cmd.DescribePermissionsFlags{
				Wallet:         walletName,
				Hostname:       "",
				PassphraseFile: passphraseFilePath,
			},
			missingFlag: "hostname",
		}, {
			name: "without wallet",
			flags: &cmd.DescribePermissionsFlags{
				Wallet:         "",
				Hostname:       hostname,
				PassphraseFile: passphraseFilePath,
			},
			missingFlag: "wallet",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// when
			req, err := tc.flags.Validate()

			// then
			assert.ErrorIs(t, err, flags.FlagMustBeSpecifiedError(tc.missingFlag))
			require.Nil(t, req)
		})
	}
}
