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
	"code.vegaprotocol.io/vega/wallet/wallet"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnotateKeyFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testAnnotateKeyFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testAnnotateKeyFlagsMissingWalletFails)
	t.Run("Missing public key fails", testAnnotateKeyFlagsMissingPubKeyFails)
	t.Run("Missing metadata fails", testAnnotateKeyFlagsMissingMetadataAndClearFails)
	t.Run("Clearing with metadata fails", testAnnotateKeyFlagsClearingWithMetadataFails)
	t.Run("Invalid metadata fails", testAnnotateKeyFlagsInvalidMetadataFails)
}

func testAnnotateKeyFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	f := &cmd.AnnotateKeyFlags{
		Wallet:         walletName,
		PubKey:         pubKey,
		PassphraseFile: passphraseFilePath,
		RawMetadata:    []string{"name:my-wallet", "role:validation"},
		Clear:          false,
	}

	expectedReq := api.AdminAnnotateKeyParams{
		Wallet:    walletName,
		PublicKey: pubKey,
		Metadata: []wallet.Metadata{
			{Key: "name", Value: "my-wallet"},
			{Key: "role", Value: "validation"},
		},
	}

	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
	assert.Equal(t, expectedPassphrase, passphrase)
}

func testAnnotateKeyFlagsMissingWalletFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newAnnotateKeyFlags(t, testDir)
	f.Wallet = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func testAnnotateKeyFlagsMissingPubKeyFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newAnnotateKeyFlags(t, testDir)
	f.PubKey = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("pubkey"))
	assert.Empty(t, req)
}

func testAnnotateKeyFlagsMissingMetadataAndClearFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newAnnotateKeyFlags(t, testDir)
	f.RawMetadata = []string{}

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.OneOfFlagsMustBeSpecifiedError("meta", "clear"))
	assert.Empty(t, req)
}

func testAnnotateKeyFlagsClearingWithMetadataFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newAnnotateKeyFlags(t, testDir)
	f.Clear = true

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MutuallyExclusiveError("meta", "clear"))
	assert.Empty(t, req)
}

func testAnnotateKeyFlagsInvalidMetadataFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newAnnotateKeyFlags(t, testDir)
	f.RawMetadata = []string{"is=invalid"}

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.InvalidFlagFormatError("meta"))
	assert.Empty(t, req)
}

func newAnnotateKeyFlags(t *testing.T, testDir string) *cmd.AnnotateKeyFlags {
	t.Helper()

	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)

	return &cmd.AnnotateKeyFlags{
		Wallet:         walletName,
		PubKey:         pubKey,
		PassphraseFile: passphraseFilePath,
		RawMetadata:    []string{"name:my-wallet", "role:validation"},
		Clear:          false,
	}
}
