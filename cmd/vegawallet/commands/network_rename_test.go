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

func TestRenameNetworkFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testRenameNetworkFlagsValidFlagsSucceeds)
	t.Run("Missing wallet fails", testRenameNetworkFlagsMissingNetworkFails)
	t.Run("Missing new name fails", testRenameNetworkFlagsMissingNewNameFails)
}

func testRenameNetworkFlagsValidFlagsSucceeds(t *testing.T) {
	// given
	walletName := vgrand.RandomStr(10)
	newName := vgrand.RandomStr(10)
	f := &cmd.RenameNetworkFlags{
		Network: walletName,
		NewName: newName,
	}

	expectedReq := api.AdminRenameNetworkParams{
		Network: walletName,
		NewName: newName,
	}

	// when
	req, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
}

func testRenameNetworkFlagsMissingNetworkFails(t *testing.T) {
	// given
	f := newRenameNetworkFlags(t)
	f.Network = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("network"))
	assert.Empty(t, req)
}

func testRenameNetworkFlagsMissingNewNameFails(t *testing.T) {
	// given
	f := newRenameNetworkFlags(t)
	f.NewName = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("new-name"))
	assert.Empty(t, req)
}

func newRenameNetworkFlags(t *testing.T) *cmd.RenameNetworkFlags {
	t.Helper()
	return &cmd.RenameNetworkFlags{
		Network: vgrand.RandomStr(10),
		NewName: vgrand.RandomStr(10),
	}
}
