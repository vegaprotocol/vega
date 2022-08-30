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

func TestDeleteWalletFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testDeleteWalletFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testDeleteWalletFlagsMissingWalletFails)
}

func testDeleteWalletFlagsValidFlagsSucceeds(t *testing.T) {
	// given
	walletName := vgrand.RandomStr(10)

	f := &cmd.DeleteWalletFlags{
		Wallet: walletName,
		Force:  true,
	}

	// when
	params, err := f.Validate()

	// then
	require.NoError(t, err)
	assert.Equal(t, api.RemoveWalletParams{
		Wallet: walletName,
	}, params)
}

func testDeleteWalletFlagsMissingWalletFails(t *testing.T) {
	// given
	f := newDeleteWalletFlags(t)
	f.Wallet = ""

	// when
	params, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.FlagMustBeSpecifiedError("wallet"))
	assert.Empty(t, params)
}

func newDeleteWalletFlags(t *testing.T) *cmd.DeleteWalletFlags {
	t.Helper()

	walletName := vgrand.RandomStr(10)

	return &cmd.DeleteWalletFlags{
		Wallet: walletName,
	}
}
