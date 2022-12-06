package cmd_test

import (
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteNetworkFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testDeleteNetworkFlagsValidFlagsSucceeds)
	t.Run("Missing network fails", testDeleteNetworkFlagsMissingNetworkFails)
}

func testDeleteNetworkFlagsValidFlagsSucceeds(t *testing.T) {
	// given
	f := &cmd.DeleteNetworkFlags{
		Network: vgrand.RandomStr(10),
		Force:   true,
	}

	// when
	req, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, f.Network, req.Name)
}

func testDeleteNetworkFlagsMissingNetworkFails(t *testing.T) {
	// given
	f := newDeleteNetworkFlags(t)
	f.Network = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("network"))
	assert.Empty(t, req)
}

func newDeleteNetworkFlags(t *testing.T) *cmd.DeleteNetworkFlags {
	t.Helper()

	return &cmd.DeleteNetworkFlags{
		Network: vgrand.RandomStr(10),
	}
}
