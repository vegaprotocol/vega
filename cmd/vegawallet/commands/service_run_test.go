package cmd_test

import (
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunServiceFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testRunServiceFlagsValidFlagsSucceeds)
	t.Run("Missing network fails", testRunServiceFlagsMissingNetworkFails)
}

func testRunServiceFlagsValidFlagsSucceeds(t *testing.T) {
	// given
	networkName := vgrand.RandomStr(10)

	f := &cmd.RunServiceFlags{
		Network: networkName,
	}

	// when
	err := f.Validate()

	// then
	require.NoError(t, err)
}

func testRunServiceFlagsMissingNetworkFails(t *testing.T) {
	// given
	f := newRunServiceFlags(t)
	f.Network = ""

	// when
	err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("network"))
}

func newRunServiceFlags(t *testing.T) *cmd.RunServiceFlags {
	t.Helper()

	networkName := vgrand.RandomStr(10)

	return &cmd.RunServiceFlags{
		Network: networkName,
	}
}
