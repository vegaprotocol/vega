// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	assert.Equal(t, api.AdminRemoveWalletParams{
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
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, params)
}

func newDeleteWalletFlags(t *testing.T) *cmd.DeleteWalletFlags {
	t.Helper()

	walletName := vgrand.RandomStr(10)

	return &cmd.DeleteWalletFlags{
		Wallet: walletName,
	}
}
