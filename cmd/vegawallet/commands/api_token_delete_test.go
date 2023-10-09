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
	"github.com/stretchr/testify/require"
)

func TestDeleteAPITokenFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testDeleteAPITokenFlagsValidFlagsSucceeds)
	t.Run("Missing flags fails", testDeleteAPITokenFlagsMissingFlagsFails)
}

func testDeleteAPITokenFlagsValidFlagsSucceeds(t *testing.T) {
	// given
	testDir := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, testDir)
	f := &cmd.DeleteAPITokenFlags{
		Token:          vgrand.RandomStr(10),
		PassphraseFile: passphraseFilePath,
		Force:          true,
	}

	// when
	err := f.Validate()

	// then
	require.NoError(t, err)
}

func testDeleteAPITokenFlagsMissingFlagsFails(t *testing.T) {
	testDir := t.TempDir()
	_, passphraseFilePath := NewPassphraseFile(t, testDir)

	tcs := []struct {
		name        string
		flags       *cmd.DeleteAPITokenFlags
		missingFlag string
	}{
		{
			name: "without token",
			flags: &cmd.DeleteAPITokenFlags{
				PassphraseFile: passphraseFilePath,
			},
			missingFlag: "token",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// when
			err := tc.flags.Validate()

			// then
			assert.ErrorIs(t, err, flags.MustBeSpecifiedError(tc.missingFlag))
		})
	}
}
