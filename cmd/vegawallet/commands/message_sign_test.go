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
	"encoding/base64"
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignMessageFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testSignMessageFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testSignMessageFlagsMissingWalletFails)
	t.Run("Missing public key fails", testSignMessageFlagsMissingPubKeyFails)
	t.Run("Missing message fails", testSignMessageFlagsMissingMessageFails)
	t.Run("Malformed message fails", testSignMessageFlagsMalformedMessageFails)
}

func testSignMessageFlagsValidFlagsSucceeds(t *testing.T) {
	testDir := t.TempDir()

	// given
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)
	encodedMessage := vgrand.RandomStr(20)

	f := &cmd.SignMessageFlags{
		Wallet:         walletName,
		PubKey:         pubKey,
		Message:        encodedMessage,
		PassphraseFile: passphraseFilePath,
	}

	expectedReq := api.AdminSignMessageParams{
		Wallet:         walletName,
		PublicKey:      pubKey,
		EncodedMessage: encodedMessage,
	}

	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
	assert.Equal(t, expectedPassphrase, passphrase)
}

func testSignMessageFlagsMissingWalletFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignMessageFlags(t, testDir)
	f.Wallet = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("wallet"))
	assert.Empty(t, req)
}

func testSignMessageFlagsMissingPubKeyFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignMessageFlags(t, testDir)
	f.PubKey = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("pubkey"))
	assert.Empty(t, req)
}

func testSignMessageFlagsMissingMessageFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignMessageFlags(t, testDir)
	f.Message = ""

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("message"))
	assert.Empty(t, req)
}

func testSignMessageFlagsMalformedMessageFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newSignMessageFlags(t, testDir)
	f.Message = vgrand.RandomStr(5)

	// when
	req, _, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBase64EncodedError("message"))
	assert.Empty(t, req)
}

func newSignMessageFlags(t *testing.T, testDir string) *cmd.SignMessageFlags {
	t.Helper()

	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	walletName := vgrand.RandomStr(10)
	pubKey := vgrand.RandomStr(20)
	decodedMessage := []byte(vgrand.RandomStr(20))

	return &cmd.SignMessageFlags{
		Wallet:         walletName,
		PubKey:         pubKey,
		Message:        base64.StdEncoding.EncodeToString(decodedMessage),
		PassphraseFile: passphraseFilePath,
	}
}
