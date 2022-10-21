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

func TestRenameWalletFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testRenameWalletFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testRenameWalletFlagsMissingWalletFails)
	t.Run("Missing new name fails", testRenameWalletFlagsMissingNewNameFails)
}

func testRenameWalletFlagsValidFlagsSucceeds(t *testing.T) {
	// given
	walletName := vgrand.RandomStr(10)
	newName := vgrand.RandomStr(10)
	f := &cmd.RenameWalletFlags{
		Wallet:  walletName,
		NewName: newName,
	}

	expectedReq := api.AdminRenameWalletParams{
		Wallet:  walletName,
		NewName: newName,
	}

	// when
	req, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
}

func testRenameWalletFlagsMissingWalletFails(t *testing.T) {
	// given
	f := newRenameWalletFlags(t)
	f.Wallet = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func testRenameWalletFlagsMissingNewNameFails(t *testing.T) {
	// given
	f := newRenameWalletFlags(t)
	f.NewName = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("new-name"))
	assert.Empty(t, req)
}

func newRenameWalletFlags(t *testing.T) *cmd.RenameWalletFlags {
	t.Helper()
	return &cmd.RenameWalletFlags{
		Wallet:  vgrand.RandomStr(10),
		NewName: vgrand.RandomStr(10),
	}
}
