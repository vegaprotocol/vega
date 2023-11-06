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

const recoveryPhrase = "swing ceiling chaos green put insane ripple desk match tip melt usual shrug turkey renew icon parade veteran lens govern path rough page render"

func TestImportWalletFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testImportWalletFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testImportWalletFlagsMissingWalletFails)
	t.Run("Missing recovery phrase file fails", testImportWalletFlagsMissingRecoveryPhraseFileFails)
}

func testImportWalletFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	passphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	recoveryPhraseFilePath := NewFile(t, testDir, "recovery-phrase.txt", recoveryPhrase)
	walletName := vgrand.RandomStr(10)

	f := &cmd.ImportWalletFlags{
		Wallet:             walletName,
		RecoveryPhraseFile: recoveryPhraseFilePath,
		PassphraseFile:     passphraseFilePath,
	}

	expectedReq := api.AdminImportWalletParams{
		Wallet:         walletName,
		RecoveryPhrase: recoveryPhrase,
		Passphrase:     passphrase,
	}

	// when
	req, err := f.Validate()

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedReq, req)
}

func testImportWalletFlagsMissingWalletFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newImportWalletFlags(t, testDir)
	f.Wallet = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func testImportWalletFlagsMissingRecoveryPhraseFileFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newImportWalletFlags(t, testDir)
	f.RecoveryPhraseFile = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("recovery-phrase-file"))
	assert.Empty(t, req)
}

func newImportWalletFlags(t *testing.T, testDir string) *cmd.ImportWalletFlags {
	t.Helper()

	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	NewFile(t, testDir, "recovery-phrase.txt", recoveryPhrase)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	return &cmd.ImportWalletFlags{
		Wallet:             walletName,
		RecoveryPhraseFile: pubKey,
		PassphraseFile:     passphraseFilePath,
	}
}
