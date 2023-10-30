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
	"github.com/stretchr/testify/assert"
)

func TestRunServiceFlags(t *testing.T) {
	t.Run("Missing loads-token flag with tokens passphrase flag fails", testRunServiceFlagsTokenPassphraseWithoutWithLOngLivingTokenFails)
	t.Run("Missing network fails", testRunServiceFlagsMissingNetworkFails)
}

func testRunServiceFlagsTokenPassphraseWithoutWithLOngLivingTokenFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	networkName := vgrand.RandomStr(10)
	f := &cmd.RunServiceFlags{
		Network:              networkName,
		TokensPassphraseFile: passphraseFilePath,
	}

	// when
	err := f.Validate(&cmd.RootFlags{
		Home: testDir,
	})

	// then
	assert.ErrorIs(t, err, flags.OneOfParentsFlagMustBeSpecifiedError("tokens-passphrase-file", "load-tokens"))
}

func testRunServiceFlagsMissingNetworkFails(t *testing.T) {
	testDir := t.TempDir()

	// given
	f := newRunServiceFlags(t)
	f.Network = ""

	// when
	err := f.Validate(&cmd.RootFlags{
		Home: testDir,
	})

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
