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

func TestUpdatePassphrase(t *testing.T) {
	t.Run("Valid flags succeeds", testUpdatePassphraseFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testUpdatePassphraseFlagsMissingWalletFails)
}

func testUpdatePassphraseFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	isolatedPassphrase, isolatedPassphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)

	f := &cmd.UpdatePassphraseFlags{
		Wallet:            walletName,
		PassphraseFile:    passphraseFilePath,
		NewPassphraseFile: isolatedPassphraseFilePath,
	}

	expectedReq := api.AdminUpdatePassphraseParams{
		Wallet:        walletName,
		NewPassphrase: isolatedPassphrase,
	}

	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedReq, req)
	assert.Equal(t, expectedPassphrase, passphrase)
}

func testUpdatePassphraseFlagsMissingWalletFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newUpdatePassphraseFlags(t, testDir)
	f.Wallet = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func newUpdatePassphraseFlags(t *testing.T, testDir string) *cmd.UpdatePassphraseFlags {
	t.Helper()

	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	_, newPassphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)

	return &cmd.UpdatePassphraseFlags{
		Wallet:            walletName,
		NewPassphraseFile: newPassphraseFilePath,
		PassphraseFile:    passphraseFilePath,
	}
}
