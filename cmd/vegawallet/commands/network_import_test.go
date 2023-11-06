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

func TestImportNetworkFlags(t *testing.T) {
	t.Run("Valid flags with URL succeeds", testImportNetworkFlagsValidFlagsWithURLSucceeds)
	t.Run("Valid flags with file path succeeds", testImportNetworkFlagsValidFlagsWithFilePathSucceeds)
	t.Run("Missing URL and file path fails", testImportNetworkFlagsMissingURLAndFilePathFails)
	t.Run("Both URL and filePath specified", testImportNetworkFlagsBothURLAndFilePathSpecifiedFails)
}

func testImportNetworkFlagsValidFlagsWithURLSucceeds(t *testing.T) {
	// given
	networkName := vgrand.RandomStr(10)
	url := vgrand.RandomStr(20)

	f := &cmd.ImportNetworkFlags{
		Name:  networkName,
		URL:   url,
		Force: true,
	}

	expectedReq := api.AdminImportNetworkParams{
		Name:      networkName,
		URL:       url,
		Overwrite: true,
	}

	// when
	req, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
}

func testImportNetworkFlagsValidFlagsWithFilePathSucceeds(t *testing.T) {
	// given
	networkName := vgrand.RandomStr(10)
	filePath := vgrand.RandomStr(20)

	f := &cmd.ImportNetworkFlags{
		Name:     networkName,
		FilePath: filePath,
		Force:    true,
	}

	expectedReq := api.AdminImportNetworkParams{
		Name:      networkName,
		URL:       api.FileSchemePrefix + filePath,
		Overwrite: true,
	}

	// when
	req, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
}

func testImportNetworkFlagsMissingURLAndFilePathFails(t *testing.T) {
	// given
	f := newImportNetworkFlags(t)
	f.URL = ""
	f.FilePath = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.OneOfFlagsMustBeSpecifiedError("from-file", "from-url"))
	assert.Empty(t, req)
}

func testImportNetworkFlagsBothURLAndFilePathSpecifiedFails(t *testing.T) {
	// given
	f := newImportNetworkFlags(t)
	f.URL = vgrand.RandomStr(20)
	f.FilePath = vgrand.RandomStr(20)

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MutuallyExclusiveError("from-file", "from-url"))
	assert.Empty(t, req)
}

func newImportNetworkFlags(t *testing.T) *cmd.ImportNetworkFlags {
	t.Helper()

	networkName := vgrand.RandomStr(10)

	return &cmd.ImportNetworkFlags{
		Name:  networkName,
		Force: true,
	}
}
