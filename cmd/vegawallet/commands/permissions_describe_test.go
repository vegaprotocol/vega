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

func TestDescribePermissionsFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testDescribePermissionsValidFlagsSucceeds)
	t.Run("Missing flags fails", testDescribePermissionsWithMissingFlagsFails)
}

func testDescribePermissionsValidFlagsSucceeds(t *testing.T) {
	// given
	testDir := t.TempDir()
	walletName := vgrand.RandomStr(10)
	hostname := vgrand.RandomStr(10)
	expectedPassphrase, passphraseFilePath := NewPassphraseFile(t, testDir)

	f := &cmd.DescribePermissionsFlags{
		Wallet:         walletName,
		Hostname:       hostname,
		PassphraseFile: passphraseFilePath,
	}

	expectedReq := api.AdminDescribePermissionsParams{
		Wallet:   walletName,
		Hostname: hostname,
	}
	// when
	req, passphrase, err := f.Validate()

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedReq, req)
	assert.Equal(t, expectedPassphrase, passphrase)
}

func testDescribePermissionsWithMissingFlagsFails(t *testing.T) {
	testDir := t.TempDir()
	walletName := vgrand.RandomStr(10)
	hostname := vgrand.RandomStr(10)
	_, passphraseFilePath := NewPassphraseFile(t, testDir)

	tcs := []struct {
		name        string
		flags       *cmd.DescribePermissionsFlags
		missingFlag string
	}{
		{
			name: "without hostname",
			flags: &cmd.DescribePermissionsFlags{
				Wallet:         walletName,
				Hostname:       "",
				PassphraseFile: passphraseFilePath,
			},
			missingFlag: "hostname",
		}, {
			name: "without wallet",
			flags: &cmd.DescribePermissionsFlags{
				Wallet:         "",
				Hostname:       hostname,
				PassphraseFile: passphraseFilePath,
			},
			missingFlag: "wallet",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// when
			req, _, err := tc.flags.Validate()

			// then
			assert.ErrorIs(t, err, flags.MustBeSpecifiedError(tc.missingFlag))
			assert.Empty(t, req)
		})
	}
}
