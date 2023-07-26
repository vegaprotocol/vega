package cmd_test

import (
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"github.com/stretchr/testify/require"
)

func TestInitAPIToken(t *testing.T) {
	t.Run("Initialising software succeeds", testInitialisingAPITokenSucceeds)
	t.Run("Forcing software initialisation succeeds", testForcingAPITokenInitialisationSucceeds)
}

func testInitialisingAPITokenSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	f := &cmd.InitAPITokenFlags{
		Force:          false,
		PassphraseFile: passphraseFilePath,
	}

	// when
	_, err := cmd.InitAPIToken(testDir, f)

	// then
	require.NoError(t, err)
}

func testForcingAPITokenInitialisationSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	f := &cmd.InitAPITokenFlags{
		Force:          false,
		PassphraseFile: passphraseFilePath,
	}

	// when
	_, err := cmd.InitAPIToken(testDir, f)

	// then
	require.NoError(t, err)

	// given
	f = &cmd.InitAPITokenFlags{
		Force:          true,
		PassphraseFile: passphraseFilePath,
	}

	// when
	_, err = cmd.InitAPIToken(testDir, f)

	// then
	require.NoError(t, err)
}
