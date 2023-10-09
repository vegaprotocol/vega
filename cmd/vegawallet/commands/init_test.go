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
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	t.Run("Initialising software succeeds", testInitialisingSoftwareSucceeds)
	t.Run("Forcing software initialisation succeeds", testForcingSoftwareInitialisationSucceeds)
}

func testInitialisingSoftwareSucceeds(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	// given
	f := &cmd.InitFlags{
		Force: false,
	}

	// when
	err := cmd.Init(testDir, f)

	// then
	require.NoError(t, err)
}

func testForcingSoftwareInitialisationSucceeds(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	// given
	f := &cmd.InitFlags{
		Force: false,
	}

	// when
	err := cmd.Init(testDir, f)

	// then
	require.NoError(t, err)

	// given
	f = &cmd.InitFlags{
		Force: true,
	}

	// when
	err = cmd.Init(testDir, f)

	// then
	require.NoError(t, err)
}
