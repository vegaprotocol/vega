package cmd_test

import (
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"github.com/stretchr/testify/require"
)

func TestAdminListAPITokensFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testAdminListAPITokensValidFlagsSucceeds)
}

func testAdminListAPITokensValidFlagsSucceeds(t *testing.T) {
	// given
	testDir := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, testDir)

	f := &cmd.ListAPITokensFlags{
		PassphraseFile: passphraseFilePath,
	}

	// when
	err := f.Validate()

	// then
	require.NoError(t, err)
}
