package cmd_test

import (
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPurgePermissionsFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testPurgePermissionsFlagsValidFlagsSucceeds)
	t.Run("Missing flags fails", testPurgePermissionsFlagsMissingFlagsFails)
}

func testPurgePermissionsFlagsValidFlagsSucceeds(t *testing.T) {
	// given
	testDir := t.TempDir()
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	f := &cmd.PurgePermissionsFlags{
		Wallet:         vgrand.RandomStr(10),
		PassphraseFile: passphraseFilePath,
		Force:          true,
	}

	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	assert.Equal(t, f.Wallet, req.Wallet)
	assert.Equal(t, expectedPassphrase, passphrase)
}

func testPurgePermissionsFlagsMissingFlagsFails(t *testing.T) {
	testDir := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, testDir)

	tcs := []struct {
		name        string
		flags       *cmd.PurgePermissionsFlags
		missingFlag string
	}{
		{
			name: "without wallet",
			flags: &cmd.PurgePermissionsFlags{
				Wallet:         "",
				PassphraseFile: passphraseFilePath,
			},
			missingFlag: "wallet",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// when
			req, _, err := tc.flags.Validate()

			// then
			assert.ErrorIs(t, err, flags.MustBeSpecifiedError(tc.missingFlag))
			require.Empty(t, req)
		})
	}
}
