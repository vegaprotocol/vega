package cmd_test

import (
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteAPITokenFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testDeleteAPITokenFlagsValidFlagsSucceeds)
	t.Run("Missing flags fails", testDeleteAPITokenFlagsMissingFlagsFails)
}

func testDeleteAPITokenFlagsValidFlagsSucceeds(t *testing.T) {
	// given
	testDir := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	f := &cmd.DeleteAPITokenFlags{
		Token:          vgrand.RandomStr(10),
		PassphraseFile: passphraseFilePath,
		Force:          true,
	}

	// when
	err := f.Validate()

	// then
	require.NoError(t, err)
}

func testDeleteAPITokenFlagsMissingFlagsFails(t *testing.T) {
	testDir := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, testDir)

	tcs := []struct {
		name        string
		flags       *cmd.DeleteAPITokenFlags
		missingFlag string
	}{
		{
			name: "without token",
			flags: &cmd.DeleteAPITokenFlags{
				PassphraseFile: passphraseFilePath,
			},
			missingFlag: "token",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// when
			err := tc.flags.Validate()

			// then
			assert.ErrorIs(t, err, flags.MustBeSpecifiedError(tc.missingFlag))
		})
	}
}
