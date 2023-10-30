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

func TestDescribeNetworkFlags(t *testing.T) {
	t.Run("Valid flags with Network succeeds", testDescribeNetworkValidFlagsWithNetworkSucceeds)
	t.Run("Invalid flags without Network fails", testDescribeNetworkInvalidFlagsWithoutNetworkFails)
}

func testDescribeNetworkValidFlagsWithNetworkSucceeds(t *testing.T) {
	// given
	networkName := vgrand.RandomStr(10)

	f := &cmd.DescribeNetworkFlags{
		Network: networkName,
	}

	expectedReq := api.AdminDescribeNetworkParams{
		Name: networkName,
	}
	// when
	req, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
}

func testDescribeNetworkInvalidFlagsWithoutNetworkFails(t *testing.T) {
	// given
	f := &cmd.DescribeNetworkFlags{}

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("network"))
	require.Empty(t, req)
}
