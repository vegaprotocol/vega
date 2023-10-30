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

func TestCreateWalletFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testCreateWalletFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testCreateWalletFlagsMissingWalletFails)
}

func testCreateWalletFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	walletName := vgrand.RandomStr(10)
	passphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	f := &cmd.CreateWalletFlags{
		Wallet:         walletName,
		PassphraseFile: passphraseFilePath,
	}

	expectedReq := api.AdminCreateWalletParams{
		Wallet:     walletName,
		Passphrase: passphrase,
	}

	// when
	req, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
}

func testCreateWalletFlagsMissingWalletFails(t *testing.T) {
	// given
	f := newCreateWalletFlags(t)
	f.Wallet = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func newCreateWalletFlags(t *testing.T) *cmd.CreateWalletFlags {
	t.Helper()
	return &cmd.CreateWalletFlags{
		Wallet:         vgrand.RandomStr(10),
		PassphraseFile: "/some/fake/path",
	}
}
